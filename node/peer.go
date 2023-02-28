package node

import (
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

type Peer struct {
	peer *p2p.Peer

	knownBlocks     *lru.Cache
	queuedBlocks    chan *common.Block
	queuedBlockAnns chan *common.Block

	txpool      *core.TxPool
	knownTxs    *lru.Cache
	txBroadcast chan []common.Hash
	txAnnounce  chan []common.Hash
}

func NewPeer(peerId peer.ID, s *network.Stream, txpool *core.TxPool) *Peer {
	pp, err := p2p.NewPeer(peerId, s)

	if err != nil {
		log.WithField("error", err).Errorln("Create p2p peer failed.")
		return nil
	}

	blockLru, err := lru.New(maxKnownBlocks)

	if err != nil {
		log.WithField("error", err).Errorln("Create block lru failed.")
		return nil
	}

	txLru, err := lru.New(maxKnownTxs)

	if err != nil {
		log.WithField("error", err).Errorln("Create transaction lru failed")
		return nil
	}

	p := &Peer{
		peer:            pp,
		knownBlocks:     blockLru,
		queuedBlocks:    make(chan *common.Block, maxQueuedBlocks),
		queuedBlockAnns: make(chan *common.Block, maxQueuedBlockAnns),
		txpool:          txpool,
		knownTxs:        txLru,
		txBroadcast:     make(chan []common.Hash, maxQueuedTxs),
		txAnnounce:      make(chan []common.Hash, maxQueuedTxAnns),
	}

	return p
}

func (p *Peer) RunPeer() {
	
}
