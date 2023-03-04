package node

import (
	"encoding/binary"
	"encoding/hex"
	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/core"
	"go-chronos/p2p"
	"go-chronos/utils"
	"math"
	"time"
)

type msgHandler func(h *Handler, msg *p2p.Message, p *Peer)

const (
	maxKnownBlock       = 1024
	maxKnownTransaction = 32768
	ProtocolId          = protocol.ID("/chronos/1.0.0")
)

var handlerMap = map[p2p.StatusCode]msgHandler{
	p2p.StatusCodeStatusMsg:                     handleStatusMsg,
	p2p.StatusCodeNewBlockMsg:                   handleNewBlockMsg,
	p2p.StatusCodeNewBlockHashesMsg:             handleNewBlockHashMsg,
	p2p.StatusCodeBlockBodiesMsg:                handleBlockMsg,
	p2p.StatusCodeTransactionsMsg:               handleTransactionMsg,
	p2p.StatusCodeNewPooledTransactionHashesMsg: handleNewPooledTransactionHashesMsg,
	p2p.StatusCodeGetPooledTransactionMsg:       handleGetPooledTransactionMsg,
	p2p.StatusCodeGetBlockByHeightMsg:           handleGetBlockByHeightMsg,
}

var (
	handlerInst *Handler = nil
)

type HandlerConfig struct {
	TxPool *core.TxPool
	Chain  *core.BlockChain
}

type Handler struct {
	peerSet []*Peer

	blockBroadcastQueue chan *common.Block
	txBroadcastQueue    chan *common.Transaction

	knownBlock       *lru.Cache
	knownTransaction *lru.Cache

	txPool *core.TxPool
	chain  *core.BlockChain
}

// HandleStream 用于在收到对端连接时候处理 stream, 在这里构建 peer 用于通信
func HandleStream(s network.Stream) {
	//ctx := GetPeerContext()
	handler := GetHandlerInst()
	conn := s.Conn()
	//peer, err := NewPeer(ctx, conn.RemotePeer(), &conn)
	_, err := handler.NewPeer(conn.RemotePeer(), &s)

	log.Infoln("Receive new stream, handle stream.")

	if err != nil {
		log.WithField("error", err).Errorln("Handle stream error.")
		return
	}
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

	handler := &Handler{
		// todo: 限制节点数量，Kad 应该限制了节点数量不超过20个
		peerSet: make([]*Peer, 0, 40),

		blockBroadcastQueue: make(chan *common.Block, 64),
		txBroadcastQueue:    make(chan *common.Transaction, 8192),

		knownBlock:       knownBlockCache,
		knownTransaction: knownTxCache,

		txPool: config.TxPool,
		chain:  config.Chain,
	}

	go handler.broadcastBlock()
	go handler.broadcastTransaction()
	go handler.packageBlockRoutine()
	handlerInst = handler

	return handler, nil
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
			//log.Debugln("Start package block.")
			latest, _ := h.chain.GetLatestBlock()

			if latest == nil {
				log.Debugln("Waiting for genesis block.")
				continue
			}

			txs := h.txPool.Package()
			log.Debugf("Package %d txs.", len(txs))
			newBlock, err := h.chain.PackageNewBlock(txs)

			if err != nil {
				log.WithField("error", err).Errorln("Package new block failed.")
				continue
			}

			log.Debugf("Package new block# 0x%s", hex.EncodeToString(newBlock.Header.BlockHash[:]))
			h.chain.AppendBlockTask(newBlock)
			h.blockBroadcastQueue <- newBlock
		}
	}
}

func (h *Handler) NewPeer(peerId peer.ID, s *network.Stream) (*Peer, error) {
	config := PeerConfig{
		chain:   h.chain,
		txPool:  h.txPool,
		handler: h,
	}

	p, err := NewPeer(peerId, s, config)

	if err != nil {
		log.WithField("error", err).Errorln("Create peer failed.")
		return nil, err
	}

	h.peerSet = append(h.peerSet, p)
	return p, err
}

func (h *Handler) isKnownTransaction(hash common.Hash) bool {
	if h.txPool.Contain(hash) {
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
			countBroadcastBlock := int(math.Sqrt(float64(peersCount)))

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
	for {
		select {
		case tx := <-h.txBroadcastQueue:
			txHash := tx.Body.Hash
			peers := h.getPeersWithoutTransaction(txHash)

			peersCount := len(peers)
			countBroadcastBody := int(math.Sqrt(float64(peersCount)))

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
	list := make([]*Peer, 0, len(h.peerSet))
	for idx := range h.peerSet {
		peer := h.peerSet[idx]
		if !peer.KnownBlock(blockHash) {
			list = append(list, peer)
		}
	}
	return list
}

func (h *Handler) getPeersWithoutTransaction(txHash common.Hash) []*Peer {
	list := make([]*Peer, 0, len(h.peerSet))
	for idx := range h.peerSet {
		peer := h.peerSet[idx]
		if !peer.KnownTransaction(txHash) {
			list = append(list, peer)
		}
	}
	return list
}

func (h *Handler) markBlock(hash common.Hash) {
	h.knownBlock.Add(hash, nil)
}

func handleStatusMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	height := int(binary.LittleEndian.Uint64(payload))

	if height > h.chain.Height() {
		requestBlockWithHeight(h.chain.Height()+1, p)
	}
}

func handleNewBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	block, err := utils.DeserializeBlock(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Deserialize block from bytes failed.")
		return
	}

	blockHash := block.Header.BlockHash
	if h.knownBlock.Contains(blockHash) {
		return
	}

	h.markBlock(blockHash)
	p.MarkBlock(blockHash)

	if block.Header.Height == 0 {
		h.chain.InsertBlock(block)
	} else {
		h.chain.AppendBlockTask(block)
	}

	h.blockBroadcastQueue <- block
}

func handleNewBlockHashMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	blockHash := [32]byte(payload)

	if h.knownBlock.Contains(blockHash) {
		return
	}

	go requestBlockWithHash(blockHash, p)
}

func handleBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	block, err := utils.DeserializeBlock(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Deserialize block from bytes failed.")
		return
	}

	blockHash := block.Header.BlockHash
	h.markBlock(blockHash)
	p.MarkBlock(blockHash)

	if block.Header.Height == 0 {
		h.chain.InsertBlock(block)
	} else {
		h.chain.AppendBlockTask(block)
	}
}

func handleTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	transaction, err := utils.DeserializeTransaction(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Deserializer transaction failed.")
		return
	}

	txHash := transaction.Body.Hash

	if h.isKnownTransaction(txHash) {
		return
	}

	h.txPool.Add(transaction)
	h.txBroadcastQueue <- transaction
}

func handleNewPooledTransactionHashesMsg(h *Handler, msg *p2p.Message, p *Peer) {
	txHash := common.Hash(msg.Payload)

	if h.isKnownTransaction(txHash) {
		return
	}

	go requestTransactionWithHash(txHash, p)
}

func handleGetPooledTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	txHash := common.Hash(msg.Payload)

	tx := h.txPool.Get(txHash)

	go respondGetPooledTransaction(tx, p)
}

func handleGetBlockByHeightMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	height := int(binary.LittleEndian.Uint64(payload))

	block, err := h.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Debugln("Get block with height failed.")
		return
	}

	go respondGetBlockByHeight(block, p)
}
