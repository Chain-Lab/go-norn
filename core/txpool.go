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

	packing  bool
	packCond *sync.Cond
	lock     *sync.Mutex

	flags  sync.Map
	height int
}

func NewTxPool(chain *BlockChain) *TxPool {
	txOnce.Do(func() {
		lock := &sync.Mutex{}

		txPoolInst = &TxPool{
			chain:    chain,
			txQueue:  make(chan string, 8192),
			packing:  false,
			lock:     lock,
			packCond: sync.NewCond(lock),
			//txs:     sync.Map{},
			//txs: make(map[common.Hash]*common.Transaction),
		}
	})
	return txPoolInst
}

func GetTxPoolInst() *TxPool {
	return txPoolInst
}

func (pool *TxPool) setPackStart() {
	log.Infoln("Set packing to true.")
	pool.packing = true
}

func (pool *TxPool) setPackStop() {
	pool.packing = false
}

// Package 用于打包交易，这里返回的是 Transaction 的切片
// todo： 需要具体观察打包交易时的效率问题
func (pool *TxPool) Package() []common.Transaction {
	log.Debugln("Start package transaction...")
	pool.setPackStart()
	pool.lock.Lock()
	log.Infoln("pool locked")

	defer log.Infoln("pool unlocked")
	defer pool.lock.Unlock()
	defer pool.packCond.Broadcast()
	defer pool.setPackStop()

	count := 0
	result := make([]common.Transaction, 0, maxTxPackageCount)
	log.Debugln("Start package tx pool.")
	for idx := 0; idx < maxTxPackageCount; idx++ {
		//log.Infof("transaction queue length: %d", len(pool.txQueue))
		if len(pool.txQueue) == 0 || count > maxTxPackageCount {
			log.Debugln("transaction queue is empty or package finish")
			break
		}

		//log.Infof("Package block index %d", idx)

		txHash := <-pool.txQueue
		metrics.TxPoolMetricsDec()
		commonHash, err := hex.DecodeString(txHash)
		if err != nil {
			log.Errorln("Decode transaction hash failed.")
			continue
		}

		tx, err := pool.chain.GetTransactionByHash(common.Hash(commonHash))
		if tx != nil {
			pool.txs.Delete(txHash)
			log.Debugln("Transaction already in database.")
			continue
		}

		value, hit := pool.txs.Load(txHash)
		pool.txs.Delete(txHash)
		if !hit {
			continue
		}

		tx = value.(*common.Transaction)
		if !tx.Verify() {
			log.Errorln("Verify failed.")
			continue
		}
		// todo： 这里是传值还是传指针？
		result = append(result, *tx)
		count++
	}
	return result
}

func (pool *TxPool) Add(transaction *common.Transaction) {
	pool.lock.Lock()
	defer pool.lock.Unlock()

	txHash := hex.EncodeToString(transaction.Body.Hash[:])

	for pool.packing {
		pool.packCond.Wait()
	}

	select {
	case pool.txQueue <- txHash:
		pool.txs.Store(txHash, transaction)
		metrics.TxPoolMetricsInc()
	default:

	}
}

func (pool *TxPool) Contain(hash string) bool {
	_, hit := pool.txs.Load(hash)
	return hit
}

func (pool *TxPool) RemoveTx(hash common.Hash) {
	pool.lock.Lock()
	defer pool.lock.Unlock()
	txHash := hex.EncodeToString(hash[:])

	for pool.packing {
		pool.packCond.Wait()
	}

	pool.txs.Delete(txHash)
}

func (pool *TxPool) Get(hash string) *common.Transaction {
	value, hit := pool.txs.Load(hash)

	if !hit {
		return nil
	}

	return value.(*common.Transaction)
}
