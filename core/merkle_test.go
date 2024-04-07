package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"testing"
	"time"
)

// TestBuildMerkleTree 计算构建 Merkle 树所耗费的时间
func TestBuildMerkleTree(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		t.Fatal(err)
	}

	txs := make([]common.Transaction, 0, 3000)

	for i := 0; i < 3000; i++ {
		tx := buildTransaction(privateKey)
		txs = append(txs, *tx)
	}

	startTime := time.Now()
	merkleRoot := BuildMerkleTree(txs)
	usedTime := time.Since(startTime)

	t.Logf("Build merkle tree use %d μs.", usedTime.Microseconds())
	t.Logf("Merkle tree root is %s.", hex.EncodeToString(merkleRoot))
}
