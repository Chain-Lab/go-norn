package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"go-chronos/common"
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

	txs := make([]common.Transaction, 3000)

	for i := 0; i < 3000; i++ {
		tx := buildTransaction(privateKey)
		txs = append(txs, *tx)
	}

	bc := GetBlockChainInst()
	block, _ := bc.PackageNewBlock(txs)

	startTime := time.Now()
	err = bc.InsertBlock(block)

	if err != nil {
		t.Fatal(err)
	}
	timeUsed := time.Since(startTime)

	t.Logf("Insert block use %d ms", timeUsed.Milliseconds())
}
