package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/gookit/config/v2"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"go-chronos/common"
	"go-chronos/crypto"
	"go-chronos/metrics"
	"time"

	//handler "go-chronos/node"
	"go-chronos/utils"
	karmem "karmem.org/golang"
	"math/big"
	"sync"
)

const (
	maxBlockCache         = 1024
	maxTransactionCache   = 32768
	maxBlockProcessList   = 12
	maxBlockChannel       = 128
	maxDbChannel          = 256
	transactionStartIndex = 3
)

type blockList []*common.Block

// !! 为了避免潜在的数据不一致的情况，任何情况下不要对一个 block 实例进行数据的修改
// 如果需要对区块进行校验或者哈希的修改，对数据进行深拷贝得到一份复制来进行处理

type BlockChain struct {
	// 数据库相关的成员变量
	db            *utils.LevelDB
	dbWriterQueue chan *common.Block

	// 当前的链所维护的区块高度对应区块 blockHeightMap、区块 cache 和交易 cache
	blockHeightMap *lru.Cache
	blockCache     *lru.Cache
	txCache        *lru.Cache

	// 当前的最新区块 latestBlock、最新高度 latestHeight
	latestBlock  *common.Block
	latestHeight int64
	// 最新状态的同步锁
	latestLock sync.RWMutex

	// 区块缓冲区的 channel，以及所维护的 buffer
	bufferChan chan *common.Block
	buffer     *BlockBuffer
	appendLock sync.RWMutex

	// genesisParams 当前所维护的链的创世区块参数
	genesisParams *common.GenesisParams
	genesisTime   int64
}

// NewBlockchain 创建一个 BlockChain 实，需要传入一个 LevelDB 实例 db
func NewBlockchain(db *utils.LevelDB) *BlockChain {
	// 实例化一系列的 cache 并处理可能出现的错误
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

	// 使用数据库实例 db 实例化一个 Blockchain 对象
	chain := &BlockChain{
		db:            db,
		dbWriterQueue: make(chan *common.Block, maxDbChannel),

		blockHeightMap: blockHeightMap,
		blockCache:     blockCache,
		txCache:        txCache,

		latestBlock:  nil,
		latestHeight: -1,

		bufferChan: make(chan *common.Block, maxBlockChannel),
	}

	// 初始化一下最新区块
	latest, _ := chain.GetLatestBlock()

	if latest != nil {
		log.Traceln("Block database is not null, create buffer.")
		chain.createBlockBuffer(latest)

		// 加载创世区块参数
		genesis, _ := chain.GetBlockByHeight(0)
		chain.genesisInitialization(genesis)
	}

	// 区块处理协程启动
	metrics.RoutineCreateHistogramObserve(7)
	go chain.BlockProcessRoutine()
	return chain
}

// BlockProcessRoutine 接收处理 channel 中的区块
func (bc *BlockChain) BlockProcessRoutine() {
	for {
		select {
		case block := <-bc.bufferChan:
			bc.InsertBlock(block)
		}
	}
}

// PackageNewBlock 打包新的区块，传入交易序列
func (bc *BlockChain) PackageNewBlock(txs []common.Transaction, timestamp int64, params *common.GeneralParams, packageInterval int64) (*common.Block, error) {
	packageStart := time.Now()

	// 对传入的区块参数进行序列化
	log.Traceln("Start package new block.")
	paramsBytes, err := utils.SerializeGeneralParams(params)

	if err != nil {
		log.WithField("error", err).Errorln("Serialize params failed while package block.")
		return nil, err
	}

	// 从区块缓冲视图中找到当前视图下最优的区块，具体的选取方法需要查阅文档 （wiki/区块缓冲视图.md）
	nowHeight := (timestamp-bc.genesisTime)/1000/packageInterval + 1
	bestBlock := bc.buffer.GetPriorityLeaf(nowHeight)
	publicKey, err := hex.DecodeString(config.String("consensus.pub"))

	if err != nil {
		log.Errorln("Get public key from config failed.")
		return nil, err
	}

	// 对交易列表构建 Merkle 哈希树
	merkleRoot := BuildMerkleTree(txs)
	// 区块创建
	block := common.Block{
		Header: common.BlockHeader{
			//Timestamp:     handler.GetLogicClock(),
			Timestamp:     timestamp,
			PrevBlockHash: bestBlock.Header.BlockHash,
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(merkleRoot),
			Height:        bestBlock.Header.Height + 1,
			PublicKey:     [33]byte(publicKey),
			Params:        paramsBytes,
		},
		Transactions: txs,
	}

	// 区块头序列化
	byteBlockHeaderData, err := utils.SerializeBlockHeader(&block.Header)

	if err != nil {
		log.WithField("error", err).Debugln("Serialize block header failed.")
		return nil, err
	}

	hash := sha256.New()
	hash.Write(byteBlockHeaderData)
	blockHash := common.Hash(hash.Sum(nil))
	block.Header.BlockHash = blockHash

	metrics.PackageBlockMetricsSet(float64(time.Since(packageStart).Milliseconds()))
	return &block, nil
}

// NewGenesisBlock 创建创世区块
func (bc *BlockChain) NewGenesisBlock() {
	// 获取本地高度为 0 的区块，检查创世区块是否存在
	block, _ := bc.GetBlockByHeight(0)

	if block != nil {
		log.Errorln("Genesis block already exists.")
		return
	}

	// 生成创世区块内的 VDF 参数
	genesisParams, err := crypto.GenerateGenesisParams()

	if err != nil {
		log.WithField("error", err).Errorln("Generate genesis params failed.")
		return
	}

	// 对参数进行序列化为字节数组
	genesisParamsBytes, err := utils.SerializeGenesisParams(genesisParams)

	if err != nil {
		log.WithField("error", err).Errorln("Genesis params serialize error.")
		return
	}

	nullHash := common.Hash{}
	// todo: 创世区块应该是前一个区块的哈希为 0x0，这里需要修改
	genesisBlock := common.Block{
		Header: common.BlockHeader{
			Timestamp: time.Now().UnixMilli(),
			//Timestamp:     handler.GetLogicClock(),
			PrevBlockHash: nullHash,
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(nullHash),
			Height:        0,
			Params:        genesisParamsBytes,
		},
		Transactions: []common.Transaction{},
	}

	bc.InsertBlock(&genesisBlock)
}

// GetLatestBlock 获取当前的最新区块
func (bc *BlockChain) GetLatestBlock() (*common.Block, error) {
	// 首先尝试获取变量下存储的区块
	if bc.latestBlock != nil {
		return bc.latestBlock, nil
	}

	// 然后从数据库的索引下查询
	latestBlockHash, err := bc.db.Get([]byte("latest"))

	if err != nil {
		log.WithField("error", err).Debugln("Get latest index failed.")
		return nil, err
	}

	// 如果索引存在，则从数据库中拉取
	blockHash := common.Hash(latestBlockHash)

	block, err := bc.GetBlockByHash(&blockHash)

	if err != nil {
		log.WithField("error", err).Errorln("Get block failed.")
		return nil, err
	}

	bc.latestLock.Lock()
	bc.latestBlock = block
	bc.latestHeight = block.Header.Height
	bc.latestLock.Unlock()

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

// GetBlockByHeight 根据高度拉取区块，需要考虑一下同步时是否换用其他的函数？
func (bc *BlockChain) GetBlockByHeight(height int64) (*common.Block, error) {
	// 先判断一下高度
	if height > bc.latestHeight && bc.latestHeight != -1 {
		return nil, errors.New("Block height error.")
	}

	// 查询缓存里面是否有区块的信息
	value, _ := bc.blockHeightMap.Get(height)
	if value != nil {
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

// writeBlockCache 向 哈希 -> 区块的 cache 下添加区块信息
func (bc *BlockChain) writeBlockCache(block *common.Block) {
	bc.blockHeightMap.Add(block.Header.Height, block.Header.BlockHash)
	blockHash := block.BlockHash()
	bc.blockCache.Add(blockHash, block)
}

// writeTxCache 向 哈希 -> 交易的 cache 中添加交易
func (bc *BlockChain) writeTxCache(transaction *common.Transaction) {
	hash := transaction.Body.Hash
	strHash := hex.EncodeToString(hash[:])
	bc.txCache.Add(strHash, transaction)
}

// InsertBlock 函数用于将区块插入到数据库中
func (bc *BlockChain) InsertBlock(block *common.Block) {
	// todo: 这里作为 Public 函数只是为了测试
	count := len(block.Transactions)

	blockHash := common.Hash(block.Header.BlockHash)

	// 获取对应哈希的区块，如果区块存在，说明链上已经存在该区块
	_, err := bc.GetBlockByHash(&blockHash)
	if err == nil {
		log.WithField("hash", block.BlockHash()[:8]).Warning("Block exists.")
		return
	}

	latestBlock, err := bc.GetLatestBlock()
	if !block.IsGenesisBlock() {
		// 插入区块非创世区块，需要检查拉取最新区块 是否成功 以及 前一个区块的哈希值是否符合
		if err != nil {
			log.WithField("error", err).Debugln("Get latest block failed.")
			return
		}

		if !isPrevBlock(latestBlock, block) {
			log.WithFields(log.Fields{
				"height": block.Header.Height,
				"prev":   block.PrevBlockHash()[:8],
				"latest": latestBlock.BlockHash()[:8],
			}).Errorln("Block error, prev block hash not match.")
			return
		}
	} else {
		// 插入区块是创世区块，说明 buffer 没有初始化，需要进行初始化
		bc.createBlockBuffer(block)
		bc.genesisInitialization(block)
	}

	bc.latestLock.Lock()
	if block.Header.Height <= bc.latestHeight {
		bc.latestLock.Unlock()
		return
	}
	bc.latestBlock = block
	bc.latestHeight = block.Header.Height
	bc.latestLock.Unlock()
	bc.writeBlockCache(block)

	// 交易列表添加到数据库前需要预留三个位置，修改 latest的哈希值、区块、区块高度对应的哈希
	keys := make([][]byte, count+transactionStartIndex)
	values := make([][]byte, count+transactionStartIndex)
	//fmt.Printf("Data size %d bytes.", len(values[0]))

	log.WithFields(log.Fields{
		"hash":    hex.EncodeToString(block.Header.BlockHash[:])[:8],
		"height":  block.Header.Height,
		"count":   len(block.Transactions),
		"address": hex.EncodeToString(block.Header.PublicKey[:])[:8],
	}).Infoln("Insert block to database.")

	keys[0] = utils.BlockHash2DBKey(block.Header.BlockHash)
	values[0], err = utils.SerializeBlock(block)
	if err != nil {
		return
	}

	keys[1] = []byte("latest")
	values[1] = block.Header.BlockHash[:]
	keys[2] = utils.BlockHeight2DBKey(block.Header.Height)
	values[2] = block.Header.BlockHash[:]

	for idx := range block.Transactions {
		// todo： 或许需要校验一下交易是否合法
		tx := block.Transactions[idx]

		txWriter := karmem.NewWriter(1024)
		keys[idx+transactionStartIndex] = append([]byte("tx#"), tx.Body.Hash[:]...)
		_, err := tx.WriteAsRoot(txWriter)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"hash":  hex.EncodeToString(tx.Body.Hash[:])[:8],
			}).Errorln("Encode transaction failed.")
			return
		}

		values[idx+transactionStartIndex] = txWriter.Bytes()
	}

	// 批量添加/修改数据到数据库
	bc.db.BatchInsert(keys, values)
	seed := new(big.Int)
	proof := new(big.Int)

	// todo: 异常处理
	// 根据区块的信息来反序列化参数
	if block.Header.Height == 0 {
		params, _ := utils.DeserializeGenesisParams(block.Header.Params)
		// todo: 将编码转换的过程放入到VRF代码中
		seed.SetBytes(params.Seed[:])
		proof.SetInt64(0)
	} else {
		params, _ := utils.DeserializeGeneralParams(block.Header.Params)
		// todo: 将编码转换的过程放入到VRF代码中
		seed.SetBytes(params.Result)
		proof.SetBytes(params.Proof)
	}

	// 向 VDF 的计算添加区块下的信息
	calculator := crypto.GetCalculatorInstance()
	calculator.AppendNewSeed(seed, proof)
}

// AppendBlockTask 向区块缓冲视图中添加区块处理任务
func (bc *BlockChain) AppendBlockTask(block *common.Block) {
	if block.IsGenesisBlock() {
		bc.InsertBlock(block)
		return
	}

	log.Debugln("Append block to buffer.")
	bc.buffer.AppendBlock(block)
}

// GetTransactionByHash 通过交易的哈希值来获取交易
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

func (bc *BlockChain) Height() int64 {
	bc.latestLock.RLock()
	defer bc.latestLock.RUnlock()
	return bc.latestHeight
}

func (bc *BlockChain) BufferedHeight() int64 {
	if bc.buffer == nil {
		return 0
	}
	return bc.buffer.bufferedHeight
}

func isPrevBlock(prev *common.Block, block *common.Block) bool {
	prevBlockHash := prev.Header.BlockHash[:]
	blockPrevHash := block.Header.PrevBlockHash[:]
	return bytes.Compare(prevBlockHash, blockPrevHash) == 0
}

func (bc *BlockChain) createBlockBuffer(latest *common.Block) {
	// todo: 需要处理报错
	log.Traceln("Create new block buffer.")
	bc.buffer, _ = NewBlockBuffer(latest, bc.bufferChan)
}

func (bc *BlockChain) genesisInitialization(block *common.Block) {
	if block != nil {
		// todo: 错误处理
		// 对区块中的参数进行反序列化
		genesisParams, _ := utils.DeserializeGenesisParams(block.Header.Params)
		bc.genesisParams = genesisParams
		bc.genesisTime = block.Header.Timestamp

		pp := new(big.Int)
		order := new(big.Int)
		// 设置大整数的值
		pp.SetBytes(genesisParams.VerifyParam[:])
		order.SetBytes(genesisParams.Order[:])

		// 初始化
		crypto.CalculatorInitialization(pp, order, genesisParams.TimeParam)
		log.Infoln("Genesis params initialization.")
	}
}
