package node

import (
	log "github.com/sirupsen/logrus"
	"go-chronos/p2p"
	"go-chronos/utils"
	"time"
)

func (p *Peer) broadcastBlock() {
	for {
		select {
		case block := <-p.queuedBlocks:
			byteBlockData, err := utils.SerializeBlock(block)

			if err != nil {
				log.WithField("error", err).Debugln("Serialize block failed.")
				continue
			}

			p.peer.Send(p2p.StatusCodeNewBlockMsg, byteBlockData)
		}
	}
}

func (p *Peer) broadcastBlockHash() {
	for {
		select {
		case blockHash := <-p.queuedBlockAnns:
			p.peer.Send(p2p.StatusCodeNewBlockHashesMsg, blockHash[:])
		}
	}
}

func (p *Peer) broadcastTransaction() {
	for {
		select {
		case tx := <-p.txBroadcast:
			// todo: 对序列化的一个优化的地方，添加 sync.Pool，需要针对不同场景来设计
			byteTxData, err := utils.SerializeTransaction(tx)

			if err != nil {
				log.WithField("error", err).Debugln("Serialize block failed.")
				break
			}
			//log.WithFields(log.Fields{
			//	"hash":      hex.EncodeToString(tx.Body.Hash[:]),
			//	"expire":    tx.Body.Expire,
			//	"data":      hex.EncodeToString(tx.Body.Data),
			//	"payload":   hex.EncodeToString(byteTxData),
			//	"timestamp": time.Now().Nanosecond(),
			//}).Infoln("Send transaction.")
			p.peer.Send(p2p.StatusCodeTransactionsMsg, byteTxData)
		}
	}
}

func (p *Peer) broadcastTxHash() {
	for {
		select {
		case txHash := <-p.txAnnounce:
			p.peer.Send(p2p.StatusCodeNewPooledTransactionHashesMsg, txHash[:])
		}
	}
}

func (p *Peer) sendStatus() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			height := p.chain.Height()

			requestSyncStatusMsg(height, p)
		}
	}
}
