package main

import (
	"crypto/sha256"
)

// MerkleTree represent a Merkle tree
type MerkleTree struct {
	RootNode *MerkleNode //默克尔树根
}

// MerkleNode represent a Merkle tree node
type MerkleNode struct {
	Left  *MerkleNode //左节点
	Right *MerkleNode //右节点
	Data  []byte      //两个节点的hash值
}

// NewMerkleTree creates a new Merkle tree from a sequence of data
func NewMerkleTree(data [][]byte) *MerkleTree { //构造默克尔树
	var nodes []MerkleNode

	if len(data)%2 != 0 { //奇数，最后一个节点double一个
		data = append(data, data[len(data)-1])
	}

	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum) //初始化每个交易的节点结构，左右节点均为空，data为hash的数据
		nodes = append(nodes, *node)
	}

	for i := 0; i < len(data)/2; i++ { //二叉树深度
		var newLevel []MerkleNode

		for j := 0; j < len(nodes); j += 2 { //每层的叶子节点
			node := NewMerkleNode(&nodes[j], &nodes[j+1], nil)
			newLevel = append(newLevel, *node)
		}

		nodes = newLevel
	}

	mTree := MerkleTree{&nodes[0]} //树根

	return &mTree
}

// NewMerkleNode creates a new Merkle tree node
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	mNode := MerkleNode{}

	if left == nil && right == nil { //左右节点都为空的情况下，只随data计算hash
		hash := sha256.Sum256(data)
		mNode.Data = hash[:]
	} else {
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		mNode.Data = hash[:]
	}

	mNode.Left = left
	mNode.Right = right

	return &mNode
}
