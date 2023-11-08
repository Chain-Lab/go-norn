package core

import (
	"encoding/hex"
	"github.com/chain-lab/go-chronos/common"
	"github.com/chain-lab/go-chronos/metrics"
	log "github.com/sirupsen/logrus"
	"sync"
)

const (
	maxTxPackageCount = 10000 // 交易池打包的最多交易数量
)

var (
	txOnce     sync.Once // 只实例化一次交易池，golang 下的单例模式
	txPoolInst *TxPool   = nil
)

type TxPool struct {
	chain        *BlockChain
	txQueue      chan string
	waitingQueue chan *common.Transaction
	txs          sync.Map

	flags  sync.Map
	height int
}

func NewTxPool(chain *BlockChain) *TxPool {
	txOnce.Do(func() {
		txPoolInst = &TxPool{
			chain:   chain,
			txQueue: make(chan string, 10240),
		}
	})
	return txPoolInst
}

func GetTxPoolInst() *TxPool {
	return txPoolInst
}

func (pool *TxPool) rangeFunc(key, value interface{}) bool {
	if len(pool.txQueue) >= maxTxPackageCount {
		return false
	}
	txHash, ok := key.(string)
	if ok {
		pool.txQueue <- txHash
	}

	return true
}

// Package 用于打包交易，这里返回的是 Transaction 的切片
// todo： 需要具体观察打包交易时的效率问题
func (pool *TxPool) Package() []common.Transaction {
	log.Debugln("Start package transaction...")

	count := 0
	result := make([]common.Transaction, 0, maxTxPackageCount)
	log.Debugln("Start package tx pool.")

	pool.txs.Range(pool.rangeFunc)

	for idx := 0; idx < maxTxPackageCount; idx++ {
		if len(pool.txQueue) == 0 || count > maxTxPackageCount {
			log.Debugln("transaction queue is empty or package finish")
			break
		}

		txHash := <-pool.txQueue
		commonHash, err := hex.DecodeString(txHash)
		if err != nil {
			log.Errorln("Decode transaction hash failed.")
			continue
		}

		value, hit := pool.txs.Load(txHash)
		if !hit {
			continue
		}

		tx, err := pool.chain.GetTransactionByHash(common.Hash(commonHash))
		if tx != nil {
			pool.txs.Delete(txHash)
			metrics.TxPoolMetricsDec()
			log.Debugln("Transaction already in database.")
			continue
		}

		pool.txs.Delete(txHash)
		metrics.TxPoolMetricsDec()
		tx = value.(*common.Transaction)

		result = append(result, *tx)
		count++
	}
	return result
}

func (pool *TxPool) Add(transaction *common.Transaction) {
	txHash := hex.EncodeToString(transaction.Body.Hash[:])
	pool.txs.Store(txHash, transaction)
	metrics.TxPoolMetricsInc()
}

func (pool *TxPool) Contain(hash string) bool {
	_, hit := pool.txs.Load(hash)
	return hit
}

func (pool *TxPool) RemoveTx(hash common.Hash) {
	txHash := hex.EncodeToString(hash[:])

	_, hit := pool.txs.Load(txHash)
	if !hit {
		return
	}

	pool.txs.Delete(txHash)
	metrics.TxPoolMetricsDec()
}

func (pool *TxPool) CleanPool() {

}

func (pool *TxPool) Get(hash string) *common.Transaction {
	value, hit := pool.txs.Load(hash)

	if !hit {
		return nil
	}

	return value.(*common.Transaction)
}
