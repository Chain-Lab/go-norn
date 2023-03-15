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
	secondQueueInterval = time.Second * 5
	maxBlockMark        = 5
)

// BlockBuffer 维护一个树形结构的缓冲区，保存当前视图下的区块信息
type BlockBuffer struct {
	blockChan  chan *common.Block // 第一区块处理队列，收到即处理
	secondChan chan *common.Block // 第二区块处理队列

	blockProcessList map[uint64]blockList     // 每个高度下的区块列表
	blockMark        map[string]uint8         // 第二队列处理区块的标记信息
	nextBlockMap     map[string]blockList     // 每个区块哈希对应的下一个区块列表
	selectedBlock    map[uint64]*common.Block // 每个高度在当前视图下的最优区块
	knownBlocks      *lru.Cache               // 区块是否在最近处理过的缓存信息

	latestBlockHash   string       // 最新区块哈希，需要注意初始化和维护
	latestBlockHeight uint64       // 当前 db 中存储的最新区块的高度
	bufferedHeight    uint64       // 缓存视图的最新高度
	updateLock        sync.RWMutex // 视图更新的读写锁
}

func NewBlockBuffer() *BlockBuffer {
	return nil
}

// Run 是 BlockBuffer 的线程函数，它依次接收区块进行处理
func (b *BlockBuffer) Run() {
	for {
		select {
		case block := <-b.blockChan:
			prevBlockHash := block.PrevBlockHash()
			blockHash := block.BlockHash()

			if b.knownBlocks.Contains(blockHash) {
				break
			}
			b.knownBlocks.Add(blockHash, nil)

			b.updateLock.RLock()
			list, ok := b.nextBlockMap[prevBlockHash]
			if !ok {
				// 前一个区块不在视图中，放到等待队列中
				// 这里保证了 nextBlockMap 能形成树形结构
				b.secondChan <- block
				b.updateLock.RUnlock()
				break
			}

			blockHeight := block.Header.Height
			selected, ok := b.selectedBlock[blockHeight]
			replaced := false

			if !ok {
				b.selectedBlock[blockHeight] = block
			} else {
				b.selectedBlock[blockHeight], replaced = compareBlock(selected, block)
			}

			if replaced {
				b.updateTreeView(blockHeight)
			}

			b.nextBlockMap[prevBlockHash] = append(list, block)
			processList, ok := b.blockProcessList[blockHeight]

			if !ok {
				b.bufferedHeight = blockHeight
				processList = make(blockList, 0, 15)
			}

			b.blockProcessList[blockHeight] = append(processList, block)
			b.updateLock.RUnlock()
		}
	}
}

// secondProcess 处理第二队列的区块，并且标记
// 如果超过多次无法处理或者过期，就丢弃该区块
func (b *BlockBuffer) secondProcess() {
	// 第二队列处理在第一队列中前一个区块不在缓冲区和链上的区块
	timer := time.NewTimer(secondQueueInterval)
	for {
		select {
		case <-timer.C:
			block := <-b.secondChan
			blockHash := block.BlockHash()
			prevBlockHash := block.PrevBlockHash()
			blockHeight := block.Header.Height

			if block.Header.Height <= b.latestBlockHeight {
				timer.Reset(secondQueueInterval)
				break
			}

			b.updateLock.RLock()
			list, ok := b.nextBlockMap[prevBlockHash]
			if !ok {
				// 区块的前一个哈希不在缓冲树中，也不是最新的区块哈希
				if prevBlockHash != b.latestBlockHash {
					log.Debug("Prev block not exists.")
					// 这样增加值是否会存在问题？
					b.blockMark[blockHash]++

					if b.blockMark[blockHash] < maxBlockMark {
						b.secondChan <- block
					}
					timer.Reset(secondQueueInterval)
					break
				} else {
					list = make(blockList, 0, 15)
				}
			}
			b.nextBlockMap[prevBlockHash] = append(list, block)

			replaced := false
			selected, ok := b.selectedBlock[blockHeight]
			if !ok {
				b.selectedBlock[blockHeight] = block
			} else {
				b.selectedBlock[blockHeight], replaced = compareBlock(selected, block)
			}

			if replaced {
				b.updateTreeView(blockHeight)
			}

			processList, ok := b.blockProcessList[blockHeight]

			if !ok {
				b.bufferedHeight = blockHeight
				processList = make(blockList, 0, 15)
			}

			b.blockProcessList[blockHeight] = append(processList, block)
			b.updateLock.RUnlock()

			timer.Reset(secondQueueInterval)
		}
	}
}

// PopSelectedBlock 推出头部的最优区块什么时候触发？
func (b *BlockBuffer) PopSelectedBlock() *common.Block {
	b.updateLock.RLock()
	defer b.updateLock.RUnlock()
	height := b.latestBlockHeight + 1

	// 检查一下列表是否存在
	_, ok := b.blockProcessList[height]

	if !ok {
		return nil
	}

	selectedBlock := b.selectedBlock[height]
	b.deleteLayer(height)

	b.latestBlockHash = selectedBlock.BlockHash()
	b.latestBlockHeight = height
	return selectedBlock
}

// AppendBlock 添加区块到该缓冲区处理队列
// 传入一个区块，区块会被添加到 channel 中
func (b *BlockBuffer) AppendBlock(block *common.Block) {
	b.blockChan <- block
}

// updateTreeView 更新缓存树上的每个高度的最优区块
func (b *BlockBuffer) updateTreeView(start uint64) {
	prevBlock := b.selectedBlock[start]
	prevBlockHash := prevBlock.BlockHash()
	height := start

	for {
		if height > b.bufferedHeight {
			break
		}
		height++

		if prevBlock == nil {
			b.selectedBlock[height] = nil
			continue
		}

		list, ok := b.nextBlockMap[prevBlockHash]

		if !ok {
			b.selectedBlock[height] = nil
			prevBlock = nil
			continue
		}

		selected := b.selectBlockFromList(list)
		b.selectedBlock[height] = selected
		prevBlock = selected
		prevBlockHash = selected.BlockHash()
	}
}

func (b *BlockBuffer) deleteLayer(layer uint64) {
	list := b.blockProcessList[layer]

	for idx := range list {
		block := list[idx]
		blockHash := block.PrevBlockHash()
		b.nextBlockMap[blockHash] = nil
	}

	b.blockProcessList[layer] = nil
}

// selectBlockFromList 从区块列表中取出优先级最高的区块
func (b *BlockBuffer) selectBlockFromList(list []*common.Block) *common.Block {
	if list == nil {
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
