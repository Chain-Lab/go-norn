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

			p.peer.Send(p2p.StatusCodeBlockBodiesMsg, byteBlockData)
		}
	}
}

func (p *Peer) broadcastBlockHash() {
	for {
		select {
		case blockHash := <-p.queuedBlockAnns:
			p.peer.Send(p2p.StatusCodeBlockBodiesMsg, blockHash[:])
		}
	}
}

func (p *Peer) broadcastTransaction() {
	for {
		select {
		case tx := <-p.txBroadcast:
			byteTxData, err := utils.SerializeTransaction(tx)

			if err != nil {
				log.WithField("error", err).Debugln("Serialize block failed.")
				continue
			}

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
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			height := p.chain.Height()

			requestStatusMsg(height, p)
		}
	}
}
