/**
  @author: decision
  @date: 2023/3/14
  @note: 为 karmem 生成的 block 代码添加的辅助函数
**/

package common

import "encoding/hex"

// BlockHash 获取区块的 hex 格式的区块哈希值
func (b *Block) BlockHash() string {
	return hex.EncodeToString(b.Header.BlockHash[:])
}

// PrevBlockHash 获取区块的 hex 格式的前一个区块哈希值
func (b *Block) PrevBlockHash() string {
	return hex.EncodeToString(b.Header.PrevBlockHash[:])
}

// IsGenesisBlock 判断区块是否为创世区块，通过判断前一个区块的哈希值
func (b *Block) IsGenesisBlock() bool {
	prevBlockHash := b.PrevBlockHash()

	// todo: 这里还只是为了实现，后面需要改为高效的算法
	if prevBlockHash == "00000000000000000000000000000000" {
		return true
	}
	return false
}
