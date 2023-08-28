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
	"go-chronos/crypto"
	"go-chronos/metrics"
	"go-chronos/p2p"
	"go-chronos/utils"
	"math/big"
)

func handleStatusMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	height := int64(binary.LittleEndian.Uint64(payload))

	log.Debugf("Remote height = %d.", height)
	//if height > h.chain.Height() {
	//	log.WithField("height", h.chain.Height()+1).Debugln("Request block.")
	//	requestBlockWithHeight(h.chain.Height()+1, p)
	//}
}

// handleNewBlockMsg 接收对端节点的新区块
func handleNewBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	if status == blockSyncing || status == syncPaused {
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
		metrics.RoutineCreateHistogramObserve(18)
		go h.chain.InsertBlock(block)
		return
	}

	if verifyBlockVRF(block) {
		log.WithField("status", status).Debugln("Receive block from p2p.")
		h.chain.AppendBlockTask(block)
		h.blockBroadcastQueue <- block
	} else {
		//log.Infoln(hex.EncodeToString(block.Header.PublicKey[:]))
		log.Warning("Block VRF verify failed.")
	}
}

func handleNewBlockHashMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	if status == blockSyncing || status == syncPaused {
		return
	}

	payload := msg.Payload
	blockHash := [32]byte(payload)

	if h.knownBlock.Contains(blockHash) {
		return
	}

	metrics.RoutineCreateHistogramObserve(19)
	go requestBlockWithHash(blockHash, p)
}

func handleBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	log.WithField("status", status).Traceln("Receive block.")
	if status != synced {
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
		metrics.RoutineCreateHistogramObserve(20)
		go h.chain.InsertBlock(block)
		return
	}

	if verifyBlockVRF(block) {
		h.chain.AppendBlockTask(block)
	}
}

func handleTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	if status != synced {
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
	status := h.blockSyncer.getStatus()
	// todo: 修改这里的条件判断为统一的函数
	if status != synced {
		return
	}

	txHash := common.Hash(msg.Payload)
	if h.isKnownTransaction(txHash) {
		return
	}

	metrics.RoutineCreateHistogramObserve(21)
	go requestTransactionWithHash(txHash, p)
}

func handleGetBlockBodiesMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	if status != synced {
		return
	}

	blockHash := common.Hash(msg.Payload)

	block, err := h.chain.GetBlockByHash(&blockHash)
	if err != nil {
		log.WithError(err).Debugln("Get block by hash failed")
		return
	}

	metrics.RoutineCreateHistogramObserve(30)
	go respondGetBlockBodies(block, p)
}
func handleGetPooledTransactionMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	if status != synced {
		return
	}

	txHash := common.Hash(msg.Payload)
	strHash := hex.EncodeToString(txHash[:])

	tx := h.txPool.Get(strHash)

	metrics.RoutineCreateHistogramObserve(22)
	go respondGetPooledTransaction(tx, p)
}

func handleSyncStatusReq(h *Handler, msg *p2p.Message, p *Peer) {
	message := h.StatusMessage()

	metrics.RoutineCreateHistogramObserve(23)
	go respondGetSyncStatus(message, p)
}

func handleSyncStatusMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload

	statusMessage, _ := utils.DeserializeStatusMsg(payload)
	h.blockSyncer.appendStatusMsg(statusMessage)
}

// handleSyncGetBlocksMsg 处理获取某个高度的区块
func handleSyncGetBlocksMsg(h *Handler, msg *p2p.Message, p *Peer) {
	status := h.blockSyncer.getStatus()
	if status != synced {
		return
	}

	// 从消息中直接转换得到需要的区块高度
	payload := msg.Payload
	height := int64(binary.LittleEndian.Uint64(payload))

	// 从链上获取到区块
	block, err := h.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Debugln("Get block with height failed.")
		return
	}

	metrics.RoutineCreateHistogramObserve(24)
	go respondSyncGetBlock(block, p)
}

func handleSyncBlockMsg(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	block, err := utils.DeserializeBlock(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Block deserialize failed.")
		return
	}
	h.appendBlockToSyncer(block)
}

func handleTimeSyncReq(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	tMsg, err := utils.DeserializeTimeSyncMsg(payload)
	tMsg.RecReqTime = h.timeSyncer.GetLogicClock()

	if err != nil {
		log.WithError(err).Debugln("Time sync message deserialize failed.")
		return
	}

	h.timeSyncer.ProcessSyncRequest(tMsg, p)
}

func handleTimeSyncRsp(h *Handler, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	tMsg, err := utils.DeserializeTimeSyncMsg(payload)
	tMsg.RecRspTime = h.timeSyncer.GetLogicClock()

	if err != nil {
		log.WithError(err).Warning("Time sync message deserialize failed.")
		return
	}

	h.timeSyncer.ProcessSyncRespond(tMsg, p)
}

func verifyBlockVRF(block *common.Block) bool {
	//println(hex.EncodeToString(block.Header.PublicKey[:]))
	bytesParams := block.Header.Params
	params, err := utils.DeserializeGeneralParams(bytesParams)

	// todo: 如果这里的数据不全，导致反序列化出错可能会使得这个区块无法正常添加
	if err != nil {
		log.WithField("error", err).Warning("Deserialize params failed.")
		return false
	}

	s := new(big.Int)
	t := new(big.Int)
	publicKey := crypto.Bytes2PublicKey(block.Header.PublicKey[:])

	s.SetBytes(params.S)
	t.SetBytes(params.T)

	verified, err := crypto.VRFCheckRemoteConsensus(publicKey, params.Result, s, t, params.RandomNumber[:])

	if err != nil || !verified {
		log.Debugln("Verify VRF failed.")
		//log.Infoln(hex.EncodeToString(s.Bytes()))
		//log.Infoln(hex.EncodeToString(t.Bytes()))
		//log.Infoln(hex.EncodeToString(params.Result))
		//log.Infoln(hex.EncodeToString(params.RandomNumber[:]))
		//log.Infoln(hex.EncodeToString(block.Header.PublicKey[:]))
		return false
	}

	return true
}
