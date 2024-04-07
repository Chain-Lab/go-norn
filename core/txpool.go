// Package core
// @Description: 交易池结构
package core

import (
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/metrics"
	log "github.com/sirupsen/logrus"
	"sync"
)

const (
	maxTxPackageCount = 10000 // 交易池打包的最多交易数量
	maxTxPoolSize     = 20480 // 交易池存放的的最多交易数量
)

var (
	txOnce     sync.Once       // 只实例化一次交易池，golang 下的单例模式
	txPoolInst *TxPool   = nil // 交易池实例，单例模式
)

type TxPool struct {
	chain        *BlockChain              // 区块链实例，用于查询交易是否存在
	txQueue      chan string              // 交易打包等待队列，存放哈希值
	waitingQueue chan *common.Transaction //
	txs          sync.Map                 // 交易 map， hash -> transaction
	count        int                      // 当前的交易池交易数量

	flags  sync.Map
	height int
}

// NewTxPool
//
//	@Description: 获取一个交易池单例，如果有则直接返回
//	@param chain - 区块链实例
//	@return *TxPool - 交易池实例
func NewTxPool(chain *BlockChain) *TxPool {
	txOnce.Do(func() {
		txPoolInst = &TxPool{
			chain:   chain,
			txQueue: make(chan string, 10240),

			count: 0,
		}
	})
	return txPoolInst
}

// GetTxPoolInst
//
//	@Description: 获取交易池实例
//	@return *TxPool
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

// Package
//
//	@Description: 用于打包交易，这里返回的是 Transaction 的切片
//	@receiver pool - 交易池实例
//	@return []common.Transaction - 交易数组
func (pool *TxPool) Package() []common.Transaction {
	log.Debugln("Start package transaction...")

	// 初始化返回列表
	count := 0
	result := make([]common.Transaction, 0, maxTxPackageCount)
	log.Debugln("Start package tx pool.")

	// 遍历存放交易的 txs 映射，并将哈希添加到打包队列 txQueue 中
	pool.txs.Range(pool.rangeFunc)

	// 遍历最多 maxTxPackageCount 次
	for idx := 0; idx < maxTxPackageCount; idx++ {
		if len(pool.txQueue) == 0 || count > maxTxPackageCount {
			log.Debugln("transaction queue is empty or package finish")
			break
		}

		// 从 txQueue 中读取一条哈希
		txHash := <-pool.txQueue
		commonHash, err := hex.DecodeString(txHash)
		if err != nil {
			log.Errorln("Decode transaction hash failed.")
			continue
		}

		//  如果交易命中，说明当前交易有效，否则继续遍历
		value, hit := pool.txs.Load(txHash)
		if !hit {
			continue
		}

		// 查询链上是否存在交易，存在则跳过
		tx, err := pool.chain.GetTransactionByHash(common.Hash(commonHash))
		if tx != nil {
			pool.txs.Delete(txHash)
			metrics.TxPoolMetricsDec()
			pool.count--
			log.Debugln("Transaction already in database.")
			continue
		}

		// 从 txs 中移除对应的交易
		pool.txs.Delete(txHash)
		metrics.TxPoolMetricsDec()
		pool.count--
		tx = value.(*common.Transaction)

		result = append(result, *tx)
		count++
	}
	return result
}

// Add
//
//	@Description: 向交易池中添加交易
//	@receiver pool - 交易池实例
//	@param transaction - 一笔交易的实例
func (pool *TxPool) Add(transaction *common.Transaction) {
	if pool.count > maxTxPoolSize {
		return
	}

	txHash := hex.EncodeToString(transaction.Body.Hash[:])
	pool.txs.Store(txHash, transaction)
	metrics.TxPoolMetricsInc()
	pool.count++
}

// Contain
//
//	@Description:  查询交易池中是否存在某个哈希值对应的交易
//	@receiver pool - 交易池实例
//	@param hash - 查询的交易哈希值
//	@return bool - 存在则返回 true
func (pool *TxPool) Contain(hash string) bool {
	_, hit := pool.txs.Load(hash)
	return hit
}

// RemoveTx
//
//	@Description: 从交易池中移除一笔交易
//	@receiver pool - 交易池实例
//	@param hash - 需要移除的交易
func (pool *TxPool) RemoveTx(hash common.Hash) {
	txHash := hex.EncodeToString(hash[:])

	_, hit := pool.txs.Load(txHash)
	if !hit {
		return
	}

	pool.txs.Delete(txHash)
	pool.count--
	metrics.TxPoolMetricsDec()
}

// Get
//
//	@Description: 获取交易池中的交易
//	@receiver pool - 交易池实例
//	@param hash - 所需要的交易的哈希值
//	@return *common.Transaction - 交易实例
func (pool *TxPool) Get(hash string) *common.Transaction {
	value, hit := pool.txs.Load(hash)

	if !hit {
		return nil
	}

	return value.(*common.Transaction)
}
