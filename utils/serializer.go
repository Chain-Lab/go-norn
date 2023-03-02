package utils

import (
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	karmem "karmem.org/golang"
	"sync"
)

var (
	writePool = sync.Pool{New: func() any { return karmem.NewWriter(1024) }}
)

func DeserializeBlock(byteBlockData []byte) (*common.Block, error) {
	block := new(common.Block)
	block.ReadAsRoot(karmem.NewReader(byteBlockData))

	return block, nil
}

func SerializeBlock(block *common.Block) ([]byte, error) {
	writer := writePool.Get().(*karmem.Writer)
	defer writer.Reset()
	defer writePool.Put(writer)

	_, err := block.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Block serialize failed.")
		return nil, err
	}

	result := writer.Bytes()

	return result, nil
}

func DeserializeTransaction(byteTxData []byte) (*common.Transaction, error) {
	transaction := new(common.Transaction)
	transaction.ReadAsRoot(karmem.NewReader(byteTxData))

	return transaction, nil
}

func SerializeTransaction(transaction *common.Transaction) ([]byte, error) {
	writer := writePool.Get().(*karmem.Writer)
	defer writer.Reset()
	defer writePool.Put(writer)

	_, err := transaction.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Transaction serialize failed.")
		return nil, err
	}

	result := writer.Bytes()

	return result, nil
}
