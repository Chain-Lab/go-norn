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
	"go-chronos/metrics"
	"sync"
	"time"
)

const (
	// todo: 前期测试使用，后面需要修改限制条件
	secondQueueInterval = 100 * time.Millisecond // 区块缓冲视图队列处理延时
	maxBlockMark        = 200                    // 单个区块最多标记多少次不再处理
	maxKnownBlock       = 1024                   // lru 缓冲下最多存放多少区块
	maxQueueBlock       = 512                    // 区块处理第二队列最多存放多少区块
	maxBufferSize       = 32                     // buffer 缓冲多少高度时弹出一个区块
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

	updateLock sync.RWMutex // 视图更新的读写锁
}

func NewBlockBuffer(latest *common.Block, popChan chan *common.Block) (*BlockBuffer, error) {
	knownBlock, err := lru.New(maxKnownBlock)

	if err != nil {
		log.WithField("error", err).Debug("Create known block cache failed.")
		return nil, err
	}

	buffer := &BlockBuffer{
		blockChan:  make(chan *common.Block, maxQueueBlock),
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

	metrics.RoutineCreateHistogramObserve(8)
	go buffer.Process()
	go buffer.secondProcess()

	return buffer, nil
}

// Process 是 BlockBuffer 的线程函数，它依次接收区块进行处理
func (b *BlockBuffer) Process() {
	for {
		select {
		case block := <-b.blockChan:

			// 取得区块和其前一个区块的哈希值
			prevBlockHash := block.PrevBlockHash()
			blockHash := block.BlockHash()

			blockHeight := block.Header.Height

			// 如果当前区块低于最高区块高度，终止处理
			if blockHeight < b.latestBlockHeight {
				log.Warningln("Block height too low.")
				break
			}

			log.WithFields(log.Fields{
				"Hash":     blockHash[:8],
				"PrevHash": prevBlockHash[:8],
				"Height":   block.Header.Height,
			}).Trace("Receive block in channel.")

			// 如果区块已知，则不再放入到缓冲队列
			if b.knownBlocks.Contains(blockHash) {
				break
			}
			b.knownBlocks.Add(blockHash, nil)

			// todo: 这里对前一个区块是否在视图中的逻辑判断存在问题
			// 根据前一个区块的哈希值查询到区块列表，如果存在则继续
			b.updateLock.Lock()
			list, _ := b.nextBlockMap[prevBlockHash]
			if list == nil {
				log.WithField("prev", prevBlockHash).Infoln("List is nil.")
				if prevBlockHash != b.latestBlockHash {
					// 前一个区块不在视图中，放到等待队列中，这里保证了 nextBlockMap 能形成树形结构
					log.Infoln("Pop block to second channel.")
					b.secondChan <- block
					b.updateLock.Unlock()
					break
				}
				// 前一个区块在视图中，创建新的列表
				list = make(blockList, 0, 15)
			}

			// 获取这个区块高度下已经选定的区块
			selected, _ := b.selectedBlock[blockHeight]
			prevSelected, _ := b.selectedBlock[blockHeight-1]
			replaced := false
			metrics.BlockBufferCountMetricsInc()

			//log.Infoln(b.latestBlock.BlockHash())
			//log.Infoln(block.PrevBlockHash())
			if selected == nil && ((prevSelected != nil && prevSelected.BlockHash() == block.PrevBlockHash()) ||
				b.latestBlock.BlockHash() == block.PrevBlockHash()) {
				// 如果某个高度下不存在选取的区块， 则默认设置为当前的区块
				b.selectedBlock[blockHeight] = block
				replaced = true
			} else if selected != nil {
				// 否则对区块进行比较，并且返回是否进行替换
				b.selectedBlock[blockHeight], replaced = compareBlock(selected, block)
			}

			if replaced {
				// 如果对该高度下的区块进行了替换，则需要更新视图
				b.updateTreeView(blockHeight)
			}

			b.nextBlockMap[prevBlockHash] = append(list, block)
			b.nextBlockMap[blockHash] = make(blockList, 0, 15)
			log.WithField("height", blockHeight).Debugln("[First channel] Create block map.")
			processList, _ := b.blockProcessList[blockHeight]

			if processList == nil {
				log.WithField("height", blockHeight).Debugln("Extend buffer height by main queue.")
				b.bufferedHeight = blockHeight
				processList = make(blockList, 0, 15)
			}

			b.blockProcessList[blockHeight] = append(processList, block)

			if block.Header.Height-b.latestBlockHeight > maxBufferSize {
				b.popChan <- b.PopSelectedBlock()
			}

			b.updateLock.Unlock()
		}
	}
}

func (b *BlockBuffer) secondProcess() {
	timer := time.NewTimer(secondQueueInterval)
	var block *common.Block = nil
	for {
		select {
		// 接收计时器到期事件
		case <-timer.C:
			if block == nil {
				block = <-b.secondChan
				log.WithField("height", block.Header.Height).Debugln("Pop block from second channel.")
			}

			// 获取区块的相关信息
			blockHash := block.BlockHash()
			prevBlockHash := block.PrevBlockHash()
			blockHeight := block.Header.Height

			if block.Header.Height <= b.latestBlockHeight {
				block = nil
				log.WithFields(log.Fields{
					"height": block.Header.Height,
					"latest": b.latestBlockHeight,
				}).Warningln("Block height too low.")
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
					//b.blockMark[blockHash]++
					//
					//if b.blockMark[blockHash] >= maxBlockMark {
					//	block = nil
					//}
					timer.Reset(secondQueueInterval)
					b.updateLock.Unlock()
					break
				} else {
					list = make(blockList, 0, 15)
				}
			}
			b.nextBlockMap[prevBlockHash] = append(list, block)
			b.nextBlockMap[blockHash] = make(blockList, 0, 15)
			log.WithField("height", blockHeight).Debugln("[Second channel] Create block map.")

			replaced := false
			selected, _ := b.selectedBlock[blockHeight]
			prevSelected, _ := b.selectedBlock[blockHeight-1]
			metrics.BlockBufferCountMetricsInc()

			if selected == nil && ((prevSelected != nil && prevSelected.BlockHash() == block.PrevBlockHash()) ||
				b.latestBlock.BlockHash() == block.PrevBlockHash()) {
				// 如果某个高度下不存在选取的区块， 则默认设置为当前的区块
				b.selectedBlock[blockHeight] = block
				replaced = true
			} else if selected != nil {
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
			}

			b.blockProcessList[blockHeight] = append(processList, block)

			if block.Header.Height-b.latestBlockHeight > maxBufferSize {
				b.popChan <- b.PopSelectedBlock()
			}

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
	log.WithField("height", height).Debugln("Pop block from view.")
	// 检查一下列表是否存在
	lst, _ := b.blockProcessList[height]

	if lst == nil {
		return nil
	}

	selected := b.selectedBlock[height]
	b.deleteLayer(height)

	b.latestBlockHash = selected.BlockHash()
	b.latestBlockHeight = height
	b.latestBlock = selected
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
	prevBlock := b.selectedBlock[start]    // 前一个区块
	prevBlockHash := prevBlock.BlockHash() // 前一个区块的哈希值
	height := start

	for {
		// 如果高度超过 buffer 中的最高高度则跳过
		if height > b.bufferedHeight {
			break
		}
		height++ // 处理下一个区块

		if prevBlock == nil {
			// 如果前一个区块为空，则继续，并且设置当前高度下的区块为 nil
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
		lst, _ := b.nextBlockMap[blockHash]
		// 这里删除的原理是删除前面的 map 让区块由系统回收
		if lst != nil {
			metrics.BlockBufferCountMetricsDec(len(lst))
			delete(b.nextBlockMap, blockHash)
		}
		//metrics.BlockBufferCountMetricsDec(1)
	}

	// 再删除高度对应的区块列表，使得blocks没有指向的指针，由 golang 进行回收
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
// upd [230813]: 对比的前提是两者的前一个区块是同一个区块，否则无法保证连续
// todo： 将优先级策略写入到创世区块中
func compareBlock(origin *common.Block, block *common.Block) (*common.Block, bool) {
	if block.PrevBlockHash() != origin.PrevBlockHash() {
		return origin, false
	}

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
