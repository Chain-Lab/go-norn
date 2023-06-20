package utils

import (
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/p2p"
	karmem "karmem.org/golang"
)

// 这里有一个坑，writer 会对底层的bytes重复写入，需要规避这里的问题，不用writepool
//var (
//	writePool = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}
//)

//type CoreTypes interface {
//	common.Block | common.BlockHeader |
//		common.GeneralParams | common.GeneralParams
//}
//
//type P2PTypes interface {
//	p2p.SyncStatusMsg
//}
//
//type KarmemTypesInterface interface {
//	CoreTypes | P2PTypes
//	ReadAsRoot(reader *karmem.Reader)
//}
//
//type KarmemType[T KarmemTypesInterface] struct{}

func SerializeBlock(block *common.Block) ([]byte, error) {
	writer := karmem.NewWriter(KARMEM_CAP)

	_, err := block.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Block serialize failed.")
		return nil, err
	}

	result := writer.Bytes()

	return result, nil
}

func SerializeTransaction(transaction *common.Transaction) ([]byte, error) {
	writer := karmem.NewWriter(KARMEM_CAP)

	_, err := transaction.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Transaction serialize failed.")
		return nil, err
	}

	result := writer.Bytes()

	return result, nil
}

func SerializeBlockHeader(header *common.BlockHeader) ([]byte, error) {
	writer := karmem.NewWriter(KARMEM_CAP)

	_, err := header.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Transaction serialize failed.")
		return nil, err
	}

	result := writer.Bytes()

	return result, nil
}

func SerializeStatusMsg(msg *p2p.SyncStatusMsg) ([]byte, error) {
	writer := karmem.NewWriter(KARMEM_CAP)

	_, err := msg.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Message serialize failed.")
		return nil, err
	}

	result := writer.Bytes()

	return result, nil
}

func SerializeGenesisParams(p *common.GenesisParams) ([]byte, error) {
	writer := karmem.NewWriter(KARMEM_CAP)

	_, err := p.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Genesis serialize failed.")
		return nil, err
	}

	result := writer.Bytes()
	return result, nil
}

func SerializeGeneralParams(p *common.GeneralParams) ([]byte, error) {
	writer := karmem.NewWriter(KARMEM_CAP)

	_, err := p.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("General serialize failed.")
		return nil, err
	}

	result := writer.Bytes()
	return result, nil
}
