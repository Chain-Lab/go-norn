package core

import (
	"crypto/sha256"
	"go-chronos/common"
	"math"
)

type node struct {
	data  []byte
	left  *node
	right *node
}

func mergeHash(leftNode *node, rightNode *node) []byte {
	hash := sha256.New()
	hash.Write(leftNode.data)
	hash.Write(rightNode.data)
	return hash.Sum(nil)
}

func BuildMerkleTree(txs []common.Transaction) []byte {
	length := len(txs)
	nodes := make([]node, length)
	for idx := range txs {
		nodes[idx] = node{
			data:  txs[idx].Body.Hash[:],
			left:  nil,
			right: nil,
		}
	}

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
