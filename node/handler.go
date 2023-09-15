package node

import (
	"crypto/elliptic"
	"encoding/hex"
	"github.com/gookit/config/v2"
	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/core"
	"go-chronos/crypto"
	"go-chronos/metrics"
	"go-chronos/p2p"
	"go-chronos/utils"
	"math"
	"net"
	"sync"
	"time"
)

type msgHandler func(h *Handler, msg *p2p.Message, p *Peer)

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
	handlerInst *Handler = nil
)

type HandlerConfig struct {
	TxPool       *core.TxPool
	Chain        *core.BlockChain
	Genesis      bool
	InitialDelta int64 // 初始时间偏移，仅仅用于进行时间同步测试
}

type Handler struct {
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
func HandleStream(s network.Stream) {
	//ctx := GetPeerContext()
	handler := GetHandlerInst()
	conn := s.Conn()

	//addr, _ := peer.AddrInfoFromP2pAddr(conn.RemoteMultiaddr())
	//log.Infoln(addr)

	_, err := handler.NewPeer("", conn.RemotePeer(), &s)

	log.Infoln("Receive new stream, handle stream.")
	log.Infoln(conn.RemoteMultiaddr())

	if err != nil {
		log.WithField("error", err).Errorln("Handle stream error.")
		return
	}

	metrics.ConnectedNodeInc()
}

func NewHandler(config *HandlerConfig) (*Handler, error) {
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

	handler := &Handler{
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
func (h *Handler) AddTransaction(tx *common.Transaction) {
	h.txBroadcastQueue <- tx
	h.txPool.Add(tx)
}

func GetHandlerInst() *Handler {
	// todo: 这样的写法会有脏读的问题，也就是在分配地址后，对象可能还没有完全初始化
	return handlerInst
}

func (h *Handler) packageBlockRoutine() {
	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-ticker.C:
			timestamp := h.timeSyncer.GetLogicClock()
			// 如果逻辑时间距离 2s 则进行区块的打包
			if (timestamp/1000)%packageBlockInterval != 0 {
				continue
			}

			if !h.Synced() {
				log.Infoln("Waiting for node synced.")
				continue
			}

			//log.Debugln("Start package block.")
			latest, _ := h.chain.GetLatestBlock()

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

			txs := h.txPool.Package()
			//log.Infof("Package %d txs.", len(txs))
			newBlock, err := h.chain.PackageNewBlock(txs, timestamp, &params, packageBlockInterval)

			if err != nil {
				log.WithField("error", err).Debugln("Package new block failed.")
				continue
			}

			h.chain.AppendBlockTask(newBlock)
			h.blockBroadcastQueue <- newBlock
			log.Infof("Package new block# 0x%s", hex.EncodeToString(newBlock.Header.BlockHash[:]))
		}
	}
}

func (h *Handler) NewPeer(addr string, peerId peer.ID, s *network.Stream) (*Peer, error) {
	config := PeerConfig{
		chain:   h.chain,
		txPool:  h.txPool,
		handler: h,
	}

	p, err := NewPeer(addr, peerId, s, config)

	if err != nil {
		log.WithField("error", err).Errorln("Create peer failed.")
		return nil, err
	}

	h.peerSetLock.Lock()
	defer h.peerSetLock.Unlock()

	h.peerSet = append(h.peerSet, p)
	h.peers[p.peerID] = p
	h.blockSyncer.AddPeer(p)
	return p, err
}

func (h *Handler) isKnownTransaction(hash common.Hash) bool {
	strHash := hex.EncodeToString(hash[:])
	if h.txPool.Contain(strHash) {
		return true
	}

	tx, err := h.chain.GetTransactionByHash(hash)

	if err != nil {
		return false
	}

	return tx != nil
}

func (h *Handler) broadcastBlock() {
	for {
		select {
		case block := <-h.blockBroadcastQueue:
			blockHash := block.Header.BlockHash

			peers := h.getPeersWithoutBlock(blockHash)

			peersCount := len(peers)
			//countBroadcastBlock := int(math.Sqrt(float64(peersCount)))
			countBroadcastBlock := 0
			log.WithField("count", countBroadcastBlock).Infoln("Broadcast block.")

			for idx := 0; idx < countBroadcastBlock; idx++ {
				peers[idx].AsyncSendNewBlock(block)
			}

			for idx := countBroadcastBlock; idx < peersCount; idx++ {
				peers[idx].AsyncSendNewBlockHash(block.Header.BlockHash)
			}
		}
	}
}

func (h *Handler) broadcastTransaction() {
	//log.Infof("Start broadcast.")
	for {
		select {
		case tx := <-h.txBroadcastQueue:
			txHash := tx.Body.Hash
			peers := h.getPeersWithoutTransaction(txHash)

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

func (h *Handler) getPeersWithoutBlock(blockHash common.Hash) []*Peer {
	h.peerSetLock.RLock()
	defer h.peerSetLock.RUnlock()

	list := make([]*Peer, 0, len(h.peerSet))
	strHash := hex.EncodeToString(blockHash[:])
	for idx := range h.peerSet {
		p := h.peerSet[idx]
		if !p.KnownBlock(strHash) && !p.Stopped() {
			list = append(list, p)
		}
	}
	return list
}

func (h *Handler) appendBlockToSyncer(block *common.Block) {
	h.blockSyncer.appendBlock(block)
}

func (h *Handler) getPeersWithoutTransaction(txHash common.Hash) []*Peer {
	h.peerSetLock.RLock()
	defer h.peerSetLock.RUnlock()

	list := make([]*Peer, 0, len(h.peerSet))
	strHash := hex.EncodeToString(txHash[:])
	for idx := range h.peerSet {
		p := h.peerSet[idx]
		if !p.KnownTransaction(strHash) && !p.Stopped() {
			list = append(list, p)
		}
	}
	return list
}

func (h *Handler) SetSynced() {
	h.blockSyncer.setSynced()
}

func (h *Handler) markBlock(hash string) {
	h.knownBlock.Add(hash, nil)
}

func (h *Handler) markTransaction(hash string) {
	h.knownTransaction.Add(hash, nil)
}

func (h *Handler) syncStatus() uint8 {
	return h.blockSyncer.status
}

func (h *Handler) Synced() bool {
	status := h.blockSyncer.getStatus()
	if status == synced && h.timeSyncer.synced() {
		h.startRoutine.Do(func() {
			log.Infoln("Block && time sync finish!")
			metrics.RoutineCreateHistogramObserve(16)
			go h.broadcastBlock()
			go h.broadcastTransaction()
		})
		return true
	}
	return false
}

// StatusMessage 生成同步信息给对端
func (h *Handler) StatusMessage() *p2p.SyncStatusMsg {
	block, err := h.chain.GetLatestBlock()

	if err != nil || !h.Synced() {
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
		BufferedEndHeight:   h.chain.BufferedHeight(),
	}
}

// removePeerIfStopped 移除断开连接的 Peer
// todo: 这里的方法感觉有点复杂，以后需要优化
// todo: 这里不能在遍历的时候进行移除
func (h *Handler) removePeerIfStopped(idx int) {
	h.peerSetLock.Lock()
	defer h.peerSetLock.Unlock()

	p := h.peerSet[idx]
	if p.Stopped() {
		h.peerSet = append(h.peerSet[:idx], h.peerSet[idx+1:]...)
		delete(h.peers, p.peerID)
	}
}

func (h *Handler) TransactionUDP() {
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
			peer := h.peers[id]
			if peer != nil {
				txHash := hex.EncodeToString(tx.Body.Hash[:])
				peer.MarkTransaction(txHash)
			}
			h.AddTransaction(tx)
		}
	}
}

func GetLogicClock() int64 {
	h := GetHandlerInst()
	return h.timeSyncer.GetLogicClock()
}
