// Package core
// @Description: 区块链类代码，主要作为单例使用
// 为了避免潜在的数据不一致的情况，任何情况下不要对一个 block 实例进行数据的修改
// 如果需要对区块进行校验或者哈希的修改，对数据进行深拷贝得到一份复制来进行处理
package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"github.com/chain-lab/go-norn/common"
	"github.com/chain-lab/go-norn/crypto"
	"github.com/chain-lab/go-norn/interfaces"
	"github.com/chain-lab/go-norn/metrics"
	"github.com/gookit/config/v2"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"time"

	//handler "github.com/chain-lab/go-norn/node"
	"github.com/chain-lab/go-norn/utils"
	karmem "karmem.org/golang"
	"math/big"
	"sync"
)

const (
	maxBlockCache         = 64    // 区块 LRU 缓存大小
	maxTransactionCache   = 40960 // 交易 LRU 缓存大小
	maxBlockProcessList   = 12    // （已弃用）区块处理队列长度
	maxBlockChannel       = 128   // 区块缓冲长度
	maxDbChannel          = 256   // 数据库缓冲长度
	transactionStartIndex = 3     // 批量存放数据时，交易数据开始的位置

	setCommandString    = "set"
	appendCommandString = "append"
)

type BlockChain struct {
	// 数据库相关的成员变量
	db interfaces.DBInterface

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

	// 数据存放处理与对应的任务 channel
	dp     *DataProcessor
	dpChan chan *DataTask

	// genesisParams 当前所维护的链的创世区块参数
	genesisParams *common.GenesisParams
	genesisTime   int64
}

// NewBlockchain
//
//	@Description: 创建一个 BlockChain ，需要传入一个 LevelDB 实例 db
//	@param db - 已初始化的 levelDB 数据库实例
//	@return *BlockChain - 实例化的数据库处理对象，如果出错返回 nil
func NewBlockchain(db interfaces.DBInterface) *BlockChain {
	// 实例化一系列的 cache 并处理可能出现的错误，创建三个 LRU 缓存
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

	dp := NewDataProcessor()
	dp.db = db
	go dp.Run()

	// 使用数据库实例 db 实例化一个 Blockchain 对象
	chain := &BlockChain{
		db:             db,
		blockHeightMap: blockHeightMap,
		blockCache:     blockCache,
		txCache:        txCache,

		latestBlock:  nil,
		latestHeight: -1,

		dp:     dp,
		dpChan: dp.taskChannel,

		bufferChan: make(chan *common.Block, maxBlockChannel),
	}

	// 初始化最新区块，从数据库中进行读取
	latest, _ := chain.GetLatestBlock()

	// 如果最新区块存在，说明当前不是新的区块链，处理缓冲逻辑并且读取创世参数
	if latest != nil {
		log.Traceln("Block database is not null, create buffer.")
		chain.createBlockBuffer(latest)

		// 加载创世区块参数
		genesis, _ := chain.GetBlockByHeight(0)
		chain.genesisInitialization(genesis)
	}

	// 区块处理协程启动
	metrics.RoutineCreateCounterObserve(7)
	go chain.BlockProcessRoutine()
	return chain
}

// BlockProcessRoutine
//
//	@Description: 接收对应了缓冲区弹出的 channel中区块的协程，从 chan 中读取区块并且调用 insertBlock 进行处理
//	@receiver BlockChain 实例
func (bc *BlockChain) BlockProcessRoutine() {
	for {
		select {
		case block := <-bc.bufferChan:
			bc.insertBlock(block)
		}
	}
}

// PackageNewBlock
//
//	@Description: 打包新的区块，传入交易序列等信息
//	@receiver BlockChain 实例
//	@param txs - 选择好的交易数组
//	@param timestamp - 当前的逻辑时间戳
//	@param params - 当前的 VDF 计算信息
//	@param packageInterval - 打包的间隔
//	@return *common.Block - 完成打包的区块，如果出错返回为 nil
//	@return error - 错误信息
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
	log.Infof("Package block height #%d.", bestBlock.Header.Height+1)
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
		log.WithField("error", err).Errorln("Serialize block header failed.")
		return nil, err
	}

	// 计算区块的哈希值并填入到区块头
	hash := sha256.New()
	hash.Write(byteBlockHeaderData)
	blockHash := common.Hash(hash.Sum(nil))
	block.Header.BlockHash = blockHash

	// 指标记录，本次区块打包耗时
	metrics.PackageBlockMetricsSet(float64(time.Since(packageStart).Milliseconds()))
	return &block, nil
}

// NewGenesisBlock
//
//	@Description: 创建创世区块
//	@receiver BlockChain 实例
func (bc *BlockChain) NewGenesisBlock() {
	// 获取本地高度为 0 的区块，检查创世区块是否存在
	block, _ := bc.GetBlockByHeight(0)
	if block != nil {
		log.Errorln("Genesis block already exists.")
		return
	}

	// 读取配置文件中的公钥，并解码
	publicKey, err := hex.DecodeString(config.String("consensus.pub"))
	if err != nil {
		log.Errorln("Get public key from config failed.")
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
	genesisBlock := common.Block{
		Header: common.BlockHeader{
			Timestamp: time.Now().UnixMilli(),
			//Timestamp:     handler.GetLogicClock(),
			PrevBlockHash: nullHash,
			BlockHash:     [32]byte{},
			MerkleRoot:    [32]byte(nullHash),
			Height:        0,
			Params:        genesisParamsBytes,
			PublicKey:     [33]byte(publicKey),
		},
		Transactions: []common.Transaction{},
	}

	// 序列化区块得到字节数组数据
	byteBlockHeaderData, err := utils.SerializeBlockHeader(&genesisBlock.Header)
	if err != nil {
		log.WithField("error", err).Errorln("Serialize block header failed.")
		return
	}

	// 对字节数据进行哈希并将哈希值填充到区块头
	hash := sha256.New()
	hash.Write(byteBlockHeaderData)
	blockHash := common.Hash(hash.Sum(nil))
	genesisBlock.Header.BlockHash = blockHash

	bc.AppendBlockTask(&genesisBlock)
}

// GetLatestBlock
//
//	@Description: 获取当前的最新区块
//	@receiver BlockChain 对象
//	@return *common.Block - 当前区块链的最新区块
//	@return error - 错误信息
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

	// 如果从数据库中拉取了最新区块的信息，需要更新 latest 缓存数据
	bc.latestLock.Lock()
	bc.latestBlock = block
	bc.latestHeight = block.Header.Height
	bc.latestLock.Unlock()

	return block, nil
}

// GetBlockByHash
//
//	@Description: 通过哈希值获取区块，如果获取失败返回 nil
//	@receiver BlockChain 实例
//	@param hash - 所需要获取区块的哈希值
//	@return *common.Block - 对应区块哈希值的区块， 如果不存在返回 nil 和对应的错误
//	@return error - 错误信息
func (bc *BlockChain) GetBlockByHash(hash *common.Hash) (*common.Block, error) {
	// 先查询缓存是否命中
	strHash := hex.EncodeToString(hash[:])
	value, hit := bc.blockCache.Get(strHash)

	// 如果命中缓存则直接返回
	if hit {
		// 命中缓存， 直接返回区块
		return value.(*common.Block), nil
	}

	// 组合得到区块在数据库内的存放索引
	blockKey := utils.BlockHash2DBKey(*hash)
	byteBlockData, err := bc.db.Get(blockKey)
	// 从数据库中获取的错误处理
	if err != nil {
		log.WithField("error", err).Debugln("Get block data in database failed.")
		return nil, err
	}

	// 成功从数据库中获取，进行 cache 的更新
	block, _ := utils.DeserializeBlock(byteBlockData)
	bc.writeBlockCache(block)

	return block, nil
}

// GetBlockByHeight
//
//	@Description: 根据高度拉取区块，需要考虑一下同步时是否换用其他的函数
//	@receiver BlockChain 实例
//	@param height - 需要获取的区块的高度
//	@return *common.Block - 对应高度的区块， 如果不存在返回 nil 和对应的错误
//	@return error - 错误信息
func (bc *BlockChain) GetBlockByHeight(height int64) (*common.Block, error) {
	// 先判断一下高度
	if height > bc.latestHeight && bc.latestHeight != -1 {
		return nil, errors.New("Block height error.")
	}

	// 查询缓存里面是否有区块的信息
	value, _ := bc.blockHeightMap.Get(height)
	if value != nil {
		// 缓存中得到区块的哈希值，然后通过哈希值调用 GetBlockByHash 获取区块链
		blockHash := common.Hash(value.([32]byte))
		block, err := bc.GetBlockByHash(&blockHash)

		if err != nil {
			log.WithField("error", err).Errorln("Get block by hash failed.")
			return nil, err
		}

		return block, nil
	}

	// 查询数据库，得到高度对应的区块哈希
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

// writeBlockCache
//
//	@Description: 向 哈希 -> 区块 的 cache 下添加区块信息
//	@receiver BlockChain 实例
//	@param block - 需要写入到缓存的区块实例
func (bc *BlockChain) writeBlockCache(block *common.Block) {
	bc.blockHeightMap.Add(block.Header.Height, block.Header.BlockHash)
	blockHash := block.BlockHash()
	bc.blockCache.Add(blockHash, block)
}

// writeTxCache
//
//	@Description: 向 哈希 -> 交易的 cache 中添加交易
//	@receiver BlockChain 实例
//	@param transaction - 需要写入到缓存的交易实例
func (bc *BlockChain) writeTxCache(transaction *common.Transaction) {
	hash := transaction.Body.Hash
	strHash := hex.EncodeToString(hash[:])
	bc.txCache.Add(strHash, transaction)
}

// insertBlock
//
//	@Description: 将区块插入到数据库的处理逻辑，通常只有从区块缓存区中弹出后才能被处理，创世区块直接被该函数处理
//	@receiver BlockChain 实例
//	@param block - 需要插入数据库的区块
func (bc *BlockChain) insertBlock(block *common.Block) {
	count := len(block.Transactions)

	blockHash := common.Hash(block.Header.BlockHash)
	pool := GetTxPoolInst()

	// 获取对应哈希的区块，如果区块存在，说明链上已经存在该区块
	_, err := bc.GetBlockByHash(&blockHash)
	if err == nil {
		log.WithField("hash", block.BlockHash()[:8]).Warning("Block exists.")
		return
	}

	// 获取最新的区块
	latestBlock, err := bc.GetLatestBlock()
	if !block.IsGenesisBlock() {
		// 插入区块非创世区块，需要检查拉取最新区块 是否成功 以及 前一个区块的哈希值是否符合
		if err != nil {
			log.WithField("error", err).Debugln("Get latest block failed.")
			return
		}

		// 如果前一个区块的哈希值错误，则插入该区块报错
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

	// 锁定 BlockChain 实例的最新区块，将当前区块写入缓存
	bc.latestLock.Lock()
	if block.Header.Height <= bc.latestHeight {
		bc.latestLock.Unlock()
		return
	}
	bc.latestBlock = block
	bc.latestHeight = block.Header.Height
	bc.latestLock.Unlock()
	bc.writeBlockCache(block)

	// 交易列表添加到数据库前需要预留三个位置：
	// 修改 latest的哈希值、区块、区块高度对应的哈希
	keys := make([][]byte, count+transactionStartIndex)
	values := make([][]byte, count+transactionStartIndex)

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

	// 从 index = 3 的位置开始，写入的数据信息均为交易
	for idx := range block.Transactions {
		// 依次对交易进行处理
		// todo： 或许需要校验一下交易是否合法
		tx := block.Transactions[idx]
		tx.Body.Height = block.Header.Height
		tx.Body.BlockHash = block.Header.BlockHash
		tx.Body.Index = int64(idx)

		// 从交易池中移除某个交易
		pool.RemoveTx(tx.Body.Hash)
		txWriter := karmem.NewWriter(1024)
		// 交易的索引：tx#{hash}
		keys[idx+transactionStartIndex] = append([]byte("tx#"), tx.Body.Hash[:]...)
		_, err := tx.WriteAsRoot(txWriter) // 序列化及处理逻辑
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
				"hash":  hex.EncodeToString(tx.Body.Hash[:])[:8],
			}).Errorln("Encode transaction failed.")
			return
		}

		values[idx+transactionStartIndex] = txWriter.Bytes()
		metrics.TransactionInsertInc()
		// 数据处理事件的逻辑，如果交易的 data 字段存在数据，则开始尝试处理
		// 否则，当前的交易处理完成，继续后续交易的处理
		if tx.Body.Data == nil {
			continue
		}

		// 尝试对 data 字段进行 command 类的反序列化，如果反序列化失败则为无效的指令，跳过该交易
		dataBytes := tx.Body.Data
		dc, err := utils.DeserializeDataCommand(dataBytes)
		if err != nil {
			continue
		}

		// 这里的两个逻辑都类似，将 key、value 取出后得到对应的 DataTask 类，然后丢给处理协程
		if string(dc.Opt) == setCommandString {
			// 如果 data 字段的命令为 set <key> <value> 类的命令
			contractAddr := tx.Body.Receiver
			key := dc.Key
			value := dc.Value

			task := DataTask{
				Type:    setCommandString,
				Hash:    tx.Body.Hash,
				Height:  block.Header.Height,
				Address: contractAddr[:],
				Key:     key,
				Value:   value,
			}

			bc.dpChan <- &task
		} else if string(dc.Opt) == appendCommandString {
			// 如果 data 字段的命令为 append <key> <value> 类的命令
			contractAddr := tx.Body.Receiver
			key := dc.Key
			value := dc.Value

			task := DataTask{
				Type:    appendCommandString,
				Hash:    tx.Body.Hash,
				Height:  block.Header.Height,
				Address: contractAddr[:],
				Key:     key,
				Value:   value,
			}

			bc.dpChan <- &task
		}
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

		calculator := crypto.GetCalculatorInstance()
		calculator.AppendNewSeed(seed, proof)
	}
	//	params, _ := utils.DeserializeGeneralParams(block.Header.Params)
	//	// todo: 将编码转换的过程放入到VRF代码中
	//	seed.SetBytes(params.Result)
	//	proof.SetBytes(params.Proof)
	//}

	//// 向 VDF 的计算添加区块下的信息
	metrics.BlockHeightSet(block.Header.Height)
}

// AppendBlockTask
//
//	@Description: 向区块缓冲视图中添加区块处理任务
//	@receiver BlockChain 实例
//	@param block - 需要添加到任务队列的的区块
func (bc *BlockChain) AppendBlockTask(block *common.Block) {
	if block.IsGenesisBlock() {
		bc.insertBlock(block)
		return
	}

	log.Debugln("Append block to buffer.")
	bc.buffer.AppendBlock(block)
}

// InsertBlock
//
//	@Description: 向区块的缓冲区添加区块
//	@receiver BlockChain 实例
//	@param block - 需要添加到 BlochBuffer 进行处理的区块
func (bc *BlockChain) InsertBlock(block *common.Block) {
	bc.bufferChan <- block
}

// GetTransactionByHash
//
//	@Description: 通过交易的哈希值来获取交易
//	@receiver BlockChain 实例
//	@param hash - 需要获取交易的哈希值
//	@return *common.Transaction - 对应哈希值的交易实例，如果不存在则返回 nil
//	@return error - 错误信息
func (bc *BlockChain) GetTransactionByHash(hash common.Hash) (*common.Transaction, error) {
	// 得到字符串类的哈希值，首先查询缓存
	strHash := hex.EncodeToString(hash[:])
	value, hit := bc.txCache.Get(strHash)
	if hit {
		return value.(*common.Transaction), nil
	}

	// 缓存不命中，尝试从数据库中获取
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

	// 将交易写入到缓存
	bc.writeTxCache(transaction)
	return transaction, nil
}

// Height
//
//	@Description: 获取当前区块链的最新高度
//	@receiver BlockChain 实例
//	@return int64 - 区块链的最新高度
func (bc *BlockChain) Height() int64 {
	bc.latestLock.RLock()
	defer bc.latestLock.RUnlock()
	return bc.latestHeight
}

// BufferedHeight
//
//	@Description: 获取当前区块链缓冲区的最新高度
//	@receiver BlockChain 实例
//	@return int64 - 区块链缓冲区的最新高度
func (bc *BlockChain) BufferedHeight() int64 {
	if bc.buffer == nil {
		return 0
	}
	return bc.buffer.bufferedHeight
}

// BufferFull
//
//	@Description: 查询区块的缓冲区是否已满
//	@receiver BlockChain 实例
//	@return bool - 区块链缓冲区是否已满
func (bc *BlockChain) BufferFull() bool {
	if bc.buffer == nil {
		return false
	}
	return bc.buffer.bufferFull
}

// createBlockBuffer
//
//	@Description: 创建区块缓冲区
//	@receiver BlockChain 实例
//	@param latest - 当前的最新区块，通常是数据库中存放的最新区块
func (bc *BlockChain) createBlockBuffer(latest *common.Block) {
	// todo: 需要处理报错
	log.Traceln("Create new block buffer.")
	var err error
	bc.buffer, err = NewBlockBuffer(latest, bc.bufferChan)

	if err != nil {
		log.WithError(err).Errorln("Create new block buffer failed.")
	}
}

// genesisInitialization
//
//	@Description: 传入创世区块，读取区块数据并初始化 VDF 计算
//	@receiver BlockChain 实例
//	@param block - 创世区块
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

// ReadAddressData
//
//	@Description: 读取某个地址下存放的数据，v1.1.0 新增的读写数据功能
//	@receiver BlockChain 实例
//	@param address - 数据存放地址
//	@param key - 数据的 key
//	@return []byte - 对应地址、key下的数据
//	@return error - 错误信息
func (bc *BlockChain) ReadAddressData(address, key string) ([]byte, error) {
	db := bc.db
	addr, err := hex.DecodeString(address)
	// 对地址进行解码
	if err != nil {
		log.WithError(err).Errorln("Decode address failed.")
		return nil, err
	}

	// 拼接在数据库中的索引：address#key
	dbKey := utils.DataAddressKey2DBKey(addr, []byte(key))

	// 读取对应的数据，如果读取失败返回 nil
	data, err := db.Get(dbKey)
	if err != nil {
		log.WithError(err).Debugln("Get database data failed.")
		return nil, err
	}

	return data, nil
}

// isPrevBlock
//
//	@Description: 分析一个区块是否为另外一个区块的前一个区块
//	@param prev - 前一个区块
//	@param block - 当前区块
//	@return bool - 如果 prev 是 block 的前一个区块，返回 True
func isPrevBlock(prev *common.Block, block *common.Block) bool {
	prevBlockHash := prev.Header.BlockHash[:]
	blockPrevHash := block.Header.PrevBlockHash[:]
	return bytes.Compare(prevBlockHash, blockPrevHash) == 0
}
