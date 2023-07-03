package core

import (
	"encoding/hex"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"go-chronos/metrics"
	"sync"
)

const (
	maxTxPackageCount = 8001 // 交易池打包的最多交易数量
)

var (
	txOnce     sync.Once // 只实例化一次交易池，golang 下的单例模式
	txPoolInst *TxPool
)

type TxPool struct {
	txQueue []string
	txs     sync.Map
	//txs    map[common.Hash]*common.Transaction
	flags  sync.Map
	height int
	lock   sync.RWMutex
}

func GetTxPoolInst() *TxPool {
	txOnce.Do(func() {
		txPoolInst = &TxPool{
			//txQueue: make([]*common.Hash, 0, 8192),
			txQueue: make([]string, 0, 8192),
			//txs:     sync.Map{},
			//txs: make(map[common.Hash]*common.Transaction),
		}
	})
	return txPoolInst
}

// Package 用于打包交易，这里返回的是 Transaction 的切片
// todo： 需要具体观察打包交易时的效率问题
func (pool *TxPool) Package() []common.Transaction {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	count := 0
	result := make([]common.Transaction, 0)
	log.Debugln("Start package txpool.")

	// 初期测试，直接返回空切片

	for idx := range pool.txQueue {
		txHash := pool.txQueue[idx]
		//log.Infoln(&txHash)
		value, hit := pool.txs.Load(txHash)
		pool.txs.Delete(txHash)
		//tx := pool.txs[txHash]
		if !hit {
			//log.Infoln("Map not hit.")
			continue
		}

		tx := value.(*common.Transaction)
		if !tx.Verify() {
			log.Errorln("Verify failed.")
			continue
		}
		// todo： 这里是传值还是传指针？
		result = append(result, *tx)
		count++
		metrics.TxPoolMetricsDec()

		if count >= maxTxPackageCount-1 {
			break
		}
	}

	pool.txQueue = pool.txQueue[count:]

	return result
}

func (pool *TxPool) Add(transaction *common.Transaction) {
	// todo: 还是需要和 Package 的锁相关联，保证 Package 能抢到锁
	// todo: 在当前版本下先直接加锁打包
	pool.lock.Lock()
	defer pool.lock.Unlock()
	defer metrics.TxPoolMetricsInc()

	txHash := hex.EncodeToString(transaction.Body.Hash[:])
	pool.txs.Store(txHash, transaction)
	//pool.txs[tx]
	pool.txQueue = append(pool.txQueue, txHash)
}

func (pool *TxPool) Contain(hash string) bool {
	_, hit := pool.txs.Load(hash)
	return hit
}

func (pool *TxPool) Get(hash string) *common.Transaction {
	value, hit := pool.txs.Load(hash)

	if !hit {
		return nil
	}

	return value.(*common.Transaction)
}
