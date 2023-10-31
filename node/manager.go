package node

import (
	"context"
	"crypto/elliptic"
	"encoding/hex"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/core"
	"github.com/chain-lab/go-chronos/crypto"
	"github.com/chain-lab/go-chronos/metrics"
	"github.com/chain-lab/go-chronos/p2p"
	"github.com/chain-lab/go-chronos/utils"
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
	p2p.StatusCodeStatusMsg:         handleStatusMsg,         // 状态消息，目前接收对端的高度信息
	p2p.StatusCodeBlockBodiesMsg:    handleBlockMsg,          // 对应上一个状态码，如果对端请求区块，在缓冲区中取出区块进行响应
	p2p.StatusCodeGetBlockBodiesMsg: handleGetBlockBodiesMsg, // 请求本地缓冲区中不存在的区块
	p2p.StatusCodeSyncStatusReq:     handleSyncStatusReq,     // 携带本地的高度信息，请求对端的状态信息，例如高度/缓冲区高度
	p2p.StatusCodeSyncStatusMsg:     handleSyncStatusMsg,     // 响应对端的状态请求
	p2p.StatusCodeSyncGetBlocksMsg:  handleSyncGetBlocksMsg,  // 根据高度请求区块
	p2p.StatusCodeSyncBlocksMsg:     handleSyncBlockMsg,      // 响应对应高度的区块
	p2p.StatusCodeTimeSyncReq:       handleTimeSyncReq,       // 时间同步请求
	p2p.StatusCodeTimeSyncRsp:       handleTimeSyncRsp,       // 时间同步响应
	p2p.StatusCodeNewBlockHashesMsg: handleNewBlockHashMsg,   // 广播新打包的区块哈希值，在同步旧区块（非缓冲区同步状态）时不处理
	//p2p.StatusCodeNewBlockMsg:       handleNewBlockMsg,       // 广播新打包的区块，在同步旧区块（非缓冲区同步状态）时不处理
}

var (
	handlerInst *P2PManager = nil
)

const (
	retryInterval          = 10 * time.Minute
	streamLimit            = 20
	maxKnownBlock          = 1024
	maxKnownTransaction    = 32768
	maxSyncerStatusChannel = 512
	packageBlockInterval   = 2
	gossipNodes            = 4
	NetworkRendezvous      = "chronos"
	TxGossipTopic          = "/chronos/1.0.1/transactions"
	BlockGossipTopic       = "/chronos/1.0.1/blocks"
	ProtocolId             = protocol.ID("/chronos/1.0.0/p2p")
	TxProtocolId           = protocol.ID("/chronos/1.0.0/transaction")
)

type P2PManagerConfig struct {
	TxPool       *core.TxPool
	Chain        *core.BlockChain
	Genesis      bool
	InitialDelta int64 // 初始时间偏移，仅仅用于进行时间同步测试
}

type P2PManager struct {
	id         peer.ID
	triedPeers map[peer.ID]time.Time // 尝试连接的节点和连接时间戳，在一定时间内不再进行尝试
	peerSet    []*Peer               // 已建立连接的节点列表
	peers      map[peer.ID]*Peer     // 节点 ID -> 节点对象

	gossip     *pubsub.PubSub
	flood      *pubsub.PubSub
	blockTopic *pubsub.Topic
	txTopic    *pubsub.Topic

	blockBroadcastQueue chan *common.Block
	txBroadcastQueue    chan *common.Transaction

	knownBlock       *lru.Cache
	knownTransaction *lru.Cache

	txPool *core.TxPool
	chain  *core.BlockChain

	blockSyncer  *BlockSyncer
	timeSyncer   *TimeSyncer
	startRoutine sync.Once

	peerSetLock sync.RWMutex
	genesis     bool
}

func NewP2PManager(config *P2PManagerConfig) (*P2PManager, error) {
	knownBlockCache, err := lru.New(maxKnownBlock)

	if err != nil {
		log.WithField("error", err).Debugln("Create known block cache failed.")
		return nil, err
	}

	knownTxCache, err := lru.New(maxKnownTransaction)

	if err != nil {
		log.WithField("error", err).Debugln("Create known transaction cache failed.")
		return nil, err
	}

	blockSyncerConfig := &BlockSyncerConfig{
		Chain: config.Chain,
	}
	bs := NewBlockSyncer(blockSyncerConfig)
	ts := NewTimeSyncer(config.Genesis, config.InitialDelta)

	handler := &P2PManager{
		// todo: 限制节点数量，Kad 应该限制了节点数量不超过20个
		triedPeers: make(map[peer.ID]time.Time),
		peerSet:    make([]*Peer, 0, 40),
		peers:      make(map[peer.ID]*Peer),

		blockBroadcastQueue: make(chan *common.Block, 256),
		txBroadcastQueue:    make(chan *common.Transaction, 8192*4),

		knownBlock:       knownBlockCache,
		knownTransaction: knownTxCache,

		txPool: config.TxPool,
		chain:  config.Chain,

		blockSyncer: bs,
		timeSyncer:  ts,

		genesis: config.Genesis,
	}

	metrics.RoutineCreateCounterObserve(15)
	go handler.packageBlockRoutine()
	bs.Start()
	ts.Start()
	handlerInst = handler

	return handler, nil
}

// AddTransaction 仅用于测试
func (pm *P2PManager) AddTransaction(tx *common.Transaction) {
	txHash := hex.EncodeToString(tx.Body.Hash[:])
	pm.markTransaction(txHash)
	pm.txBroadcastQueue <- tx
	pm.txPool.Add(tx)
}

func GetP2PManager() *P2PManager {
	// todo: 这样的写法会有脏读的问题，也就是在分配地址后，对象可能还没有完全初始化
	return handlerInst
}

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

			if !pm.Synced() {
				log.Infoln("Waiting for node synced.")
				continue
			}

			latest, _ := pm.chain.GetLatestBlock()

			if latest == nil {
				log.Infoln("Waiting for genesis block.")
				continue
			}

			calc := crypto.GetCalculatorInstance()
			full := pm.chain.BufferFull()
			//pm.chain.
			seed, pi := calc.GetSeedParams(full)
			log.Debugf("Get seed: %s", hex.EncodeToString(seed.Bytes()))

			consensus, err := crypto.VRFCheckLocalConsensus(seed.Bytes())
			if !consensus || err != nil {
				log.Infof("Local is not consensus node")
				continue
			}

			randNumber, s, t, err := crypto.VRFCalculate(elliptic.P256(), seed.Bytes())
			log.Debugf("Package with seed: %s", hex.EncodeToString(seed.Bytes()))

			params := common.GeneralParams{
				Result:       seed.Bytes(),
				Proof:        pi.Bytes(),
				RandomNumber: [33]byte(randNumber),
				S:            s.Bytes(),
				T:            t.Bytes(),
			}

			txs := pm.txPool.Package()
			//log.Infof("Package %d txs.", len(txs))
			newBlock, err := pm.chain.PackageNewBlock(txs, timestamp, &params, packageBlockInterval)

			if err != nil {
				log.WithField("error", err).Debugln("Package new block failed.")
				continue
			}

			pm.chain.AppendBlockTask(newBlock)
			pm.blockBroadcastQueue <- newBlock
			log.Infof("Package new block# 0x%s", hex.EncodeToString(newBlock.Header.BlockHash[:]))
		}
	}
}

func (pm *P2PManager) NewPeer(peerId peer.ID, s *network.Stream,
	remoteAddr string) (*Peer,
	error) {
	cfg := PeerConfig{
		chain:          pm.chain,
		txPool:         pm.txPool,
		handler:        pm,
		remoteMultAddr: remoteAddr,
	}

	p, err := NewPeer(peerId, s, cfg)

	if err != nil {
		log.WithField("error", err).Errorln("Create peer failed.")
		return nil, err
	}

	pm.peerSet = append(pm.peerSet, p)
	log.Infoln(p.addr)
	pm.peers[p.peerID] = p
	pm.blockSyncer.AddPeer(p)
	return p, err
}

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

			err = pm.blockTopic.Publish(context.Background(), blockData)
			if err != nil {
				log.WithError(err).Errorln("Publish block failed.")
				continue
			}
		}
	}
}

func (pm *P2PManager) broadcastTransaction() {
	log.Infoln("P2P manger broadcast transaction routine start!")
	for {
		select {
		case tx := <-pm.txBroadcastQueue:
			pm.UDPGossipBroadcast(tx)
		}
	}
}

func (pm *P2PManager) gossipBlockSubscribe(ctx context.Context,
	sub *pubsub.Subscription, h host.Host) {
	log.Infoln("P2P gossip block subscription routine start!")
	for {
		blockMsg, err := sub.Next(ctx)

		if err != nil {
			log.Errorf("Get block data from subscription failed: %s", err)
			continue
		}

		if blockMsg.ReceivedFrom == h.ID() {
			continue
		}

		metrics.GossipReceiveBlocksCountInc()

		status := pm.blockSyncer.getStatus()
		if status == blockSyncing || status == syncPaused {
			continue
		}

		block, err := utils.DeserializeBlock(blockMsg.Data)

		if err != nil {
			log.WithField("error", err).Debugln("Deserialize block from bytes failed.")
			return
		}

		blockHash := block.Header.BlockHash
		strHash := hex.EncodeToString(blockHash[:])
		if pm.knownBlock.Contains(strHash) {
			return
		}

		pm.markBlock(strHash)

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

func (pm *P2PManager) gossipTxSubscribe(ctx context.Context,
	sub *pubsub.Subscription, h host.Host) {
	log.Infoln("P2P gossip transaction subscription routine start!")
	for {
		txMsg, err := sub.Next(ctx)

		if err != nil {
			log.Errorf("Get tx data from subscription failed: %s", err)
			continue
		}

		if txMsg.ReceivedFrom == h.ID() {
			continue
		}
		//log.Infoln("Receive message from gossip.")

		if !pm.Synced() {
			continue
		}

		metrics.GossipReceiveCountInc()
		transaction, err := utils.DeserializeTransaction(txMsg.Data)
		if err != nil {
			log.WithField("error", err).Errorln("Deserializer transaction failed.")
			continue
		}

		txHash := hex.EncodeToString(transaction.Body.Hash[:])
		if pm.isKnownTransaction(transaction.Body.Hash) {
			return
		}

		pm.markTransaction(txHash)
		pm.txPool.Add(transaction)
	}
}

func (pm *P2PManager) getPeersWithoutBlock(blockHash common.Hash) []*Peer {
	pm.peerSetLock.RLock()
	defer pm.peerSetLock.RUnlock()

	list := make([]*Peer, 0, len(pm.peerSet))
	strHash := hex.EncodeToString(blockHash[:])
	for idx := range pm.peerSet {
		p := pm.peerSet[idx]
		if !p.KnownBlock(strHash) && !p.Stopped() {
			list = append(list, p)
		}
	}
	return list
}

func (pm *P2PManager) appendBlockToSyncer(block *common.Block) {
	pm.blockSyncer.appendBlock(block)
}

func (pm *P2PManager) getPeersWithoutTransaction(txHash common.Hash) []*Peer {
	pm.peerSetLock.RLock()
	defer pm.peerSetLock.RUnlock()

	list := make([]*Peer, 0, len(pm.peerSet))
	strHash := hex.EncodeToString(txHash[:])
	for idx := range pm.peerSet {
		p := pm.peerSet[idx]
		if !p.KnownTransaction(strHash) && !p.Stopped() {
			list = append(list, p)
		}
	}
	return list
}

func (pm *P2PManager) SetSynced() {
	pm.blockSyncer.setSynced()
}

func (pm *P2PManager) markBlock(hash string) {
	pm.knownBlock.Add(hash, nil)
}

func (pm *P2PManager) markTransaction(hash string) {
	pm.knownTransaction.Add(hash, nil)
}

func (pm *P2PManager) syncStatus() uint8 {
	return pm.blockSyncer.status
}

// HandleStream 用于在收到对端连接时候处理 stream, 在这里构建 peer 用于通信
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

// Discover 基于 kademlia 协议发现其他节点
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
	pm.gossip, err = pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.WithError(err).Fatalf("Create gossip sub failed")
		return
	}
	//pm.flood, err = pubsub.NewFloodSub(ctx, h)
	//if err != nil {
	//	log.WithError(err).Fatalf("Create flood sub failed")
	//	return
	//}

	//pm.txTopic, err = pm.gossip.Join(TxGossipTopic)
	//if err != nil {
	//	log.WithError(err).Fatalf("Join transaction topic failed")
	//	return
	//}
	//log.Infof("Join to topic %s", TxGossipTopic)
	pm.blockTopic, err = pm.gossip.Join(BlockGossipTopic)
	if err != nil {
		log.WithError(err).Fatalf("Join block topic failed")
		return
	}
	log.Infof("Join to topic %s", BlockGossipTopic)

	//txSub, err := pm.txTopic.Subscribe()
	//if err != nil {
	//	log.WithError(err).Fatalf("Get sub from topic failed")
	//	return
	//}
	blockSub, err := pm.blockTopic.Subscribe()
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

func (pm *P2PManager) UDPGossipBroadcast(tx *common.Transaction) {
	pm.peerSetLock.RLock()
	defer pm.peerSetLock.RUnlock()

	txData, err := utils.SerializeTransaction(tx)
	if err != nil {
		// todo: 如果交易序列化失败先不处理
		log.WithError(err).Errorln("Serialize transaction failed.")
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
			Port: 31259,
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
	time.Sleep(500 * time.Microsecond)
}

// TransactionUDP 计划使用的交易广播独立网络，
func (pm *P2PManager) TransactionUDP() {
	port := config.Int("node.udp", 31259)
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
		var data [10240]byte
		n, _, err := listen.ReadFromUDP(data[:])
		if err != nil {
			log.WithError(err).Warningln("Receive tx failed.")
			continue
		}

		// bm: BroadcastMessage
		tx, err := utils.DeserializeTransaction(data[:n])

		//var id peer.ID
		//var txData []byte

		if err != nil {
			log.WithError(err).Warningln("Deserialize data failed.")
			continue
		}

		// todo: 如何验证一个交易是非法的？当前的逻辑下可以使用 Verify 方法来校验
		//tx, err := utils.DeserializeTransaction(txData)
		//if err != nil {
		//p := pm.peers[id]
		//if p != nil {
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
