/**
  @author: decision
  @date: 2023/3/15
  @note: 一系列的消息处理函数
**/

package node

import (
	"encoding/binary"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/p2p"
	"go-chronos/utils"
)

func handleStatusMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	height := int(binary.LittleEndian.Uint64(payload))

	log.Debugf("Remote height = %d.", height)
	//if height > h.chain.Height() {
	//	log.WithField("height", h.chain.Height()+1).Debugln("Request block.")
	//	requestBlockWithHeight(h.chain.Height()+1, p)
	//}
}

func handleNewBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	if h.syncStatus() != bufferSyncing || h.syncStatus() != synced {
		return
	}

	payload := msg.Payload
	block, err := utils.DeserializeBlock(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Deserialize block from bytes failed.")
		return
	}

	blockHash := block.Header.BlockHash
	strHash := hex.EncodeToString(blockHash[:])
	if h.knownBlock.Contains(strHash) {
		return
	}

	h.markBlock(strHash)
	p.MarkBlock(strHash)

	if block.Header.Height == 0 {
		go h.chain.InsertBlock(block)
	} else {
		h.chain.AppendBlockTask(block)
	}

	h.blockBroadcastQueue <- block
}

func handleNewBlockHashMsg(h *Handler, msg *p2p.Message, p *Peer) {
	if h.syncStatus() != bufferSyncing || h.syncStatus() != synced {
		return
	}

	payload := msg.Payload
	blockHash := [32]byte(payload)

	if h.knownBlock.Contains(blockHash) {
		return
	}

	go requestBlockWithHash(blockHash, p)
}

func handleBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	if h.syncStatus() != synced {
		return
	}

	payload := msg.Payload
	block, err := utils.DeserializeBlock(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Deserialize block from bytes failed.")
		return
	}

	blockHash := block.Header.BlockHash
	strHash := hex.EncodeToString(blockHash[:])
	h.markBlock(strHash)
	p.MarkBlock(strHash)

	log.WithField("height", block.Header.Height).Infoln("Receive block.")

	if block.Header.Height == 0 {
		go h.chain.InsertBlock(block)
	} else {
		h.chain.AppendBlockTask(block)
	}
}

func handleTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	if h.syncStatus() != synced {
		return
	}

	payload := msg.Payload
	transaction, err := utils.DeserializeTransaction(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Deserializer transaction failed.")
		return
	}

	txHash := hex.EncodeToString(transaction.Body.Hash[:])

	if h.isKnownTransaction(transaction.Body.Hash) {
		return
	}

	p.MarkTransaction(txHash)
	h.markTransaction(txHash)
	h.txPool.Add(transaction)
	h.txBroadcastQueue <- transaction
}

func handleNewPooledTransactionHashesMsg(h *Handler, msg *p2p.Message, p *Peer) {
	if h.syncStatus() != synced {
		return
	}

	txHash := common.Hash(msg.Payload)
	if h.isKnownTransaction(txHash) {
		return
	}

	go requestTransactionWithHash(txHash, p)
}

func handleGetPooledTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	if h.syncStatus() != synced {
		return
	}

	txHash := common.Hash(msg.Payload)
	strHash := hex.EncodeToString(txHash[:])

	tx := h.txPool.Get(strHash)

	go respondGetPooledTransaction(tx, p)
}

//func handleGetBlockByHeightMsg(h *Handler, msg *p2p.Message, p *Peer) {
//	if h.syncStatus() != synced {
//		return
//	}
//
//	payload := msg.Payload
//	height := int(binary.LittleEndian.Uint64(payload))
//
//	block, err := h.chain.GetBlockByHeight(height)
//	if err != nil {
//		log.WithField("error", err).Debugln("Get block with height failed.")
//		return
//	}
//
//	//log.Infof("Send block to peer.")
//	go respondGetBlockByHeight(block, p)
//}

func handleSyncStatusReq(h *Handler, msg *p2p.Message, p *Peer) {
	message := h.StatusMessage()
	go respondGetSyncStatus(message, p)
}

func handleSyncStatusMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload

	statusMessage, _ := utils.DeserializeStatusMsg(payload)
	h.blockSyncer.appendStatusMsg(statusMessage)
}
