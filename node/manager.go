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
	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	log "github.com/sirupsen/logrus"
	"math"
	"net"
	"sync"
	"time"
)

type msgHandler func(pm *P2PManager, msg *p2p.Message, p *Peer)

const (
	maxKnownBlock          = 1024
	maxKnownTransaction    = 32768
	maxSyncerStatusChannel = 512
	packageBlockInterval   = 2
	ProtocolId             = protocol.ID("/chronos/1.0.0/p2p")
	TxProtocolId           = protocol.ID("/chronos/1.0.0/transaction")
)

var handlerMap = map[p2p.StatusCode]msgHandler{
	p2p.StatusCodeStatusMsg:                     handleStatusMsg,
	p2p.StatusCodeNewBlockMsg:                   handleNewBlockMsg,
	p2p.StatusCodeNewBlockHashesMsg:             handleNewBlockHashMsg,
	p2p.StatusCodeBlockBodiesMsg:                handleBlockMsg,
	p2p.StatusCodeGetBlockBodiesMsg:             handleGetBlockBodiesMsg,
	p2p.StatusCodeTransactionsMsg:               handleTransactionMsg,
	p2p.StatusCodeNewPooledTransactionHashesMsg: handleNewPooledTransactionHashesMsg,
	p2p.StatusCodeGetPooledTransactionMsg:       handleGetPooledTransactionMsg,
	p2p.StatusCodeSyncStatusReq:                 handleSyncStatusReq,
	p2p.StatusCodeSyncStatusMsg:                 handleSyncStatusMsg,
	p2p.StatusCodeSyncGetBlocksMsg:              handleSyncGetBlocksMsg,
	p2p.StatusCodeSyncBlocksMsg:                 handleSyncBlockMsg,
	p2p.StatusCodeTimeSyncReq:                   handleTimeSyncReq,
	p2p.StatusCodeTimeSyncRsp:                   handleTimeSyncRsp,
}

var (
	handlerInst *P2PManager = nil
)

type P2PManagerConfig struct {
	TxPool       *core.TxPool
	Chain        *core.BlockChain
	Genesis      bool
	InitialDelta int64 // 初始时间偏移，仅仅用于进行时间同步测试
}

type P2PManager struct {
	peerSet []*Peer
	peers   map[peer.ID]*Peer

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

// HandleStream 用于在收到对端连接时候处理 stream, 在这里构建 peer 用于通信
func (pm *P2PManager) HandleStream(s network.Stream) {
	conn := s.Conn()

	//addr, _ := peer.AddrInfoFromP2pAddr(conn.RemoteMultiaddr())
	//log.Infoln(addr)

	_, err := pm.NewPeer(conn.RemotePeer(), &s)

	log.Infoln("Receive new stream, handle stream.")
	log.Infoln(conn.RemoteMultiaddr())

	if err != nil {
		log.WithField("error", err).Errorln("Handle stream error.")
		return
	}

	metrics.ConnectedNodeInc()
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
		peerSet: make([]*Peer, 0, 40),
		peers:   make(map[peer.ID]*Peer),

		blockBroadcastQueue: make(chan *common.Block, 64),
		txBroadcastQueue:    make(chan *common.Transaction, 8192),

		knownBlock:       knownBlockCache,
		knownTransaction: knownTxCache,

		txPool: config.TxPool,
		chain:  config.Chain,

		blockSyncer: bs,
		timeSyncer:  ts,

		genesis: config.Genesis,
	}

	metrics.RoutineCreateHistogramObserve(15)
	go handler.packageBlockRoutine()
	bs.Start()
	ts.Start()
	handlerInst = handler

	return handler, nil
}

// AddTransaction 仅用于测试
func (pm *P2PManager) AddTransaction(tx *common.Transaction) {
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

			//log.Debugln("Start package block.")
			latest, _ := pm.chain.GetLatestBlock()

			if latest == nil {
				log.Infoln("Waiting for genesis block.")
				continue
			}

			calc := crypto.GetCalculatorInstance()
			seed, pi := calc.GetSeedParams()

			consensus, err := crypto.VRFCheckLocalConsensus(seed.Bytes())
			if !consensus || err != nil {
				log.Infoln("Local is not consensus node.")
				continue
			}

			randNumber, s, t, err := crypto.VRFCalculate(elliptic.P256(), seed.Bytes())
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

func (pm *P2PManager) NewPeer(peerId peer.ID, s *network.Stream) (*Peer, error) {
	cfg := PeerConfig{
		chain:   pm.chain,
		txPool:  pm.txPool,
		handler: pm,
	}

	p, err := NewPeer(peerId, s, cfg)

	if err != nil {
		log.WithField("error", err).Errorln("Create peer failed.")
		return nil, err
	}

	pm.peerSetLock.Lock()
	defer pm.peerSetLock.Unlock()

	pm.peerSet = append(pm.peerSet, p)
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
	for {
		select {
		case block := <-pm.blockBroadcastQueue:
			blockHash := block.Header.BlockHash

			peers := pm.getPeersWithoutBlock(blockHash)

			peersCount := len(peers)
			countBroadcastBlock := int(math.Sqrt(float64(peersCount)))
			//countBroadcastBlock := 0
			//log.WithField("count", countBroadcastBlock).Infoln("Broadcast block.")

			for idx := 0; idx < countBroadcastBlock; idx++ {
				peers[idx].AsyncSendNewBlock(block)
			}

			for idx := countBroadcastBlock; idx < peersCount; idx++ {
				peers[idx].AsyncSendNewBlockHash(block.Header.BlockHash)
			}
		}
	}
}

func (pm *P2PManager) broadcastTransaction() {
	//log.Infof("Start broadcast.")
	for {
		select {
		case tx := <-pm.txBroadcastQueue:
			txHash := tx.Body.Hash
			peers := pm.getPeersWithoutTransaction(txHash)

			peersCount := len(peers)

			//todo: 这里是由于获取新区块的逻辑还没有，所以先全部广播
			//  在后续完成对应的逻辑后，再修改这里的逻辑来降低广播的时间复杂度
			countBroadcastBody := int(math.Sqrt(float64(peersCount)))
			//countBroadcastBody := peersCount

			for idx := 0; idx < countBroadcastBody; idx++ {
				peers[idx].AsyncSendTransaction(tx)
			}

			for idx := countBroadcastBody; idx < peersCount; idx++ {
				peers[idx].AsyncSendTxHash(txHash)
			}
		}
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

// Discover 基于 kademlia 协议发现其他节点
func (pm *P2PManager) Discover(ctx context.Context, h host.Host,
	dht *dht.IpfsDHT,
	rendezvous string) {
	var routingDiscovery = routing.NewRoutingDiscovery(dht)
	ttl, err := routingDiscovery.Advertise(ctx, rendezvous)

	if err != nil {
		log.WithField("error", err).Errorln("Routing discovery start failed.")
	}

	log.WithFields(log.Fields{
		"ttl": ttl,
	}).Infoln("Routing discovery start.")

	// 每 5s 通过 Kademlia 获取对端节点列表
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	handler := GetP2PManager()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dht.RefreshRoutingTable()
			peers, err := routingDiscovery.FindPeers(ctx, rendezvous,
				discovery.Limit(40))

			if err != nil {
				log.WithField("error", err).Errorln("Find peers failed.")
				continue
			}

			for p := range peers {
				if p.ID == h.ID() {
					continue
				}
				peer := handler.peers[p.ID]
				if peer == nil || peer.Stopped() {
					if h.Network().Connectedness(p.ID) != network.Connected {
						_, err := h.Network().DialPeer(ctx, p.ID)
						if err != nil {
							log.WithFields(log.Fields{
								"peerID": p.ID,
								"error":  err,
							}).Debugln("Connect to node failed.")
							continue
						}
					} else {
						//log.Infoln(h.Network().ConnsToPeer(p.ID))
						log.Debugf("%s connected", p.ID)
					}

					s, err := h.NewStream(ctx, p.ID, ProtocolId)
					if err != nil {
						log.WithError(err).Debugln("Create new stream failed.")
						continue
					}

					_, err = handler.NewPeer(p.ID, &s)
					if err != nil {
						log.WithError(err).Debugln("Create new peer failed.")
						continue
					}

					log.Debugln("Connect to peer: %s", p.ID)
					metrics.ConnectedNodeInc()
				}
			}
		}
	}

}

func (pm *P2PManager) Synced() bool {
	status := pm.blockSyncer.getStatus()
	if status == synced && pm.timeSyncer.synced() {
		pm.startRoutine.Do(func() {
			log.Infoln("Block && time sync finish!")
			metrics.RoutineCreateHistogramObserve(16)
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
func (pm *P2PManager) removePeerIfStopped(idx int) {
	pm.peerSetLock.Lock()
	defer pm.peerSetLock.Unlock()

	p := pm.peerSet[idx]
	if p.Stopped() {
		pm.peerSet = append(pm.peerSet[:idx], pm.peerSet[idx+1:]...)
		delete(pm.peers, p.peerID)
	}
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

	for {
		var data [10240]byte
		n, _, err := listen.ReadFromUDP(data[:])
		if err != nil {
			log.WithError(err).Debugln("Receive tx failed.")
			continue
		}

		// bm: BroadcastMessage
		bm, err := utils.DeserializeBroadcastMessage(data[:n])

		var id peer.ID
		var txData []byte

		if err != nil {
			id = peer.ID(bm.ID)
			txData = bm.Data
		} else {
			continue
		}

		// todo: 如何验证一个交易是非法的？当前的逻辑下可以使用 Verify 方法来校验
		tx, err := utils.DeserializeTransaction(txData)
		if err != nil {
			peer := pm.peers[id]
			if peer != nil {
				txHash := hex.EncodeToString(tx.Body.Hash[:])
				peer.MarkTransaction(txHash)
			}
			pm.AddTransaction(tx)
		}
	}
}

func GetLogicClock() int64 {
	pm := GetP2PManager()
	return pm.timeSyncer.GetLogicClock()
}
