// Package node
// @Description: 节点发送的请求方法
package node

import (
	"encoding/binary"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/p2p"
	"github.com/chain-lab/go-norn/utils"
)

func requestBlockWithHash(blockHash common.Hash, p *Peer) {
	p.peer.Send(p2p.StatusCodeGetBlockBodiesMsg, blockHash[:])
}

func requestSyncStatusMsg(height int64, p *Peer) {
	byteHeight := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteHeight, uint64(height))

	p.peer.Send(p2p.StatusCodeSyncStatusReq, byteHeight)
}

func requestSyncGetBlock(height int64, p *Peer) {
	byteHeight := make([]byte, 8)

	binary.LittleEndian.PutUint64(byteHeight, uint64(height))

	p.peer.Send(p2p.StatusCodeSyncGetBlocksMsg, byteHeight)
}

func requestTimeSync(msg *p2p.TimeSyncMsg, p *Peer) {
	byteTimeSyncMsg, err := utils.SerializeTimeSyncMsg(msg)

	if err != nil {
		return
	}

	p.peer.Send(p2p.StatusCodeTimeSyncReq, byteTimeSyncMsg)
}
