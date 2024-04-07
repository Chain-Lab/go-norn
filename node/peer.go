package node

import (
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/core"
	"github.com/chain-lab/go-norn/metrics"
	"github.com/chain-lab/go-norn/p2p"
	lru "github.com/hashicorp/golang-lru"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	maxKnownTxs        = 32768
	maxKnownBlocks     = 1024
	maxQueuedTxs       = 4096
	maxQueuedTxAnns    = 4096
	maxQueuedBlocks    = 4
	maxQueuedBlockAnns = 4
	messageQueueCap    = 5000
)

type PeerConfig struct {
	chain          *core.BlockChain
	txPool         *core.TxPool
	handler        *P2PManager
	remoteMultAddr string
}

type Peer struct {
	peer   *p2p.Peer
	peerID peer.ID
	addr   string

	knownBlocks     *lru.Cache
	queuedBlocks    chan *common.Block
	queuedBlockAnns chan common.Hash

	chain       *core.BlockChain
	txPool      *core.TxPool
	handler     *P2PManager
	knownTxs    *lru.Cache
	txBroadcast chan *common.Transaction
	txAnnounce  chan common.Hash

	syncBlock chan *int64

	msgQueue chan *p2p.Message

	lock       sync.RWMutex
	markSynced bool
	// todo: 这里是传值还是需要传指针用于构建 channel？
	// todo： 还需要将区块、交易传出给上层结构处理的管道
}

// NewPeer
//
//	@Description: 新建节点实例，在接收到其它节点的连接建立请求或发现节点尝试建立连接时调用
//	@param peerId - 对端节点的 id
//	@param s - 数据流
//	@param config - 节点配置信息
//	@return *Peer - 节点实例，作为 manager 和 p2p.Peer 的中间层
//	@return error
func NewPeer(peerId peer.ID, s *network.Stream, config PeerConfig) (*Peer, error) {
	msgQueue := make(chan *p2p.Message, messageQueueCap)

	// 创建一个 p2p.Peer 实例，p2p.Peer 用于发送消息，并接收到的消息传递到 node.Peer
	pp, err := p2p.NewPeer(peerId, s, msgQueue)
	if err != nil {
		log.WithField("error", err).Errorln("Create p2p peer failed.")
		return nil, err
	}

	// 缓存初始化
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
		peerID:          pp.Id(),
		addr:            config.remoteMultAddr,
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
		markSynced:      false,
	}

	metrics.RoutineCreateCounterObserve(25)
	//go p.broadcastBlock()
	//go p.broadcastBlockHash()
	go p.sendStatus()
	go p.Handle()

	return p, nil
}

func (p *Peer) SetMarkSynced(v bool) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.markSynced = v
}

func (p *Peer) MarkSynced() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.markSynced
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

func (p *Peer) Stopped() bool {
	return p.peer.Stopped()
}

func (p *Peer) Handle() {
	for {
		if p.peer.Stopped() {
			break
		}

		select {
		case msg := <-p.msgQueue:
			handle := handlerMap[msg.Code]

			if handle != nil {
				metrics.RoutineCreateCounterObserve(17)
				metrics.RecordHandleReceivedCode(int(msg.Code))
				go handle(p.handler, msg, p)
			}
		}
	}
}

// sendStatus
//
//	@Description: 向其它节点请求同步信息
//	@receiver p
func (p *Peer) sendStatus() {
	ticker := time.NewTicker(490 * time.Millisecond)
	for {
		if p.peer.Stopped() {
			break
		}

		select {
		case <-ticker.C:
			height := p.chain.Height()
			requestSyncStatusMsg(height, p)
		}

		// fixed[230725]：在同步完成后不再请求对端高度
		status := p.handler.blockSyncer.getStatus()
		if status == synced {
			break
		}
	}
}
