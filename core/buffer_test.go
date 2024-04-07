/**
  @author: decision
  @date: 2023/3/16
  @note:
**/

package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/utils"
	log "github.com/sirupsen/logrus"
	mrand "math/rand"
	"testing"
	"time"
)

func testCreateBlock(prev *common.Block, txs []common.Transaction) *common.Block {
	var prevBlockHash []byte
	var strPrevHash string
	var height int64
	if prev == nil {
		strPrevHash = "0000000000000000000000000000000000000000000000000000000000000000"
	} else {
		strPrevHash = prev.BlockHash()
	}

	prevBlockHash, _ = hex.DecodeString(strPrevHash)
	if prev == nil {
		height = 0
	} else {
		height = prev.Header.Height + 1
	}

	merkleRoot := BuildMerkleTree(txs)
	block := &common.Block{
		Header: common.BlockHeader{
			Timestamp:     time.Now().UnixMilli(),
			PrevBlockHash: [32]byte(prevBlockHash),
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(merkleRoot),
			Height:        height,
		},
		Transactions: txs,
	}

	byteBlockHeaderData, err := utils.SerializeBlockHeader(&block.Header)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize block header failed.")
		return nil
	}

	hash := sha256.New()
	hash.Write(byteBlockHeaderData)
	blockHash := common.Hash(hash.Sum(nil))
	block.Header.BlockHash = blockHash

	return block
}

func testCreateTransactionList() []common.Transaction {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	if err != nil {
		return nil
	}

	count := mrand.Intn(20) + 10
	res := make([]common.Transaction, 0, count)

	for i := 0; i < count; i++ {
		tx := buildTransaction(privateKey)
		res = append(res, *tx)
	}
	return res
}

// testRandomCreateLayer 随机创建一个高度下的区块，包括随机的交易和随机的区块数量
func testRandomCreateLayer(b *BlockBuffer) {
	counts := mrand.Intn(5) + 1
	best := b.GetPriorityLeaf(100)
	log.Tracef("Get prority leaf height = %d", best.Header.Height)

	blocks := make([]*common.Block, 0, counts)

	for i := 0; i < counts; i++ {
		txs := testCreateTransactionList()
		block := testCreateBlock(best, txs)
		//b.AppendBlock(block)
		blocks = append(blocks, block)
	}

	// 为了模拟乱序进入视图
	for i := 0; i < counts; i++ {
		j := mrand.Intn(i + 1)
		blocks[i], blocks[j] = blocks[j], blocks[i]
	}

	for idx := range blocks {
		b.AppendBlock(blocks[idx])
	}
}

func TestCreateGenesisBlock(t *testing.T) {
	genesisBlock := testCreateBlock(nil, nil)

	if genesisBlock == nil {
		t.Fatal("Create Genesis block failed.")
	}
}

func TestBlockBuffer(t *testing.T) {
	// upd(2023/3/23): 这里修改了推出区块的逻辑，测试代码未修改，可能无法通过测试
	log.SetLevel(log.TraceLevel)
	genesisBlock := testCreateBlock(nil, nil)
	buffer, err := NewBlockBuffer(genesisBlock, nil)
	go buffer.Process()
	go buffer.secondProcess()

	if err != nil {
		t.Fatal(err)
	}

	testRandomCreateLayer(buffer)

	// 测试插入一个区块到视图后的高度是否为 1
	time.Sleep(2 * time.Second)
	if buffer.bufferedHeight != 1 {
		log.WithField("height", buffer.bufferedHeight).Errorln("Append block failed.")
		t.Fatal("Append block to view failed.")
	}

	for i := 0; i < 10; i++ {
		testRandomCreateLayer(buffer)
		time.Sleep(1 * time.Second)
	}

	//time.Sleep(10 * time.Second)
	if buffer.bufferedHeight != 11 {
		t.Fatalf("Buffer height %d, except %d", buffer.bufferedHeight, 11)
	}

	block := buffer.PopSelectedBlock()
	//t.Log(buffer.bufferedHeight)
	if block == nil || buffer.latestBlockHeight != 1 {
		t.Fatal("Pop block from view failed.")
	}
}
