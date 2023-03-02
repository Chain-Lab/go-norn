package node

import (
	"encoding/binary"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/core"
	"go-chronos/p2p"
	"go-chronos/utils"
	"math"
)

type msgHandler func(h *Handler, msg *p2p.Message, p *Peer)

const (
	maxKnownBlock = 1024
)

var handlerMap = map[p2p.StatusCode]msgHandler{
	p2p.StatusCodeStatusMsg:                     handleStatusMsg,
	p2p.StatusCodeNewBlockMsg:                   handleNewBlockMsg,
	p2p.StatusCodeNewBlockHashesMsg:             handleNewBlockHashMsg,
	p2p.StatusCodeBlockBodiesMsg:                handleBlockMsg,
	p2p.StatusCodeTransactionsMsg:               handlerTransactionMsg,
	p2p.StatusCodeNewPooledTransactionHashesMsg: handlerNewPooledTransactionHashesMsg,
	p2p.StatusCodeGetPooledTransactionMsg:       handlerGetPooledTransactionMsg,
	p2p.StatusCodeGetBlockByHeightMsg:           handlerGetBlockByHeightMsg,
}

type HandlerConfig struct {
	txPool *core.TxPool
	chain  *core.BlockChain
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

func NewHandler() *Handler {
	knownBlockCache, err := lru.New(maxKnownBlock)

	if err != nil {
		log.WithField("error", err).Debugln("Create known block cache failed.")
	}

	handler := &Handler{
		// todo: 限制节点数量，Kad 应该限制了节点数量不超过20个
		peerSet:             make([]*Peer, 40),
		blockBroadcastQueue: make(chan *common.Block, 64),
		knownBlock:          knownBlockCache,
	}

	return handler
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
}

func handlerTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
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

func handlerNewPooledTransactionHashesMsg(h *Handler, msg *p2p.Message, p *Peer) {
	txHash := common.Hash(msg.Payload)

	if h.isKnownTransaction(txHash) {
		return
	}

	go requestTransactionWithHash(txHash, p)
}

func handlerGetPooledTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	txHash := common.Hash(msg.Payload)

	tx := h.txPool.Get(txHash)

	go respondGetPooledTransaction(tx, p)
}

func handlerGetBlockByHeightMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	height := int(binary.LittleEndian.Uint64(payload))

	block, err := h.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Debugln("Get block with height failed.")
		return
	}

	go respondGetBlockByHeight(block, p)
}
