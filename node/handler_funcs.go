/**
  @author: decision
  @date: 2023/3/15
  @note: 一系列的消息处理函数 handlers
**/

package node

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/crypto"
	"github.com/chain-lab/go-chronos/metrics"
	"github.com/chain-lab/go-chronos/p2p"
	"github.com/chain-lab/go-chronos/utils"
	log "github.com/sirupsen/logrus"
	"math/big"
)

func handleStatusMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	height := int64(binary.LittleEndian.Uint64(payload))

	log.Debugf("Remote height = %d.", height)
	//if height > pm.chain.Height() {
	//	log.WithField("height", pm.chain.Height()+1).Debugln("Request block.")
	//	requestBlockWithHeight(pm.chain.Height()+1, p)
	//}
}

// handleNewBlockMsg 接收对端节点的新区块
func handleNewBlockMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	status := pm.blockSyncer.getStatus()
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
	if pm.knownBlock.Contains(strHash) {
		return
	}

	pm.markBlock(strHash)
	p.MarkBlock(strHash)

	if block.Header.Height == 0 {
		metrics.RoutineCreateCounterObserve(18)
		go pm.chain.InsertBlock(block)
		return
	}

	if verifyBlockVRF(block) {
		log.WithField("status", status).Debugln("Receive block from p2p.")
		pm.chain.AppendBlockTask(block)
		pm.blockBroadcastQueue <- block
	} else {
		//log.Infoln(hex.EncodeToString(block.Header.PublicKey[:]))
		log.Warning("Block VRF verify failed.")
	}
}

func handleNewBlockHashMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	status := pm.blockSyncer.getStatus()
	if status == blockSyncing || status == syncPaused {
		return
	}

	payload := msg.Payload
	blockHash := [32]byte(payload)

	if pm.knownBlock.Contains(blockHash) {
		return
	}

	metrics.RoutineCreateCounterObserve(19)
	go requestBlockWithHash(blockHash, p)
}

func handleBlockMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	status := pm.blockSyncer.getStatus()
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
	pm.markBlock(strHash)
	p.MarkBlock(strHash)

	//log.WithField("height", block.Header.Height).Infoln("Receive block.")

	if block.Header.Height == 0 {
		metrics.RoutineCreateCounterObserve(20)
		go pm.chain.InsertBlock(block)
		return
	}

	if verifyBlockVRF(block) {
		pm.chain.AppendBlockTask(block)
	}
}

func handleGetBlockBodiesMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	status := pm.blockSyncer.getStatus()
	if status != synced || status != bufferSyncing {
		return
	}

	blockHash := common.Hash(msg.Payload)

	block := pm.chain.GetBlockFromBuffer(blockHash)
	if block == nil {
		log.Debugln("Get block by hash failed")
		return
	}

	metrics.RoutineCreateCounterObserve(30)
	respondGetBlockBodies(block, p)
}

func handleSyncStatusReq(pm *P2PManager, msg *p2p.Message, p *Peer) {
	message := pm.StatusMessage()

	metrics.RoutineCreateCounterObserve(23)
	respondGetSyncStatus(message, p)
}

func handleSyncStatusMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	payload := msg.Payload

	statusMessage, _ := utils.DeserializeStatusMsg(payload)
	pm.blockSyncer.appendStatusMsg(statusMessage)
}

// handleSyncGetBlocksMsg 处理获取某个高度的区块
func handleSyncGetBlocksMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	status := pm.blockSyncer.getStatus()
	if status != synced {
		return
	}

	// 从消息中直接转换得到需要的区块高度
	payload := msg.Payload
	height := int64(binary.LittleEndian.Uint64(payload))

	// 从链上获取到区块
	block, err := pm.chain.GetBlockByHeight(height)
	if err != nil {
		log.WithField("error", err).Debugln("Get block with height failed.")
		return
	}

	metrics.RoutineCreateCounterObserve(24)
	respondSyncGetBlock(block, p)
}

func handleSyncBlockMsg(pm *P2PManager, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	block, err := utils.DeserializeBlock(payload)

	if err != nil {
		log.WithField("error", err).Debugln("Block deserialize failed.")
		return
	}
	pm.appendBlockToSyncer(block)
	p.SetMarkSynced(false)
}

func handleTimeSyncReq(pm *P2PManager, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	tMsg, err := utils.DeserializeTimeSyncMsg(payload)
	tMsg.RecReqTime = pm.timeSyncer.GetLogicClock()

	if err != nil {
		log.WithError(err).Debugln("Time sync message deserialize failed.")
		return
	}

	pm.timeSyncer.ProcessSyncRequest(tMsg, p)
}

func handleTimeSyncRsp(pm *P2PManager, msg *p2p.Message, p *Peer) {
	payload := msg.Payload
	tMsg, err := utils.DeserializeTimeSyncMsg(payload)
	tMsg.RecRspTime = pm.timeSyncer.GetLogicClock()

	if err != nil {
		log.WithError(err).Warning("Time sync message deserialize failed.")
		return
	}

	pm.timeSyncer.ProcessSyncRespond(tMsg, p)
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
