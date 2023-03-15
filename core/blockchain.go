package core

import (
	"bytes"
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
	maxBlockProcessList = 12
)

type blockList []*common.Block

// !! 为了避免潜在的数据不一致的情况，任何情况下不要对一个 block 实例进行数据的修改
// 如果需要对区块进行校验或者哈希的修改，对数据进行深拷贝得到一份复制来进行处理

type BlockChain struct {
	db            *utils.LevelDB
	dbWriterQueue chan *common.Block

	blockHeightMap *lru.Cache
	blockCache     *lru.Cache
	txCache        *lru.Cache

	latestBlock  *common.Block
	latestHeight int
	latestLock   sync.RWMutex

	blockProcessList   []blockList
	blockProcessHeight int
	currentBlock       *common.Block
	nextBlockMap       map[common.Hash]blockList
	blockQueueSize     int

	buffer     *BlockBuffer
	appendLock sync.RWMutex
}

func NewBlockchain(db *utils.LevelDB) *BlockChain {
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

	chain := &BlockChain{
		db:                 db,
		dbWriterQueue:      make(chan *common.Block),
		blockHeightMap:     blockHeightMap,
		blockCache:         blockCache,
		txCache:            txCache,
		latestBlock:        nil,
		latestHeight:       -1,
		blockProcessList:   make([]blockList, 0, maxBlockProcessList),
		blockProcessHeight: -1,
		currentBlock:       nil,
		nextBlockMap:       make(map[common.Hash]blockList),
	}

	return chain
}

// PackageNewBlock 打包新的区块，传入交易序列
func (bc *BlockChain) PackageNewBlock(txs []common.Transaction) (*common.Block, error) {
	latestBlock, err := bc.selectBlock(false)

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

func (bc *BlockChain) NewGenesisBlock() {
	nullHash := common.Hash{}

	genesisBlock := common.Block{
		Header: common.BlockHeader{
			Timestamp:     uint64(time.Now().UnixMilli()),
			PrevBlockHash: nullHash,
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(nullHash),
			Height:        0,
		},
		Transactions: []common.Transaction{},
	}

	//println(hex.EncodeToString(nullHash[:]))
	bc.InsertBlock(&genesisBlock)
}

//todo：数据初始化

func (bc *BlockChain) GetLatestBlock() (*common.Block, error) {
	if bc.latestBlock != nil {
		return bc.latestBlock, nil
	}

	latestBlockHash, err := bc.db.Get([]byte("latest"))

	if err != nil {
		log.WithField("error", err).Debugln("Get latest index failed.")
		return nil, err
	}

	blockHash := common.Hash(latestBlockHash)

	block, err := bc.GetBlockByHash(&blockHash)

	if err != nil {
		log.WithField("error", err).Errorln("Get block failed.")
		return nil, err
	}

	return block, nil
}

func (bc *BlockChain) GetBlockByHash(hash *common.Hash) (*common.Block, error) {
	// 先查询缓存是否命中
	strHash := hex.EncodeToString(hash[:])
	value, hit := bc.blockCache.Get(strHash)

	if hit {
		// 命中缓存， 直接返回区块
		return value.(*common.Block), nil
	}

	blockKey := utils.BlockHash2DBKey(*hash)
	// todo: 需要一个命名规范
	byteBlockData, err := bc.db.Get(blockKey)

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
	if height > bc.latestHeight && bc.latestHeight != -1 {
		return nil, errors.New("Block height error.")
	}

	// 查询缓存里面是否有区块的信息
	value, ok := bc.blockHeightMap.Get(uint64(height))
	if ok {
		blockHash := common.Hash(value.([32]byte))
		block, err := bc.GetBlockByHash(&blockHash)

		if err != nil {
			log.WithField("error", err).Errorln("Get block by hash failed.")
			return nil, err
		}

		return block, nil
	}

	// 查询数据库
	heightDBKey := utils.BlockHeight2DBKey(height)
	blockHash, err := bc.db.Get(heightDBKey)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"key":   string(heightDBKey),
		}).Errorln("Get block hash with height failed.")
		return nil, err
	}

	hash := common.Hash(blockHash)
	block, err := bc.GetBlockByHash(&hash)

	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Errorln("Get block by hash failed.")
		return nil, err
	}

	return block, nil
}

func (bc *BlockChain) writeBlockCache(block *common.Block) {
	bc.blockHeightMap.Add(block.Header.Height, block.Header.BlockHash)
	blockHash := hex.EncodeToString(block.Header.BlockHash[:])
	bc.blockCache.Add(blockHash, block)
}

func (bc *BlockChain) writeTxCache(transaction *common.Transaction) {
	hash := transaction.Body.Hash
	strHash := hex.EncodeToString(hash[:])
	bc.txCache.Add(strHash, transaction)
}

// InsertBlock 函数用于将区块插入到数据库中
func (bc *BlockChain) InsertBlock(block *common.Block) {
	// todo: 这里作为 Public 函数只是为了测试
	var err error
	count := len(block.Transactions)

	latestBlock, err := bc.GetLatestBlock()

	if err != nil {
		log.WithField("error", err).Debugln("Get latest block failed.")
		return
	}

	if !isPrevBlock(latestBlock, block) {
		log.Errorln("Block error, prev block hash not match.")
		return
	}

	bc.latestLock.RLock()
	if int(block.Header.Height) <= bc.latestHeight {
		bc.latestLock.RUnlock()
		return
	}
	bc.latestBlock = block
	bc.latestHeight = int(block.Header.Height)
	bc.latestLock.RUnlock()
	bc.writeBlockCache(block)

	// todo: 把这里的魔数改成声明
	keys := make([][]byte, count+3)
	values := make([][]byte, count+3)
	//fmt.Printf("Data size %d bytes.", len(values[0]))

	log.WithFields(log.Fields{
		"hash":   hex.EncodeToString(block.Header.BlockHash[:]),
		"height": block.Header.Height,
		"count":  len(block.Transactions),
	}).Infoln("Insert block to database.")

	keys[0] = utils.BlockHash2DBKey(block.Header.BlockHash)
	values[0], err = utils.SerializeBlock(block)
	if err != nil {
		return
	}

	keys[1] = []byte("latest")
	values[1] = block.Header.BlockHash[:]
	keys[2] = utils.BlockHeight2DBKey(int(block.Header.Height))
	values[2] = block.Header.BlockHash[:]

	for idx := range block.Transactions {
		// todo： 或许需要校验一下交易是否合法
		tx := block.Transactions[idx]

		txWriter := karmem.NewWriter(1024)
		keys[idx+3] = append([]byte("tx#"), tx.Body.Hash[:]...)
		_, err := tx.WriteAsRoot(txWriter)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"hash":  hex.EncodeToString(tx.Body.Hash[:]),
			}).Errorln("Encode transaction failed.")
			return
		}

		values[idx+3] = txWriter.Bytes()
	}

	bc.db.BatchInsert(keys, values)
}

// databaseWriter 负责插入数据到数据库的协程
func (bc *BlockChain) databaseWriter() {
	for {
		select {
		case block := <-bc.dbWriterQueue:
			bc.InsertBlock(block)
			//
			//if err != nil {
			//	log.WithField("error", err).Errorln("Insert block to database failed")
			//}
		}
	}
}

func (bc *BlockChain) AppendBlockTask(block *common.Block) {
	bc.appendLock.RLock()
	defer bc.appendLock.RUnlock()

	if int(block.Header.Height) <= bc.latestHeight {
		return
	}

	blockHash := block.Header.BlockHash

	prevBlockHash := block.Header.PrevBlockHash
	blockHeight := int(block.Header.Height)
	latestHeight := bc.latestHeight

	// 这里是一个高度差
	idx := blockHeight - latestHeight - 1
	log.WithFields(log.Fields{
		"hash":   "0x" + hex.EncodeToString(blockHash[:]),
		"index":  idx,
		"height": block.Header.Height,
	}).Infoln("Append new block.")

	if idx >= maxBlockProcessList {
		bestBlock, err := bc.selectBlock(true)
		if err != nil {
			log.WithField("error", err).Debugln("Get best block failed.")
			return
		}

		bc.cleanupSelectQueue()
		idx--
		go bc.InsertBlock(bestBlock)
	}

	log.Infof("Blcok process list length: %d", len(bc.blockProcessList))
	if len(bc.blockProcessList) > idx {
		bc.blockProcessList[idx] = append(bc.blockProcessList[idx], block)
	} else {
		length := idx - len(bc.blockProcessList) + 1

		for i := 0; i < length; i++ {
			bc.blockProcessList = append(bc.blockProcessList, blockList{})
		}

		bc.blockProcessList[idx] = append(bc.blockProcessList[idx], block)
	}

	list := bc.nextBlockMap[prevBlockHash]

	if list == nil {
		// 大坑， make 存在长度的时候需要预先设置好初始长度
		list := make(blockList, 0, 16)
		list = append(list, block)
		bc.nextBlockMap[prevBlockHash] = list
	} else {
		bc.nextBlockMap[prevBlockHash] = append(bc.nextBlockMap[prevBlockHash], block)
	}
}

func (bc *BlockChain) GetTransactionByHash(hash common.Hash) (*common.Transaction, error) {
	strHash := hex.EncodeToString(hash[:])
	value, hit := bc.txCache.Get(strHash)

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
	bc.latestLock.RLock()
	defer bc.latestLock.RUnlock()
	return bc.latestHeight
}

func (bc *BlockChain) selectBlock(head bool) (*common.Block, error) {
	bc.appendLock.RLock()
	defer bc.appendLock.RUnlock()

	if len(bc.blockProcessList) == 0 {
		log.Infoln("Process list is empty, return latest block.")
		return bc.GetLatestBlock()
	}
	list := bc.blockProcessList[0]

	if len(list) == 0 {
		log.Infoln("Process list[0] is empty, return latest block.")
		return bc.GetLatestBlock()
	}

	current := list[0]

	for idx := range list {
		if len(list[idx].Transactions) > len(current.Transactions) {
			current = list[idx]
			continue
		} else if len(list[idx].Transactions) < len(current.Transactions) {
			continue
		}

		if list[idx].Header.Timestamp < current.Header.Timestamp {
			current = list[idx]
			continue
		}
	}

	if head {
		return current, nil
	}

	searchResult := bc.searchBestBlock(current.Header.BlockHash)

	if searchResult != nil {
		return searchResult, nil
	} else {
		return current, nil
	}
}

func (bc *BlockChain) searchBestBlock(hash common.Hash) *common.Block {
	list := bc.nextBlockMap[hash]

	if list == nil || len(list) <= 0 {
		return nil
	}
	current := list[0]

	for idx := range list {
		if len(list[idx].Transactions) > len(current.Transactions) {
			current = list[idx]
			continue
		} else if len(list[idx].Transactions) < len(current.Transactions) {
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

func (bc *BlockChain) cleanupSelectQueue() {
	bc.appendLock.RLock()
	defer bc.appendLock.RUnlock()

	var list blockList

	if len(bc.blockProcessList) == 0 {
		return
	}

	list, bc.blockProcessList = bc.blockProcessList[0], bc.blockProcessList[1:]

	for idx := range list {
		delete(bc.nextBlockMap, list[idx].Header.BlockHash)
	}
}

func isPrevBlock(prev *common.Block, block *common.Block) bool {
	prevBlockHash := prev.Header.BlockHash[:]
	blockPrevHash := block.Header.PrevBlockHash[:]
	return bytes.Compare(prevBlockHash, blockPrevHash) == 0
}
