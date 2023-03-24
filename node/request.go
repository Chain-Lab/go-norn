package node

import (
	"encoding/binary"
	"go-chronos/common"
	"go-chronos/p2p"
)

func requestBlockWithHash(blockHash common.Hash, p *Peer) {
	p.peer.Send(p2p.StatusCodeGetBlockBodiesMsg, blockHash[:])
}

func requestTransactionWithHash(txHash common.Hash, p *Peer) {
	p.peer.Send(p2p.StatusCodeGetPooledTransactionMsg, txHash[:])
}

func requestStatusMsg(height int64, p *Peer) {
	byteHeight := make([]byte, 8)
	binary.LittleEndian.PutUint64(byteHeight, uint64(height))

	p.peer.Send(p2p.StatusCodeStatusMsg, byteHeight)
}

func requestSyncGetBlock(height int64, p *Peer) {
	byteHeight := make([]byte, 8)

	binary.LittleEndian.PutUint64(byteHeight, uint64(height))

	p.peer.Send(p2p.StatusCodeSyncGetBlocksMsg, byteHeight)
}
