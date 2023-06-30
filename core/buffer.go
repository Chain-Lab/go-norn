/**
  @author: decision
  @date: 2023/3/13
  @note: 作为区块缓冲区，保存待确认区块
**/

package core

import (
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"go-chronos/common"
	"sync"
	"time"
)

const (
	// todo: 前期测试使用，后面需要修改限制条件
	secondQueueInterval = 100 * time.Millisecond // 区块缓冲视图队列处理延时
	maxBlockMark        = 200                    // 单个区块最多标记多少次不再处理
	maxKnownBlock       = 1024                   // lru 缓冲下最多存放多少区块
	maxQueueBlock       = 512                    // 区块处理第二队列最多存放多少区块
	maxBufferSize       = 12                     // buffer 缓冲多少高度时弹出一个区块
)

// BlockBuffer 维护一个树形结构的缓冲区，保存当前视图下的区块信息
type BlockBuffer struct {
	blockChan  chan *common.Block // 第一区块处理队列，收到即处理
	secondChan chan *common.Block // 第二区块处理队列
	popChan    chan *common.Block // 推出队列

	blockProcessList map[int64]blockList     // 每个高度下的区块列表
	blockMark        map[string]uint8        // 第二队列处理区块的标记信息
	nextBlockMap     map[string]blockList    // 每个区块哈希对应的下一个区块列表
	selectedBlock    map[int64]*common.Block // 每个高度在当前视图下的最优区块
	knownBlocks      *lru.Cache              // 区块是否在最近处理过的缓存信息

	latestBlockHash   string        // 最新区块哈希，需要注意初始化和维护
	latestBlockHeight int64         // 当前 db 中存储的最新区块的高度
	latestBlock       *common.Block // 当前 db 中存储的最新区块
	bufferedHeight    int64         // 缓存视图的最新高度
	updateLock        sync.RWMutex  // 视图更新的读写锁
}

func NewBlockBuffer(latest *common.Block, popChan chan *common.Block) (*BlockBuffer, error) {
	knownBlock, err := lru.New(maxKnownBlock)

	if err != nil {
		log.WithField("error", err).Debug("Create known block cache failed.")
		return nil, err
	}

	buffer := &BlockBuffer{
		blockChan:  make(chan *common.Block),
		secondChan: make(chan *common.Block, maxQueueBlock),
		popChan:    popChan,

		blockProcessList: make(map[int64]blockList),
		blockMark:        make(map[string]uint8),
		nextBlockMap:     make(map[string]blockList),
		selectedBlock:    make(map[int64]*common.Block),
		knownBlocks:      knownBlock,

		latestBlockHeight: latest.Header.Height,
		latestBlockHash:   latest.BlockHash(),
		latestBlock:       latest,
		bufferedHeight:    latest.Header.Height,
	}

	go buffer.Run()
	go buffer.secondProcess()

	return buffer, nil
}

// Run 是 BlockBuffer 的线程函数，它依次接收区块进行处理
func (b *BlockBuffer) Run() {
	for {
		select {
		case block := <-b.blockChan:
			prevBlockHash := block.PrevBlockHash()
			blockHash := block.BlockHash()

			log.WithFields(log.Fields{
				"Hash":     blockHash,
				"PrevHash": prevBlockHash,
				"Height":   block.Header.Height,
			}).Trace("Receive block in channel.")

			if b.knownBlocks.Contains(blockHash) {
				break
			}
			b.knownBlocks.Add(blockHash, nil)

			// todo: 这里对前一个区块是否在视图中的逻辑判断存在问题
			b.updateLock.Lock()
			list, ok := b.nextBlockMap[prevBlockHash]
			if !ok {
				if prevBlockHash != b.latestBlockHash {
					// 前一个区块不在视图中，放到等待队列中
					// 这里保证了 nextBlockMap 能形成树形结构
					b.secondChan <- block
					b.updateLock.Unlock()
					break
				}
				list = make(blockList, 0, 15)
			}

			blockHeight := block.Header.Height
			selected, ok := b.selectedBlock[blockHeight]
			replaced := false

			if selected == nil {
				b.selectedBlock[blockHeight] = block
			} else {
				b.selectedBlock[blockHeight], replaced = compareBlock(selected, block)
			}
			if replaced {
				b.updateTreeView(blockHeight)
			}

			b.nextBlockMap[prevBlockHash] = append(list, block)
			b.nextBlockMap[blockHash] = make(blockList, 0, 15)
			processList, ok := b.blockProcessList[blockHeight]

			if !ok {
				log.WithField("height", blockHeight).Debugln("Extend buffer height by main queue.")
				b.bufferedHeight = blockHeight
				processList = make(blockList, 0, 15)

				if block.Header.Height-b.latestBlockHeight > maxBufferSize {
					b.popChan <- b.PopSelectedBlock()
				}
			}

			b.blockProcessList[blockHeight] = append(processList, block)
			b.updateLock.Unlock()
		}
	}
}

// secondProcess 处理第二队列的区块，并且标记
// 如果超过多次无法处理或者过期，就丢弃该区块
func (b *BlockBuffer) secondProcess() {
	// 第二队列处理在第一队列中前一个区块不在缓冲区和链上的区块
	// 第二队列处理存在的一个问题：在同步的时候区块有可能长时间不能连接上一个区块，会导致同步出现问题
	timer := time.NewTimer(secondQueueInterval)
	var block *common.Block = nil
	for {
		select {
		// 接收计时器到期事件
		case <-timer.C:
			if block == nil {
				block = <-b.secondChan
			}

			// 获取区块的相关信息
			blockHash := block.BlockHash()
			prevBlockHash := block.PrevBlockHash()
			blockHeight := block.Header.Height

			if block.Header.Height <= b.latestBlockHeight {
				timer.Reset(secondQueueInterval)
				break
			}

			b.updateLock.Lock()
			list, _ := b.nextBlockMap[prevBlockHash]
			if list == nil {
				// 区块的前一个哈希不在缓冲树中，也不是最新的区块哈希
				if prevBlockHash != b.latestBlockHash {
					//log.WithField("height", blockHeight).Infof("Prev block #%s not exists.", prevBlockHash)
					// 这样增加值是否会存在问题？
					b.blockMark[blockHash]++

					if b.blockMark[blockHash] >= maxBlockMark {
						block = nil
					}
					timer.Reset(secondQueueInterval)
					b.updateLock.Unlock()
					break
				} else {
					list = make(blockList, 0, 15)
				}
			}
			b.nextBlockMap[prevBlockHash] = append(list, block)
			b.nextBlockMap[blockHash] = make(blockList, 0, 15)

			replaced := false
			selected, _ := b.selectedBlock[blockHeight]
			if selected == nil {
				b.selectedBlock[blockHeight] = block
			} else {
				b.selectedBlock[blockHeight], replaced = compareBlock(selected, block)
			}

			if replaced {
				b.updateTreeView(blockHeight)
			}

			// 一个坑，满足 ok=true 不一定保证数据不是 nil
			processList, _ := b.blockProcessList[blockHeight]

			if processList == nil {
				log.WithField("height", blockHeight).Debugln("Extend buffer height by second queue.")
				b.bufferedHeight = blockHeight
				processList = make(blockList, 0, 15)
				if block.Header.Height-b.latestBlockHeight > maxBufferSize {
					b.popChan <- b.PopSelectedBlock()
				}
			}

			b.blockProcessList[blockHeight] = append(processList, block)
			block = nil
			b.updateLock.Unlock()

			timer.Reset(secondQueueInterval)
		}
	}
}

// PopSelectedBlock 推出头部的最优区块什么时候触发？
// 应该来说是在 bufferedHeight - latestBlockHeight >= maxSize 的情况下触发？
// 以及，收到其他节点发来的已选取区块时触发该逻辑，但是需要确定一下高度和哈希值
func (b *BlockBuffer) PopSelectedBlock() *common.Block {
	// 只在两个 routine 中使用，所以不用上锁
	//b.updateLock.Lock()
	//defer b.updateLock.Unlock()

	height := b.latestBlockHeight + 1
	log.WithField("height", height).Info("Pop block from view.")
	// 检查一下列表是否存在
	_, ok := b.blockProcessList[height]

	if !ok {
		return nil
	}

	selected := b.selectedBlock[height]
	b.deleteLayer(height)

	b.latestBlockHash = selected.BlockHash()
	b.latestBlockHeight = height
	return selected
}

// AppendBlock 添加区块到该缓冲区处理队列
// 传入一个区块，区块会被添加到 channel 中
func (b *BlockBuffer) AppendBlock(block *common.Block) {
	b.blockChan <- block
}

// GetPriorityLeaf 获取当前视图下的最优树叶
func (b *BlockBuffer) GetPriorityLeaf() *common.Block {
	log.Traceln("Start get priority leaf.")
	b.updateLock.RLock()
	defer b.updateLock.RUnlock()

	log.WithFields(log.Fields{
		"start": b.bufferedHeight,
		"end":   b.latestBlockHeight,
	}).Traceln("Start scan all selected.")

	for height := b.bufferedHeight; height > b.latestBlockHeight; height-- {
		if b.selectedBlock[height] != nil {
			log.WithField("height", height).Trace("Return leaf block.")
			return b.selectedBlock[height]
		}
	}
	log.Traceln("All height is nil, return latest block.")
	return b.latestBlock
}

// updateTreeView 更新缓存树上的每个高度的最优区块
func (b *BlockBuffer) updateTreeView(start int64) {
	log.Traceln("Start update buffer tree view.")

	// 从高度 start 开始往后更新
	prevBlock := b.selectedBlock[start]
	prevBlockHash := prevBlock.BlockHash()
	height := int64(start)

	for {
		// 如果高度超过 buffer 中的最高高度则跳过
		if height > b.bufferedHeight {
			break
		}
		height++

		if prevBlock == nil {
			b.selectedBlock[height] = nil
			continue
		}

		// 获取当前情况下的某个区块后续的区块列表
		list, _ := b.nextBlockMap[prevBlockHash]

		if list == nil || len(list) == 0 {
			b.selectedBlock[height] = nil
			prevBlock = nil
			continue
		}

		// 从列表中选出一个最优的区块，然后进入下一次循环
		selected := b.selectBlockFromList(list)
		b.selectedBlock[height] = selected
		prevBlock = selected
		prevBlockHash = selected.BlockHash()
	}
}

func (b *BlockBuffer) deleteLayer(layer int64) {
	list := b.blockProcessList[layer]

	for idx := range list {
		block := list[idx]
		blockHash := block.PrevBlockHash()
		_, ok := b.nextBlockMap[blockHash]
		if ok {
			delete(b.nextBlockMap, blockHash)
		}
	}

	delete(b.blockProcessList, layer)
}

// selectBlockFromList 从区块列表中取出优先级最高的区块
func (b *BlockBuffer) selectBlockFromList(list []*common.Block) *common.Block {
	if list == nil || len(list) == 0 {
		log.Errorln("Block list is empty or null.")
		return nil
	}

	var result *common.Block
	for idx := range list {
		if idx == 0 {
			result = list[0]
			continue
		}
		result, _ = compareBlock(result, list[idx])
	}
	return result
}

// compareBlock 对比区块优先级，后面考虑一下处理异常
// todo： 将优先级策略写入到创世区块中
func compareBlock(origin *common.Block, block *common.Block) (*common.Block, bool) {
	if len(origin.Transactions) == len(block.Transactions) {
		if origin.Header.Timestamp < block.Header.Timestamp {
			return origin, false
		}
		return block, true
	}

	if len(origin.Transactions) > len(block.Transactions) {
		return origin, false
	}

	return block, true
}
