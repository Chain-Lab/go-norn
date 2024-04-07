// Package core
// @Description: Merkle 树处理逻辑
package core

import (
	"crypto/sha256"
	"github.com/chain-lab/go-chronos/common"
	"math"
)

type node struct {
	data  []byte
	left  *node
	right *node
}

// mergeHash
//
//	@Description: 计算 merkle 树左右节点相加后的哈希值
//	@param leftNode - merkle 树左节点
//	@param rightNode - merkle 树右节点
//	@return []byte - 相加后的哈希值
func mergeHash(leftNode *node, rightNode *node) []byte {
	hash := sha256.New()
	hash.Write(leftNode.data)
	hash.Write(rightNode.data)
	return hash.Sum(nil)
}

// BuildMerkleTree
//
//	@Description: Merkle 树构建程序
//	@param txs - 需要构建的交易列表
//	@return []byte - Merkle 树的根哈希值
func BuildMerkleTree(txs []common.Transaction) []byte {
	// 获取交易数量
	length := len(txs)

	if length <= 0 {
		return make([]byte, 32)
	}

	// 最底层一共有对应交易数量的节点
	nodes := make([]node, length)
	for idx := range txs {
		// 初始化节点信息
		nodes[idx] = node{
			data:  txs[idx].Body.Hash[:],
			left:  nil,
			right: nil,
		}
	}

	// 二叉树的结构，其高度为 log2(length)
	height := int(math.Log2(float64(length))) + 1
	
	for i := 0; i < height; i++ {
		levelLength := len(nodes) / 2
		if len(nodes)%2 != 0 {
			levelLength += 1
		}
		level := make([]node, levelLength)
		for j := 0; j < levelLength; j += 2 {
			if j+1 >= levelLength {
				nullNode := node{}
				n := node{
					data:  mergeHash(&nodes[j], &nullNode),
					left:  &nodes[j],
					right: &nullNode,
				}
				level[j/2] = n
			} else {
				n := node{
					data:  mergeHash(&nodes[j], &nodes[j+1]),
					left:  &nodes[j],
					right: &nodes[j+1],
				}
				level[j/2] = n
			}
		}
		nodes = level
		if len(nodes) == 1 {
			break
		}
	}
	return nodes[0].data
}
