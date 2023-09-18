package node

import (
	"encoding/binary"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/p2p"
	"github.com/chain-lab/go-chronos/utils"
)

func requestBlockWithHash(blockHash common.Hash, p *Peer) {
	p.peer.Send(p2p.StatusCodeGetBlockBodiesMsg, blockHash[:])
}

func requestTransactionWithHash(txHash common.Hash, p *Peer) {
	p.peer.Send(p2p.StatusCodeGetPooledTransactionMsg, txHash[:])
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
