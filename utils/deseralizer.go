package utils

import (
	"go-chronos/common"
	"go-chronos/p2p"
	karmem "karmem.org/golang"
)

func DeserializeBlock(byteBlockData []byte) (*common.Block, error) {
	block := new(common.Block)
	block.ReadAsRoot(karmem.NewReader(byteBlockData))

	return block, nil
}

func DeserializeTransaction(byteTxData []byte) (*common.Transaction, error) {
	transaction := new(common.Transaction)
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
