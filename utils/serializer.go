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
	defer writePool.Put(writer)

	_, err := block.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Debugln("Block serialize failed.")
		return nil, err
	}

	result := writer.Bytes()
	writer.Reset()
	writePool.Put(result)

	return result, nil
}
