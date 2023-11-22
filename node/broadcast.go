package node

import (
	"time"
)

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
		status := p.handler.blockSyncer.getStatus()
		if status == synced {
			break
		}
	}
}
