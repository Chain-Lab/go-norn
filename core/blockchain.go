package core

import (
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/utils"
	karmem "karmem.org/golang"
	"time"
)

type BlockChain struct {
	dbWriterQueue chan *common.Block
}

// PackageNewBlock 打包新的区块，传入交易序列
func (bc *BlockChain) PackageNewBlock(txs []common.Transaction) (*common.Block, error) {
	merkleRoot := BuildMerkleTree(txs)
	block := common.Block{
		Header: common.BlockHeader{
			Timestamp:     uint64(time.Now().UnixMilli()),
			PrevBlockHash: [32]byte{},
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(merkleRoot),
			Height:        0,
		},
		Transactions: txs,
	}
	return &block, nil
}
func (bc *BlockChain) insertBlock(block *common.Block) error {
	writer := karmem.NewWriter(1024)
	_, err := block.WriteAsRoot(writer)
	if err != nil {
		log.WithField("error", err).Errorln("Encode block failed.")
		return err
	}
	count := len(block.Transactions)
	keys := make([][]byte, count+1)
	values := make([][]byte, count+1)

	values[0] = writer.Bytes()
	db := utils.GetLevelDBInst()

	keys[0] = append([]byte("block#"), block.Header.BlockHash[:]...)

	for idx := range block.Transactions {
		tx := block.Transactions[idx]

		txWriter := karmem.NewWriter(1024)
		keys[idx+1] = append([]byte("tx#"), tx.Body.Hash[:]...)
		_, err := tx.WriteAsRoot(txWriter)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"hash":  hex.EncodeToString(tx.Body.Hash[:]),
			}).Errorln("Encode transaction failed.")
			return err
		}

		values[idx+1] = txWriter.Bytes()
	}

	return db.BatchInsert(keys, values)
}

// databaseWriter 负责插入数据到数据库的协程
func (bc *BlockChain) databaseWriter() {
	select {
	case block := <-bc.dbWriterQueue:
		err := bc.insertBlock(block)

		if err != nil {
			log.WithField("error", err).Errorln("Insert block to database failed")
		}
	}
}
