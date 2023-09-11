package node

import (
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/p2p"
	"go-chronos/utils"
)

func respondGetBlockBodies(block *common.Block, p *Peer) {
	bytesBlockData, err := utils.SerializeBlock(block)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize block to bytes failed.")
		return
	}

	p.peer.Send(p2p.StatusCodeBlockBodiesMsg, bytesBlockData)
}
func respondGetPooledTransaction(tx *common.Transaction, p *Peer) {
	bytesTransactionData, err := utils.SerializeTransaction(tx)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize transaction to bytes failed.")
		return
	}

	p.peer.Send(p2p.StatusCodePooledTransactionsMsg, bytesTransactionData)
}

func respondSyncGetBlock(block *common.Block, p *Peer) {
	bytesBlockData, err := utils.SerializeBlock(block)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize block to bytes failed.")
		return
	}

	p.peer.Send(p2p.StatusCodeSyncBlocksMsg, bytesBlockData)
}

func respondGetSyncStatus(msg *p2p.SyncStatusMsg, p *Peer) {
	//metrics.RespondGetSyncStatusGauge.Inc()
	byteStatusMsg, err := utils.SerializeStatusMsg(msg)

	if err != nil {
		return
	}

	p.peer.Send(p2p.StatusCodeSyncStatusMsg, byteStatusMsg)
	//metrics.RespondGetSyncStatusGauge.Dec()
}

func respondTimeSync(msg *p2p.TimeSyncMsg, p *Peer) {
	//metrics.RespondTimeSyncRoutineGauge.Inc()
	byteTimeSyncMsg, err := utils.SerializeTimeSyncMsg(msg)

	if err != nil {
		return
	}

	p.peer.Send(p2p.StatusCodeTimeSyncRsp, byteTimeSyncMsg)
	//metrics.RespondTimeSyncRoutineGauge.Dec()
}
