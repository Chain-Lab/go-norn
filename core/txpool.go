package core

import (
	"go-chronos/common"
	"sync"
)

var (
	txOnce     sync.Once
	txPoolInst *txPool
)

type txPool struct {
	txQueue chan *common.Hash
	txs     map[common.Hash]*common.Transaction
	lock    sync.RWMutex
}

func GetTxPool() *txPool {
	txOnce.Do(func() {
		txPoolInst = &txPool{
			txQueue: make(chan *common.Hash),
			txs:     make(map[common.Hash]*common.Transaction),
		}
	})
	return txPoolInst
}

// Package 用于打包交易，这里返回的是 Transaction 的切片
// todo： 需要具体观察打包交易时的效率问题
func (pool *txPool) Package() []common.Transaction {
	pool.lock.RLock()
	defer pool.lock.RUnlock()
}
