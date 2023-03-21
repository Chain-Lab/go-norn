package node

import (
	"encoding/hex"
	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/core"
	"go-chronos/p2p"
)

const (
	maxKnownTxs        = 32768
	maxKnownBlocks     = 1024
	maxQueuedTxs       = 4096
	maxQueuedTxAnns    = 4096
	maxQueuedBlocks    = 4
	maxQueuedBlockAnns = 4
)

type PeerConfig struct {
	chain   *core.BlockChain
	txPool  *core.TxPool
	handler *Handler
}

type Peer struct {
	peer *p2p.Peer

	knownBlocks     *lru.Cache
	queuedBlocks    chan *common.Block
	queuedBlockAnns chan common.Hash

	chain       *core.BlockChain
	txPool      *core.TxPool
	handler     *Handler
	knownTxs    *lru.Cache
	txBroadcast chan *common.Transaction
	txAnnounce  chan common.Hash

	msgQueue chan *p2p.Message
	// todo: 这里是传值还是需要传指针用于构建 channel？
	// todo： 还需要将区块、交易传出给上层结构处理的管道
}

func NewPeer(peerId peer.ID, s *network.Stream, config PeerConfig) (*Peer, error) {
	msgQueue := make(chan *p2p.Message)
	pp, err := p2p.NewPeer(peerId, s, msgQueue)

	if err != nil {
		log.WithField("error", err).Errorln("Create p2p peer failed.")
		return nil, err
	}

	blockLru, err := lru.New(maxKnownBlocks)

	if err != nil {
		log.WithField("error", err).Errorln("Create block lru failed.")
		return nil, err
	}

	txLru, err := lru.New(maxKnownTxs)

	if err != nil {
		log.WithField("error", err).Errorln("Create transaction lru failed")
		return nil, err
	}

	p := &Peer{
		peer:            pp,
		knownBlocks:     blockLru,
		queuedBlocks:    make(chan *common.Block, maxQueuedBlocks),
		queuedBlockAnns: make(chan common.Hash, maxQueuedBlockAnns),
		chain:           config.chain,
		txPool:          config.txPool,
		handler:         config.handler,
		knownTxs:        txLru,
		txBroadcast:     make(chan *common.Transaction, maxQueuedTxs),
		txAnnounce:      make(chan common.Hash, maxQueuedTxAnns),
		msgQueue:        msgQueue,
	}

	go p.broadcastBlock()
	go p.broadcastBlockHash()
	go p.broadcastTransaction()
	go p.broadcastTxHash()
	go p.sendStatus()
	go p.Handle()

	return p, nil
}

func (p *Peer) RunPeer() {

}

func (p *Peer) MarkBlock(blockHash string) {
	p.knownBlocks.Add(blockHash, nil)
}

func (p *Peer) MarkTransaction(txHash string) {
	p.knownTxs.Add(txHash, nil)
}

func (p *Peer) KnownBlock(blockHash string) bool {
	return p.knownBlocks.Contains(blockHash)
}

func (p *Peer) KnownTransaction(txHash string) bool {
	return p.knownTxs.Contains(txHash)
}

func (p *Peer) AsyncSendNewBlock(block *common.Block) {
	blockHash := block.Header.BlockHash
	strHash := hex.EncodeToString(blockHash[:])
	p.MarkBlock(strHash)
	p.queuedBlocks <- block
}

func (p *Peer) AsyncSendNewBlockHash(blockHash common.Hash) {
	p.queuedBlockAnns <- blockHash
}

func (p *Peer) AsyncSendTransaction(tx *common.Transaction) {
	txHash := tx.Body.Hash
	strHash := hex.EncodeToString(txHash[:])
	p.MarkTransaction(strHash)
	p.txBroadcast <- tx
}

func (p *Peer) AsyncSendTxHash(txHash common.Hash) {
	p.txAnnounce <- txHash
}

func (p *Peer) Handle() {
	for {
		select {
		case msg := <-p.msgQueue:
			if msg.Code != p2p.StatusCodeTransactionsMsg {
				log.Infoln("Receive code ", msg.Code)
			}
			handle := handlerMap[msg.Code]

			if handle != nil {
				go handle(p.handler, msg, p)
			}
		}
	}
}