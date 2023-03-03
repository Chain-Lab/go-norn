package core

import (
	"go-chronos/common"
	"sync"
)

const (
	maxTxPackageCount = 3000
)

var (
	txOnce     sync.Once
	txPoolInst *TxPool
)

type TxPool struct {
	txQueue chan common.Hash
	txs     sync.Map
	flags   sync.Map
	height  int
	lock    sync.RWMutex
}

func NewTxPool() *TxPool {
	return &TxPool{
		txQueue: make(chan common.Hash, 8192),
		txs:     sync.Map{},
	}
}

// Package 用于打包交易，这里返回的是 Transaction 的切片
// todo： 需要具体观察打包交易时的效率问题
func (pool *TxPool) Package() []common.Transaction {
	pool.lock.RLock()
	defer pool.lock.RUnlock()

	count := 0
	result := make([]common.Transaction, 3000)

	for txHash := range pool.txQueue {
		value, hit := pool.txs.Load(txHash)
		if !hit {
			continue
		}

		tx := value.(*common.Transaction)
		if !tx.Verify() {
			continue
		}
		// todo： 这里是传值还是传指针？
		result = append(result, *tx)
		count++

		if count >= maxTxPackageCount-1 {
			break
		}
	}

	return result
}

func (pool *TxPool) Add(transaction *common.Transaction) {
	// todo: 还是需要和 Package 的锁相关联，保证 Package 能抢到锁
	// todo: 在当前版本下先直接加锁打包

	txHash := transaction.Body.Hash
	pool.txs.Store(txHash, transaction)
	pool.txQueue <- txHash
}

func (pool *TxPool) Contain(hash common.Hash) bool {
	_, hit := pool.txs.Load(hash)
	return hit
}

func (pool *TxPool) Get(hash common.Hash) *common.Transaction {
	value, hit := pool.txs.Load(hash)

	if !hit {
		return nil
	}

	return value.(*common.Transaction)
}
