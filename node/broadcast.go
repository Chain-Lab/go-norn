package node

import (
	"github.com/chain-lab/go-chronos/p2p"
	"github.com/chain-lab/go-chronos/utils"
	log "github.com/sirupsen/logrus"
	"time"
)

func (p *Peer) broadcastBlock() {
	for {
		if p.peer.Stopped() {
			break
		}

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
		if p.peer.Stopped() {
			break
		}

		select {
		case blockHash := <-p.queuedBlockAnns:
			p.peer.Send(p2p.StatusCodeNewBlockHashesMsg, blockHash[:])
		}
	}
}

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
		if p.handler.Synced() {
			break
		}
	}
}
