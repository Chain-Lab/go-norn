package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/utils"
	"testing"
	"time"
)

// 测试插入区块到数据库的耗时
// 单次测试 3000 笔交易的交易耗时 58ms
func TestBlockChain_InsertBlock(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	txs := make([]common.Transaction, 0, 3000)

	for i := 0; i < 3000; i++ {
		tx := buildTransaction(privateKey)
		txs = append(txs, *tx)
	}

	db, err := utils.NewLevelDB("./data")

	if err != nil {
		log.WithField("error", err).Errorln("Create or load database failed.")
		return
	}
	bc := NewBlockchain(db)
	bc.NewGenesisBlock()
	block, _ := bc.PackageNewBlock([]common.Transaction{})
	println(hex.EncodeToString(block.Header.BlockHash[:]))

	startTime := time.Now()
	bc.InsertBlock(block)

	if err != nil {
		t.Fatal(err)
	}
	timeUsed := time.Since(startTime)

	t.Logf("Insert block use %d ms", timeUsed.Milliseconds())
}
