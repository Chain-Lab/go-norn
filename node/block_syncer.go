/**
  @author: decision
  @date: 2023/3/13
  @note: 同步器，在完成同步之前管理所有的对端节点，进行区块数据的同步，在完成同步后才将管理权交给 backend
**/

package node

import (
	lru "github.com/hashicorp/golang-lru"
	"go-chronos/core"
	"go-chronos/p2p"
	"sync"
)

const (
	syncPaused    uint8 = 0x00 // 同步状态为暂停
	blockSyncing  uint8 = 0x01 // 同步区块中
	bufferSyncing uint8 = 0x02 // 同步缓冲区
	synced        uint8 = 0x03 // 同步完成
)

const (
	blockNoneMark      uint8 = 0x00 // 已发送区块请求，但是还没有回复
	blockGottenMark    uint8 = 0x01 // 已经得到区块
	blockRequestedMark uint8 = 0x02 // 没有发送区块的请求
)

type BlockSyncerConfig struct {
	Chain *core.BlockChain
}

type BlockSyncer struct {
	peerSet []*Peer

	knownHeight      uint64           // 目前已知节点中的最新高度
	blockStatus      *lru.Cache       // 某个高度的区块状态
	requestTimestamp map[uint64]int64 // 区块请求的时间戳

	chain     *core.BlockChain
	status    uint8
	statusMsg chan *p2p.SyncStatusMsg
	lock      sync.RWMutex
}

func NewBlockSyncer(config *BlockSyncerConfig) *BlockSyncer {
	syncer := BlockSyncer{
		peerSet: make([]*Peer, 0, 40),

		chain:  config.Chain,
		status: syncPaused,
	}
	return &syncer
}

func Run() {

}

func (bs *BlockSyncer) appendStatusMsg(msg *p2p.SyncStatusMsg) {
	bs.statusMsg <- msg
}

func (bs *BlockSyncer) statusMsgRoutine() {
	for {
		select {
		case msg := <-bs.statusMsg:
			
		}
	}
}
