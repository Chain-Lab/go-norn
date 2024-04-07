// Package node
// @Description: 连接管理器，作为一个集中化的实例管理其它 peer 传来的信息
package node

import (
	"context"
	"crypto/elliptic"
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/core"
	"github.com/chain-lab/go-norn/crypto"
	"github.com/chain-lab/go-norn/metrics"
	"github.com/chain-lab/go-norn/p2p"
	"github.com/chain-lab/go-norn/utils"
	"github.com/gookit/config/v2"
	lru "github.com/hashicorp/golang-lru"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"net"
	"sync"
	"time"
)

type msgHandler func(pm *P2PManager, msg *p2p.Message, p *Peer)

var handlerMap = map[p2p.StatusCode]msgHandler{
	p2p.StatusCodeStatusMsg:        handleStatusMsg,        // 状态消息，目前接收对端的高度信息
	p2p.StatusCodeBlockBodiesMsg:   handleBlockMsg,         // 对应上一个状态码，如果对端请求区块，在缓冲区中取出区块进行响应
	p2p.StatusCodeSyncStatusReq:    handleSyncStatusReq,    // 携带本地的高度信息，请求对端的状态信息，例如高度/缓冲区高度
	p2p.StatusCodeSyncStatusMsg:    handleSyncStatusMsg,    // 响应对端的状态请求
	p2p.StatusCodeSyncGetBlocksMsg: handleSyncGetBlocksMsg, // 根据高度请求区块
	p2p.StatusCodeSyncBlocksMsg:    handleSyncBlockMsg,     // 响应对应高度的区块
	p2p.StatusCodeTimeSyncReq:      handleTimeSyncReq,      // 时间同步请求
	p2p.StatusCodeTimeSyncRsp:      handleTimeSyncRsp,      // 时间同步响应
	//p2p.StatusCodeGetBlockBodiesMsg: handleGetBlockBodiesMsg, // 请求本地缓冲区中不存在的区块
	//p2p.StatusCodeNewBlockHashesMsg: handleNewBlockHashMsg,   // 广播新打包的区块哈希值，在同步旧区块（非缓冲区同步状态）时不处理
	//p2p.StatusCodeNewBlockMsg:       handleNewBlockMsg,       // 广播新打包的区块，在同步旧区块（非缓冲区同步状态）时不处理
}

var (
	managerInst *P2PManager = nil
)

const (
	// 节点重连限制，如果节点在 10 分钟内尝试连接过但是失败，跳过该节点
	retryInterval          = 10 * time.Minute
	streamLimit            = 20    // 用于 P2P 通信的流的数量限制
	maxKnownBlock          = 1024  // 最大已知区块的 LRU 缓存大小
	maxKnownTransaction    = 32768 // 最大已知交易的 LRU 缓存大小
	maxSyncerStatusChannel = 512   // 同步器 syncer 的最大状态 channel 限制

	packageBlockInterval = 2       // 区块打包间隔
	gossipNodes          = 8       // UDP Gossip 广播节点数量
	udpBufferSize        = 1024    // UDP 缓冲大小 = 1M
	pubsubMaxSize        = 1 << 22 // 4 MB

	NetworkRendezvous = "chronos"
	TxGossipTopic     = "/chronos/1.0.1/transactions"
	BlockGossipTopic  = "/chronos/1.0.1/blocks"
	ProtocolId        = protocol.ID("/chronos/1.0.0/p2p")
	TxProtocolId      = protocol.ID("/chronos/1.0.0/transaction")
)

// P2PManagerConfig P2PManager 的实例化配置信息
type P2PManagerConfig struct {
	TxPool       *core.TxPool     // 交易池实例
	Chain        *core.BlockChain // 区块链实例
	Genesis      bool             // 是否创世节点
	InitialDelta int64            // 初始时间偏移，仅仅用于进行时间同步测试
}

type P2PManager struct {
	id         peer.ID               // 本地 P2P 节点 id
	triedPeers map[peer.ID]time.Time // 尝试连接的节点和连接时间戳，在一定时间内不再进行尝试
	peerSet    []*Peer               // 已建立连接的节点列表
	peers      map[peer.ID]*Peer     // 节点 ID -> 节点对象

	gossip     *pubsub.PubSub // go-libp2p 的 gossip 广播实例
	blockTopic *pubsub.Topic  // 区块广播 topic
	txTopic    *pubsub.Topic  // 交易广播 topic

	blockBroadcastQueue chan *common.Block       // 区块广播队列
	txBroadcastQueue    chan *common.Transaction // 交易广播队列

	knownBlock       *lru.Cache // 本地已知的区块缓存
	knownTransaction *lru.Cache // 本地已知的交易缓存

	txPool *core.TxPool     // 交易池实例
	chain  *core.BlockChain // 区块链实例

	blockSyncer  *BlockSyncer // 区块同步实例
	timeSyncer   *TimeSyncer  // 时间同步实例
	startRoutine sync.Once

	peerSetLock sync.RWMutex // 节点管理锁
	genesis     bool         // 是否创世节点
}

func NewP2PManager(config *P2PManagerConfig) (*P2PManager, error) {
	// 创建新的区块缓存
	knownBlockCache, err := lru.New(maxKnownBlock)

	if err != nil {
		log.WithField("error", err).Debugln("Create known block cache failed.")
		return nil, err
	}

	// 创建新的交易缓存
	knownTxCache, err := lru.New(maxKnownTransaction)

	if err != nil {
		log.WithField("error", err).Debugln("Create known transaction cache failed.")
		return nil, err
	}

	// 区块同步配置
	blockSyncerConfig := &BlockSyncerConfig{
		Chain: config.Chain,
	}
	bs := NewBlockSyncer(blockSyncerConfig)
	ts := NewTimeSyncer(config.Genesis, config.InitialDelta)

	manager := &P2PManager{
		// todo: 限制节点数量，Kad 应该限制了节点数量不超过20个
		triedPeers: make(map[peer.ID]time.Time),
		peerSet:    make([]*Peer, 0, 40),
		peers:      make(map[peer.ID]*Peer),

		blockBroadcastQueue: make(chan *common.Block, 512),
		txBroadcastQueue:    make(chan *common.Transaction, 10240),

		knownBlock:       knownBlockCache,
		knownTransaction: knownTxCache,

		txPool: config.TxPool,
		chain:  config.Chain,

		blockSyncer: bs,
		timeSyncer:  ts,

		genesis: config.Genesis,
	}

	metrics.RoutineCreateCounterObserve(15)

	// 启动节点的打包交易协程
	go manager.packageBlockRoutine()

	// 启动区块同步器和时间同步器
	bs.Start()
	ts.Start()
	managerInst = manager

	return manager, nil
}

// GetBlockChain
//
//	@Description: 获取当前节点的区块链实例
//	@receiver pm
//	@return *core.BlockChain - 区块链实例
func (pm *P2PManager) GetBlockChain() *core.BlockChain {
	return pm.chain
}

// AddTransaction
//
//	@Description: 用于测试，添加交易到交易池
//	@receiver pm
//	@param tx - 交易实例
func (pm *P2PManager) AddTransaction(tx *common.Transaction) {
	txHash := hex.EncodeToString(tx.Body.Hash[:])
	pm.markTransaction(txHash)
	pm.txBroadcastQueue <- tx
	pm.txPool.Add(tx)
}

// GetP2PManager
//
//	@Description: 获取 Manager 实例
//	@return *P2PManager - Manager 实例
func GetP2PManager() *P2PManager {
	// todo: 这样的写法会有脏读的问题，也就是在分配地址后，对象可能还没有完全初始化
	return managerInst
}

// packageBlockRoutine
//
//	@Description: 交易打包实例
//	@receiver pm
func (pm *P2PManager) packageBlockRoutine() {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			timestamp := pm.timeSyncer.GetLogicClock()
			// 如果逻辑时间距离 2s 则进行区块的打包
			if (timestamp/1000)%packageBlockInterval != 0 {
				continue
			}

			//  如果未完成同步，则不进行打包
			if !pm.Synced() {
				log.Infoln("Waiting for node synced.")
				continue
			}

			// 获取最新区块
			latest, _ := pm.chain.GetLatestBlock()
			if latest == nil {
				log.Infoln("Waiting for genesis block.")
				continue
			}

			// 获取当前的 VDF 计算信息
			calc := crypto.GetCalculatorInstance()
			seed, pi := calc.GetSeedParams()
			log.Debugf("Get seed: %s", hex.EncodeToString(seed.Bytes()))

			// 判断当前节点是否为共识节点
			consensus, err := crypto.VRFCheckLocalConsensus(seed.Bytes())
			if !consensus || err != nil {
				//log.Infof("Local is not consensus node")
				continue
			}

			// VRF 计算得到的随机数，需要包含到区块中
			randNumber, s, t, err := crypto.VRFCalculate(elliptic.P256(), seed.Bytes())
			log.Infof("Package with seed: %s", hex.EncodeToString(seed.Bytes()))

			params := common.GeneralParams{
				Result:       seed.Bytes(),
				Proof:        pi.Bytes(),
				RandomNumber: [33]byte(randNumber),
				S:            s.Bytes(),
				T:            t.Bytes(),
			}

			// 交易池打包返回一个交易数组
			txs := pm.txPool.Package()
			log.Infof("Package %d txs.", len(txs))
			newBlock, err := pm.chain.PackageNewBlock(txs, timestamp, &params, packageBlockInterval)
			log.Infof("Package new block.")

			if err != nil {
				log.WithField("error", err).Warning("Package new block failed.")
				continue
			}

			pm.chain.AppendBlockTask(newBlock)
			log.Infoln("Append block to buffer.")
			pm.blockBroadcastQueue <- newBlock
			log.Infof("Package new block# 0x%s", hex.EncodeToString(newBlock.Header.BlockHash[:]))
		}
	}
}

// NewPeer
//
//	@Description: 创建一个新的 Peer，它基于底层发送消息的 p2p.Peer 实现
//	@receiver pm - Manager 实例
//	@param peerId - 节点的 p2p id信息
//	@param s - 消息读写流
//	@param remoteAddr - 对端的连接地址
//	@return *Peer - node.Peer 实例
//	@return error - 如果出错则返回报错
func (pm *P2PManager) NewPeer(peerId peer.ID, s *network.Stream,
	remoteAddr string) (*Peer,
	error) {
	cfg := PeerConfig{
		chain:          pm.chain,
		txPool:         pm.txPool,
		handler:        pm,
		remoteMultAddr: remoteAddr,
	}

	// 调用公有的 NewPeer 方法创建新节点
	p, err := NewPeer(peerId, s, cfg)

	if err != nil {
		log.WithField("error", err).Errorln("Create peer failed.")
		return nil, err
	}

	// 添加到 peerset 中，并且添加给同步器
	pm.peerSet = append(pm.peerSet, p)
	log.Infoln(p.addr)
	pm.peers[p.peerID] = p
	pm.blockSyncer.AddPeer(p)
	return p, err
}

// isKnownTransaction
//
//	@Description: 判断交易哈希为 hash 的交易是否在当前节点出现过
//	@receiver pm
//	@param hash - 交易的哈希值
//	@return bool - 如果交易出现过，返回 true
func (pm *P2PManager) isKnownTransaction(hash common.Hash) bool {
	strHash := hex.EncodeToString(hash[:])
	if pm.txPool.Contain(strHash) {
		return true
	}

	tx, err := pm.chain.GetTransactionByHash(hash)

	if err != nil {
		return false
	}

	return tx != nil
}

// broadcastBlock
//
//	@Description: 区块广播协程，这里使用的是 go-libp2p 的 gossip 广播
//	@receiver pm
func (pm *P2PManager) broadcastBlock() {
	log.Infoln("P2P manger broadcast block routine start!")
	for {
		select {
		case block := <-pm.blockBroadcastQueue:
			//blockHash := block.Header.BlockHash
			blockData, err := utils.SerializeBlock(block)
			if err != nil {
				log.WithError(err).Errorln("Serialize block failed.")
				continue
			}
			log.Infof("Block #%s size: %d kb txs: %d", block.BlockHash()[:8],
				len(blockData)/1024, len(block.Transactions))

			// 向 Topic 中广播区块
			err = pm.blockTopic.Publish(context.Background(), blockData)
			if err != nil {
				log.WithError(err).Errorln("Publish block failed.")
				continue
			}

			log.Infof("Broadcast block 0x%s", block.BlockHash())
			metrics.GossipBroadcastBlocksCountInc()
		}
	}
}

// broadcastTransaction
//
//	@Description: 交易广播协程，这里使用的是基于 UDP 的 gossip 协议，如果使用 go-libp2p 会影响到区块的广播效率，进而导致分叉
//	@receiver pm
func (pm *P2PManager) broadcastTransaction() {
	log.Infoln("P2P manger broadcast transaction routine start!")
	for {
		select {
		case tx := <-pm.txBroadcastQueue:
			pm.UDPGossipBroadcast(tx)
			time.Sleep(200 * time.Microsecond)
		}
	}
}

// gossipBlockSubscribe
//
//	@Description: 区块接收协程，在 topic 中接收到其它节点广播的区块
//	@receiver pm
//	@param ctx - 广播上下文
//	@param sub - 订阅实例
//	@param h - 本地 p2p 节点实例
func (pm *P2PManager) gossipBlockSubscribe(ctx context.Context,
	sub *pubsub.Subscription, h host.Host) {
	log.Infoln("P2P gossip block subscription routine start!")
	for {
		// 从订阅中接收消息
		blockMsg, err := sub.Next(ctx)
		if err != nil {
			log.Errorf("Get block data from subscription failed: %s", err)
			continue
		}

		// 如果是来自本地的广播，则跳过
		if blockMsg.ReceivedFrom == h.ID() {
			continue
		}

		metrics.GossipReceiveBlocksCountInc()

		// 获取同步状态，如果未同步完成则跳过处理
		status := pm.blockSyncer.getStatus()
		if status == blockSyncing || status == syncPaused {
			continue
		}

		// 尝试反序列化区块，如果失败则跳过并报错
		block, err := utils.DeserializeBlock(blockMsg.Data)
		if err != nil {
			log.WithField("error",
				err).Warning("Deserialize block from bytes failed.")
			return
		}

		blockHash := block.Header.BlockHash
		strHash := hex.EncodeToString(blockHash[:])
		if pm.knownBlock.Contains(strHash) {
			return
		}

		pm.markBlock(strHash)
		log.WithField("hash", strHash).Debugln("Receive new block.")

		if block.Header.Height == 0 {
			metrics.RoutineCreateCounterObserve(18)
			go pm.chain.InsertBlock(block)
			return
		}

		if verifyBlockVRF(block) {
			log.WithField("status", status).Debugln("Receive block from p2p.")
			pm.chain.AppendBlockTask(block)
			//pm.blockBroadcastQueue <- block
		} else {
			//log.Infoln(hex.EncodeToString(block.Header.PublicKey[:]))
			log.Warning("Block VRF verify failed.")
		}
	}
}

// appendBlockToSyncer
//
//	@Description: 将区块添加到同步器中，这里是在端对端接收到区块时进行添加
//	@receiver pm
//	@param block - 区块实例
func (pm *P2PManager) appendBlockToSyncer(block *common.Block) {
	pm.blockSyncer.appendBlock(block)
}

// SetSynced
//
//	@Description: 设置当前的同步器的同步状态为完成
//	@receiver pm
func (pm *P2PManager) SetSynced() {
	pm.blockSyncer.setSynced()
}

// markBlock
//
//	@Description: 标志区块在本地出现过
//	@receiver pm
//	@param hash - 区块的哈希值
func (pm *P2PManager) markBlock(hash string) {
	pm.knownBlock.Add(hash, nil)
}

// markTransaction
//
//	@Description: 标志交易在本地出现过
//	@receiver pm
//	@param hash - 交易的哈希值
func (pm *P2PManager) markTransaction(hash string) {
	pm.knownTransaction.Add(hash, nil)
}

// syncStatus
//
//	@Description: 获取同步器的同步状态
//	@receiver pm
//	@return uint8
func (pm *P2PManager) syncStatus() uint8 {
	return pm.blockSyncer.status
}

// HandleStream
//
//	@Description: 用于在收到对端连接时候处理 stream, 在这里构建 peer 用于通信
//	@receiver pm
//	@param s - 接收到新的连接请求
func (pm *P2PManager) HandleStream(s network.Stream) {
	pm.peerSetLock.Lock()
	defer pm.peerSetLock.Unlock()

	if len(pm.peerSet) > streamLimit {
		_ = s.Close()
		return
	}

	conn := s.Conn()
	remoteAddr := conn.RemoteMultiaddr().String()

	_, err := pm.NewPeer(conn.RemotePeer(), &s, remoteAddr)

	log.Infoln("Receive new stream, handle stream.")
	log.Infoln(conn.RemoteMultiaddr())

	if err != nil {
		log.WithField("error", err).Errorln("Handle stream error.")
		return
	}

	metrics.ConnectedNodeInc()
}

// Discover
//
//	@Description: 基于 kademlia 协议发现其他节点
//	@receiver pm
//	@param ctx - 上下文 context
//	@param h - 本地节点实例
//	@param dht - kademlia 实例
//	@param rendezvous - 进行节点发现的 “密语/暗号”
func (pm *P2PManager) Discover(ctx context.Context, h host.Host,
	dht *dht.IpfsDHT,
	rendezvous string) {
	var err error
	var routingDiscovery = routing.NewRoutingDiscovery(dht)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)

	// 广播路由的初始化不能使用下面的语句，否则广播网络无法工作, 逆天 go-libp2p，
	// _, err := routingDiscovery.Advertise(ctx, rendezvous)

	pm.id = h.ID()

	// 利用 go-libp2p-pubsub 构建广播网络
	pm.gossip, err = pubsub.NewGossipSub(ctx, h,
		pubsub.WithMaxMessageSize(pubsubMaxSize),
	)
	if err != nil {
		log.WithError(err).Fatalf("Create gossip sub failed")
		return
	}

	pm.blockTopic, err = pm.gossip.Join(BlockGossipTopic)
	if err != nil {
		log.WithError(err).Fatalf("Join block topic failed")
		return
	}
	log.Infof("Join to topic %s", BlockGossipTopic)

	blockSub, err := pm.blockTopic.Subscribe(pubsub.WithBufferSize(512))
	if err != nil {
		log.WithError(err).Fatalf("Get sub from topic failed")
		return
	}

	go pm.TransactionUDP()
	go pm.gossipBlockSubscribe(ctx, blockSub, h)

	// 每 500ms 通过 Kademlia 获取对端节点列表
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dht.RefreshRoutingTable()
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous)

			if err != nil {
				log.WithField("error", err).Debugln("Find peers failed.")
				continue
			}
			for p := range peers {
				pm.CheckAndCreateStream(ctx, h, p)
			}
		}
	}

}

// CheckAndCreateStream
//
//	@Description: 检查并创建数据传输流
//	@receiver pm
//	@param ctx - 处理上下文
//	@param h - 本地节点实例
//	@param p - 对端节点实例
func (pm *P2PManager) CheckAndCreateStream(ctx context.Context, h host.Host,
	p peer.AddrInfo) {
	pm.peerSetLock.Lock()
	defer pm.peerSetLock.Unlock()
	// 对端节点 ID 和本地节点 ID 相同
	if len(pm.peers) >= streamLimit || p.ID == h.ID() {
		return
	}

	// 如果在 retryInterval 内尝试连接过，跳过该节点
	if tried, ok := pm.triedPeers[p.ID]; ok {
		if time.Since(tried) < retryInterval {
			return
		}
	}

	// 如果节点已经连接，并且当前的状态正常
	if remote, ok := pm.peers[p.ID]; ok {
		if !remote.Stopped() {
			return
		}
	}

	// 如果没有连接，建立新的连接
	if h.Network().Connectedness(p.ID) != network.Connected {
		_, err := h.Network().DialPeer(ctx, p.ID)
		if err != nil {
			log.WithFields(log.Fields{
				"peerID": p.ID,
				"error":  err,
			}).Debugln("Connect to node failed.")
			return
		}
	}

	stream, err := h.NewStream(ctx, p.ID, ProtocolId)
	if err != nil {
		log.WithError(err).Errorln("Create new stream failed: %d", err)
		pm.triedPeers[p.ID] = time.Now()
		return
	}

	conn := stream.Conn()

	_, err = pm.NewPeer(p.ID, &stream, conn.RemoteMultiaddr().String())
	if err != nil {
		log.WithError(err).Debugln("Create new peer failed.")
		return
	}

	metrics.ConnectedNodeInc()
}

func (pm *P2PManager) Synced() bool {
	status := pm.blockSyncer.getStatus()

	if status == synced && pm.timeSyncer.synced() {
		pm.startRoutine.Do(func() {
			log.Infoln("Block && time sync finish!")
			metrics.RoutineCreateCounterObserve(16)
			go pm.broadcastBlock()
			go pm.broadcastTransaction()
		})
		return true
	}
	return false
}

// StatusMessage 生成同步信息给对端
func (pm *P2PManager) StatusMessage() *p2p.SyncStatusMsg {
	block, err := pm.chain.GetLatestBlock()

	if err != nil || !pm.Synced() {
		return &p2p.SyncStatusMsg{
			LatestHeight:        -1,
			LatestHash:          [32]byte{},
			BufferedStartHeight: 0,
			BufferedEndHeight:   -1,
		}
	}

	return &p2p.SyncStatusMsg{
		LatestHeight:        block.Header.Height,
		LatestHash:          block.Header.BlockHash,
		BufferedStartHeight: 0,
		BufferedEndHeight:   pm.chain.BufferedHeight(),
	}
}

// removePeerIfStopped 移除断开连接的 Peer
// todo: 这里的方法感觉有点复杂，以后需要优化
// todo: 这里不能在遍历的时候进行移除
func (pm *P2PManager) removePeerIfStopped(idx int) bool {
	pm.peerSetLock.Lock()
	defer pm.peerSetLock.Unlock()

	p := pm.peerSet[idx]
	if p.Stopped() {
		pm.peerSet = append(pm.peerSet[:idx], pm.peerSet[idx+1:]...)
		delete(pm.peers, p.peerID)
		return true
	}

	return false
}

// UDPGossipBroadcast
//
//	@Description: 交易的 UDP 广播方法，随机选取，邻居节点发送交易
//	@receiver pm
//	@param tx
func (pm *P2PManager) UDPGossipBroadcast(tx *common.Transaction) {
	pm.peerSetLock.RLock()
	defer pm.peerSetLock.RUnlock()

	txData, err := utils.SerializeTransaction(tx)
	if err != nil {
		// todo: 如果交易序列化失败先不处理
		log.WithError(err).Errorln("Serialize transaction failed.")
		return
	}

	if len(pm.peerSet) <= 0 {
		return
	}

	for i := 0; i < gossipNodes; i++ {
		randomIdx := rand.Intn(len(pm.peerSet))
		p := pm.peerSet[randomIdx]

		if p == nil {
			continue
		}

		multAddr, err := multiaddr.NewMultiaddr(p.addr)
		//log.Infoln(p.peer.Id().String())
		if err != nil {
			log.WithError(err).Errorln("Get peer addr failed.")
			continue
		}

		ip4Addr, err := multAddr.ValueForProtocol(multiaddr.P_IP4)
		if err != nil {
			log.WithError(err).Errorln("Get IPV4 address failed.")
			continue
		}

		ip, err := net.ResolveIPAddr("ip", ip4Addr)
		if err != nil {
			log.WithError(err).Errorln("Cast string to ip failed.")
			continue
		}

		socket, err := net.DialUDP("udp", nil, &net.UDPAddr{
			IP:   ip.IP,
			Port: 53333,
		})

		if err != nil {
			continue
		}

		_, err = socket.Write(txData)
		if err != nil {
			continue
		}

		metrics.GossipUDPSendCountInc()
	}
}

// TransactionUDP
//
//	@Description: 从 UDP 中接收其它节点广播到该节点的交易
//	@receiver pm
func (pm *P2PManager) TransactionUDP() {
	port := config.Int("node.udp", 53333)
	listen, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: port,
	})

	if err != nil {
		log.WithError(err).Warningln("Transaction UDP port listen failed.")
		return
	}

	defer listen.Close()
	log.Infoln("Transaction gossip routine start!")

	for {
		var data [udpBufferSize]byte
		n, _, err := listen.ReadFromUDP(data[:])
		if err != nil {
			log.WithError(err).Warningln("Receive tx failed.")
			continue
		}

		tx, err := utils.DeserializeTransaction(data[:n])

		if err != nil {
			log.WithError(err).Warningln("Deserialize data failed.")
			continue
		}

		// todo: 如何验证一个交易是非法的？当前的逻辑下可以使用 Verify 方法来校验
		txHash := hex.EncodeToString(tx.Body.Hash[:])
		exists := pm.knownTransaction.Contains(txHash)
		if !exists {
			pm.AddTransaction(tx)
		}

		metrics.GossipUDPRecvCountInc()
		//}
	}
}

func (pm *P2PManager) GetConnectNodeInfo() (string, []string) {
	pm.peerSetLock.RLock()
	defer pm.peerSetLock.RUnlock()

	result := make([]string, len(pm.peerSet))

	for idx, p := range pm.peerSet {
		result[idx] = p.peerID.String()
	}

	return pm.id.String(), result
}

func GetLogicClock() int64 {
	pm := GetP2PManager()
	return pm.timeSyncer.GetLogicClock()
}
