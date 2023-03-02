package core

import (
	"crypto/sha256"
	"encoding/hex"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"go-chronos/common"
	"go-chronos/utils"
	karmem "karmem.org/golang"
	"sync"
	"time"
)

const (
	maxBlockCache       = 1024
	maxTransactionCache = 32768
	maxBlockProcessList = 64
)

type blockList []*common.Block

type BlockChain struct {
	db            *utils.LevelDB
	dbWriterQueue chan *common.Block

	blockHeightMap *lru.Cache
	blockCache     *lru.Cache
	txCache        *lru.Cache

	latestBlock  *common.Block
	latestHeight int
	latestLock   sync.RWMutex

	blockProcessList   blockList
	blockProcessHeight int
	currentBlock       *common.Block
	nextBlockMap       *lru.Cache
	blockQueueSize     int
	appendLock         sync.RWMutex
}

func NewBlockchain() *BlockChain {
	blockCache, err := lru.New(maxBlockCache)
	if err != nil {
		log.WithField("error", err).Debugln("Create block cache failed")
		return nil
	}

	txCache, err := lru.New(maxTransactionCache)
	if err != nil {
		log.WithField("error", err).Debugln("Create transaction cache failed.")
		return nil
	}

	blockHeightMap, err := lru.New(maxBlockCache)
	if err != nil {
		log.WithField("error", err).Debugln("Create block map cache failed.")
		return nil
	}

	nextBlockMap, err := lru.New(maxBlockProcessList)
	if err != nil {
		log.WithField("error", err).Debugln("Create block process map failed.")
		return nil
	}

	chain := &BlockChain{
		dbWriterQueue:      make(chan *common.Block),
		blockHeightMap:     blockHeightMap,
		blockCache:         blockCache,
		txCache:            txCache,
		latestBlock:        nil,
		latestHeight:       -1,
		blockProcessList:   make(blockList, maxBlockProcessList),
		blockProcessHeight: -1,
		currentBlock:       nil,
		nextBlockMap:       nextBlockMap,
	}

	return chain
}

// PackageNewBlock 打包新的区块，传入交易序列
func (bc *BlockChain) PackageNewBlock(txs []common.Transaction) (*common.Block, error) {
	latestBlock, err := bc.selectBlock()

	if err != nil {
		log.WithField("error", err).Debugln("Get latest block failed.")
		return nil, err
	}

	merkleRoot := BuildMerkleTree(txs)
	block := common.Block{
		Header: common.BlockHeader{
			Timestamp:     uint64(time.Now().UnixMilli()),
			PrevBlockHash: latestBlock.Header.BlockHash,
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(merkleRoot),
			Height:        latestBlock.Header.Height + 1,
		},
		Transactions: txs,
	}

	byteBlockHeaderData, err := utils.SerializeBlockHeader(&block.Header)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize block header failed.")
		return nil, err
	}

	hash := sha256.New()
	hash.Write(byteBlockHeaderData)
	blockHash := common.Hash(hash.Sum(nil))
	block.Header.BlockHash = blockHash

	return &block, nil
}

//todo：数据初始化

func (bc *BlockChain) GetLatestBlock() (*common.Block, error) {
	if bc.latestBlock != nil {
		return bc.latestBlock, nil
	}

	db := utils.GetLevelDBInst()
	latestBlockHash, err := db.Get([]byte("latest"))
	if err != nil {
		log.WithField("error", err).Errorln("Get latest block hash failed.")
		return nil, err
	}

	byteBlockData, err := db.Get(utils.BlockHash2DBKey(common.Hash(latestBlockHash)))

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"hash":  hex.EncodeToString(latestBlockHash),
		}).Errorln("Get latest block data failed.")
		return nil, err
	}

	block, err := utils.DeserializeBlock(byteBlockData)

	if err != nil {
		log.WithField("error", err).Errorln("Deserialize block failed.")
		return nil, err
	}

	return block, nil
}

func (bc *BlockChain) GetBlockByHash(hash *common.Hash) (*common.Block, error) {
	// 先查询缓存是否命中
	value, hit := bc.blockCache.Get(hash)

	if hit {
		// 命中缓存， 直接返回区块
		return value.(*common.Block), nil
	}

	blockKey := utils.BlockHash2DBKey(*hash)
	db := utils.GetLevelDBInst()
	// todo: 需要一个命名规范
	byteBlockData, err := db.Get(blockKey)

	if err != nil {
		log.WithField("error", err).Debugln("Get block data in database failed.")
		return nil, err
	}
	block, _ := utils.DeserializeBlock(byteBlockData)
	bc.writeBlockCache(block)

	return block, nil
}

func (bc *BlockChain) GetBlockByHeight(height int) (*common.Block, error) {
	// 先判断一下高度
	if height > bc.latestHeight {
		return nil, errors.New("Block height error.")
	}

	// 查询缓存里面是否有区块的信息
	value, ok := bc.blockHeightMap.Get(height)
	if ok {
		blockHash := value.(*common.Hash)
		block, err := bc.GetBlockByHash(blockHash)

		if err != nil {
			log.WithField("error", err).Errorln("Get block by hash failed.")
			return nil, err
		}

		return block, nil
	}

	// 查询数据库
	db := utils.GetLevelDBInst()
	heightDBKey := utils.BlockHeight2DBKey(height)
	blockHash, err := db.Get(heightDBKey)

	if err != nil {
		log.WithField("error", err).Errorln("Get block hash with height failed.")
		return nil, err
	}

	hash := common.Hash(blockHash)
	block, err := bc.GetBlockByHash(&hash)

	if err != nil {
		log.WithField("error", err).Errorln("Get block by hash failed.")
		return nil, err
	}

	return block, nil
}

func (bc *BlockChain) writeBlockCache(block *common.Block) {
	bc.blockHeightMap.Add(block.Header.Height, block.Header.BlockHash)
	bc.blockCache.Add(block.Header.BlockHash, block)
}

func (bc *BlockChain) writeTxCache(transaction *common.Transaction) {
	hash := transaction.Body.Hash
	bc.txCache.Add(hash, transaction)
}

// InsertBlock 函数用于将区块插入到数据库中
func (bc *BlockChain) InsertBlock(block *common.Block) error {
	// todo: 这里作为 Public 函数只是为了测试
	var err error
	count := len(block.Transactions)
	keys := make([][]byte, count+1)
	values := make([][]byte, count+1)

	values[0], err = utils.SerializeBlock(block)
	if err != nil {
		return err
	}
	//fmt.Printf("Data size %d bytes.", len(values[0]))

	bc.latestLock.RLock()
	bc.latestBlock = block
	bc.latestHeight = int(block.Header.Height)
	bc.latestLock.RUnlock()
	bc.writeBlockCache(block)

	db := utils.GetLevelDBInst()

	for idx := range block.Transactions {
		// todo： 或许需要校验一下交易是否合法
		tx := block.Transactions[idx]

		txWriter := karmem.NewWriter(1024)
		keys[idx+1] = append([]byte("tx#"), tx.Body.Hash[:]...)
		_, err := tx.WriteAsRoot(txWriter)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"hash":  hex.EncodeToString(tx.Body.Hash[:]),
			}).Errorln("Encode transaction failed.")
			return err
		}

		values[idx+1] = txWriter.Bytes()
	}

	return db.BatchInsert(keys, values)
}

// databaseWriter 负责插入数据到数据库的协程
func (bc *BlockChain) databaseWriter() {
	for {
		select {
		case block := <-bc.dbWriterQueue:
			err := bc.InsertBlock(block)

			if err != nil {
				log.WithField("error", err).Errorln("Insert block to database failed")
			}
		}
	}
}

func (bc *BlockChain) AppendBlockTask(block *common.Block) {
	bc.appendLock.RLock()
	defer bc.appendLock.RUnlock()

	if int(block.Header.Height) <= bc.latestHeight {
		return
	}

	//blockHash := block.Header.BlockHash
}

func (bc *BlockChain) GetTransactionByHash(hash common.Hash) (*common.Transaction, error) {
	value, hit := bc.txCache.Get(hash)

	if hit {
		return value.(*common.Transaction), nil
	}

	txDBKey := utils.TxHash2DBKey(hash)
	byteTransactionData, err := bc.db.Get(txDBKey)

	if err != nil {
		log.WithField("error", err).Debugln("Get tx data from database failed.")
		return nil, err
	}

	transaction, err := utils.DeserializeTransaction(byteTransactionData)

	if err != nil {
		log.WithField("error", err).Debugln("Deserialize transaction failed.")
		return nil, err
	}

	bc.writeTxCache(transaction)
	return transaction, nil
}

func (bc *BlockChain) Height() int {
	return bc.latestHeight
}

func (bc *BlockChain) selectBlock() (*common.Block, error) {
	bc.appendLock.RLock()
	defer bc.appendLock.RUnlock()

	list := bc.blockProcessList
	current := list[0]

	for idx := range list {
		if len(list[idx].Transactions) > len(list[idx].Transactions) {
			current = list[idx]
			continue
		} else if len(list[idx].Transactions) < len(list[idx].Transactions) {
			continue
		}

		if list[idx].Header.Timestamp < current.Header.Timestamp {
			current = list[idx]
			continue
		}
	}

	return bc.searchBestBlock(current.Header.BlockHash), nil
}

func (bc *BlockChain) searchBestBlock(hash common.Hash) *common.Block {
	value, hit := bc.nextBlockMap.Get(hash)

	if !hit {
		return nil
	}

	list := value.(blockList)
	current := list[0]

	for idx := range list {
		if len(list[idx].Transactions) > len(list[idx].Transactions) {
			current = list[idx]
			continue
		} else if len(list[idx].Transactions) < len(list[idx].Transactions) {
			continue
		}

		if list[idx].Header.Timestamp < current.Header.Timestamp {
			current = list[idx]
			continue
		}
	}

	nxt := bc.searchBestBlock(current.Header.BlockHash)

	if nxt == nil {
		return current
	} else {
		return nxt
	}
}
