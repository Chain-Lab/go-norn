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
	"go-chronos/utils"
	karmem "karmem.org/golang"
	"math/big"
	"sync"
	"time"
)

const (
	maxBlockCache       = 1024
	maxTransactionCache = 32768
	maxBlockProcessList = 12
	maxBlockChannel     = 128
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
	latestHeight int64
	latestLock   sync.RWMutex

	bufferChan    chan *common.Block
	buffer        *BlockBuffer
	appendLock    sync.RWMutex
	genesisParams *common.GenesisParams
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
		db:            db,
		dbWriterQueue: make(chan *common.Block),

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
func (bc *BlockChain) PackageNewBlock(txs []common.Transaction, params *common.GeneralParams) (*common.Block, error) {
	log.Traceln("Start package new block.")
	paramsBytes, err := utils.SerializeGeneralParams(params)

	if err != nil {
		log.WithField("error", err).Errorln("Serialize params failed while package block.")
		return nil, err
	}

	bestBlock := bc.buffer.GetPriorityLeaf()
	publicKey, err := hex.DecodeString(config.String("pub"))

	if err != nil {
		log.Errorln("Get public key from config failed.")
		return nil, err
	}

	merkleRoot := BuildMerkleTree(txs)
	block := common.Block{
		Header: common.BlockHeader{
			Timestamp:     time.Now().UnixMilli(),
			PrevBlockHash: bestBlock.Header.BlockHash,
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(merkleRoot),
			Height:        bestBlock.Header.Height + 1,
			PublicKey:     [33]byte(publicKey),
			Params:        paramsBytes,
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
			Timestamp:     time.Now().UnixMilli(),
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
	value, ok := bc.blockHeightMap.Get(height)
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
	blockHash := block.BlockHash()
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

	blockHash := common.Hash(block.Header.BlockHash)

	// 获取对应哈希的区块，如果区块存在，说明链上已经存在该区块
	_, err = bc.GetBlockByHash(&blockHash)
	if err == nil {
		log.WithField("hash", block.BlockHash()).Warning("Block exists.")
		return
	}

	latestBlock, err := bc.GetLatestBlock()
	if !block.IsGenesisBlock() {
		if err != nil {
			log.WithField("error", err).Debugln("Get latest block failed.")
			return
		}

		if !isPrevBlock(latestBlock, block) {
			log.Errorln("Block error, prev block hash not match.")
			return
		}
	} else {
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
	keys[2] = utils.BlockHeight2DBKey(block.Header.Height)
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
	seed := new(big.Int)
	proof := new(big.Int)

	// todo: 异常处理
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
	calculator := crypto.GetCalculatorInstance()
	calculator.AppendNewSeed(seed, proof)
}

func (bc *BlockChain) AppendBlockTask(block *common.Block) {
	if block.IsGenesisBlock() {
		bc.InsertBlock(block)
		return
	}

	log.Infoln("Append block to buffer.")
	bc.buffer.AppendBlock(block)
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

		pp := new(big.Int)
		order := new(big.Int)
		// 设置大整数的值
		pp.SetBytes(genesisParams.VerifyParam[:])
		order.SetBytes(genesisParams.Order[:])

		// 初始化
		crypto.CalculatorInitialization(pp, order, genesisParams.TimeParam)
	}
}
