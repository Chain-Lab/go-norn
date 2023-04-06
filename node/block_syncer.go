/**
  @author: decision
  @date: 2023/3/13
  @note: 同步器，在完成同步之前管理所有的对端节点，进行区块数据的同步，在完成同步后才将管理权交给 backend
**/

package node

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"go-chronos/common"
	"go-chronos/core"
	"go-chronos/p2p"
	"sync"
	"time"
)

const (
	syncPaused    uint8 = 0x00 // 同步状态为暂停
	blockSyncing  uint8 = 0x01 // 同步区块中，此时关闭缓冲区的接收
	bufferSyncing uint8 = 0x02 // 同步缓冲区，开放缓冲区的接收
	synced        uint8 = 0x03 // 同步完成，到达缓冲高度，加入共识
)

const (
	blockNoneMark      uint8 = 0x00 // 已发送区块请求，但是还没有回复
	blockGottenMark    uint8 = 0x01 // 已经得到区块
	blockRequestedMark uint8 = 0x02 // 没有发送区块的请求
)

const (
	checkInterval        = 1 * time.Second
	requestBlockInterval = 3 * time.Second
)

type BlockSyncerConfig struct {
	Chain *core.BlockChain
}

type BlockSyncer struct {
	peerSet []*Peer

	remoteHeight     int64               // 目前其他节点的最高高度
	knownHeight      int64               // 目前已知节点中的最新高度
	requestTimestamp map[int64]time.Time // 区块请求的时间戳
	blockMap         map[int64]*common.Block
	peerReqTime      map[peer.ID]time.Time

	chain          *core.BlockChain
	status         uint8
	statusMsg      chan *p2p.SyncStatusMsg
	lock           sync.RWMutex
	peerStatusLock sync.RWMutex
}

func NewBlockSyncer(config *BlockSyncerConfig) *BlockSyncer {
	syncer := BlockSyncer{
		peerSet: make([]*Peer, 0, 40),

		chain:  config.Chain,
		status: syncPaused,
	}

	return &syncer
}

// Run 同步协程，每秒触发检查是否有空闲的 peer，如果有，就由该 peer 去拉取区块
func (bs *BlockSyncer) Run() {
	ticker := time.NewTicker(checkInterval)
	for {
		select {
		case <-ticker.C:
			bs.peerStatusLock.RLock()
			for idx := range bs.peerSet {
				p := bs.peerSet[idx]
				id := p.peerID
				if time.Since(bs.peerReqTime[id]) < requestBlockInterval {
					continue
				}

				height := bs.selectBlockHeight()

				if height < 0 {
					continue
				}

				go requestSyncGetBlock(height, p)
			}
			bs.peerStatusLock.RUnlock()
		}

		// 关闭，不再主动拉取
		if bs.status == bufferSyncing {
			break
		}
	}
}

func (bs *BlockSyncer) AddPeer(p *Peer) {
	bs.peerStatusLock.RLock()
	defer bs.peerStatusLock.RUnlock()

	bs.peerSet = append(bs.peerSet, p)
	bs.peerReqTime[p.peerID] = time.Now()
}

func (bs *BlockSyncer) appendStatusMsg(msg *p2p.SyncStatusMsg) {
	bs.statusMsg <- msg
}

// statusMsgRoutine 状态信息处理协程，获取状态信息队列中的信息， 然后计算对端最高高度
func (bs *BlockSyncer) statusMsgRoutine() {
	for {
		select {
		case msg := <-bs.statusMsg:
			height := msg.LatestHeight
			bs.lock.RLock()
			bs.remoteHeight = max(height, bs.remoteHeight)

			if bs.remoteHeight == bs.knownHeight {
				bs.status = bufferSyncing
			}

			bs.lock.RUnlock()
		}
	}
}

// blockProcessRoutine 区块处理协程， 每 10s 取出map中的区块加入到 chain 中
func (bs *BlockSyncer) blockProcessRoutine() {
	ticker := time.NewTicker(checkInterval)
	for {
		select {
		case <-ticker.C:
			bs.lock.RLock()
			knownHeight := bs.knownHeight + 1
			remoteHeight := bs.remoteHeight
			bs.lock.RUnlock()

			for height := knownHeight; height <= remoteHeight; height++ {
				if bs.blockMap[height] == nil {
					bs.insertBlock(height)
				} else {
					break
				}
			}
		}

		// 如果到了同步buffer的阶段， 退出该协程
		if bs.status == bufferSyncing {
			break
		}
	}
}

func (bs *BlockSyncer) selectBlockHeight() int64 {
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	for height := bs.knownHeight + 1; height <= bs.remoteHeight; height++ {
		if bs.blockMap[height] == nil || time.Since(bs.requestTimestamp[height]) > requestBlockInterval {
			bs.requestTimestamp[height] = time.Now()
			return height
		}
	}

	return -1
}

func (bs *BlockSyncer) appendBlock(block *common.Block) {
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	blockHeight := block.Header.Height
	bs.blockMap[blockHeight] = block
}

func (bs *BlockSyncer) insertBlock(height int64) {
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	block, ok := bs.blockMap[height]

	if !ok {
		return
	}

	bs.chain.InsertBlock(block)
	bs.knownHeight = height

	delete(bs.requestTimestamp, height)
	delete(bs.blockMap, height)
}

func max(h1 int64, h2 int64) int64 {
	if h1 > h2 {
		return h1
	} else {
		return h2
	}
}
