package utils

import (
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/p2p"
	karmem "karmem.org/golang"
)

// DeserializeBlock
//
//	@Description: 区块反序列化函数，将 bytes 数据反序列化为 karmem 的 block
//	@param byteBlockData
//	@return *common.Block - 如果序列化失败返回 nil
//	@return error - 该版本默认返回 nil
func DeserializeBlock(byteBlockData []byte) (*common.Block, error) {
	block := new(common.Block)
	block.ReadAsRoot(karmem.NewReader(byteBlockData))

	return block, nil
}

// DeserializeTransaction
//
//	@Description: 交易的反序列化函数，将 bytes 数据反序列化为 karmem 的 transaction
//	@param byteTxData
//	@return *common.Transaction - 如果序列化失败返回 nil
//	@return error - 该版本默认返回 nil
func DeserializeTransaction(byteTxData []byte) (*common.Transaction, error) {
	transaction := new(common.Transaction)
	transaction.ReadAsRoot(karmem.NewReader(byteTxData))

	return transaction, nil
}

func DeserializeBroadcastMessage(byteTxData []byte) (*p2p.BroadcastMessage, error) {
	transaction := new(p2p.BroadcastMessage)
	transaction.ReadAsRoot(karmem.NewReader(byteTxData))

	return transaction, nil
}

func DeserializeStatusMsg(byteMsgData []byte) (*p2p.SyncStatusMsg, error) {
	msg := new(p2p.SyncStatusMsg)
	msg.ReadAsRoot(karmem.NewReader(byteMsgData))

	return msg, nil
}

func DeserializeGeneralParams(byteParamsData []byte) (*common.GeneralParams, error) {
	generalParams := new(common.GeneralParams)
	generalParams.ReadAsRoot(karmem.NewReader(byteParamsData))

	return generalParams, nil
}

func DeserializeGenesisParams(byteParamsData []byte) (*common.GenesisParams, error) {
	genesisParams := new(common.GenesisParams)
	genesisParams.ReadAsRoot(karmem.NewReader(byteParamsData))

	return genesisParams, nil
}

func DeserializeTimeSyncMsg(byteTimeSyncMsg []byte) (*p2p.TimeSyncMsg, error) {
	msg := new(p2p.TimeSyncMsg)
	msg.ReadAsRoot(karmem.NewReader(byteTimeSyncMsg))

	return msg, nil
}

func DeserializeDataCommand(byteDataCommand []byte) (*common.DataCommand, error) {
	dc := new(common.DataCommand)
	dc.ReadAsRoot(karmem.NewReader(byteDataCommand))

	return dc, nil
}
