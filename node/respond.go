package node

import (
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/p2p"
	"go-chronos/utils"
)

func respondGetPooledTransaction(tx *common.Transaction, p *Peer) {
	bytesTransactionData, err := utils.SerializeTransaction(tx)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize transaction to bytes failed.")
		return
	}

	p.peer.Send(p2p.StatusCodePooledTransactionsMsg, bytesTransactionData)
}

func respondGetBlockByHeight(block *common.Block, p *Peer) {
	bytesBlockData, err := utils.SerializeBlock(block)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize block to bytes failed.")
		return
	}

	p.peer.Send(p2p.StatusCodeBlockBodiesMsg, bytesBlockData)
}

func respondGetSyncStatus(msg *p2p.SyncStatusMsg, p *Peer) {
	byteStatusMsg, err := utils.SerializeStatusMsg(msg)

	if err != nil {
		return
	}

	p.peer.Send(p2p.StatusCodeSyncStatusReq, byteStatusMsg)
}
