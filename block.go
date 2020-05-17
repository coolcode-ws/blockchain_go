package main

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

// 区块
type Block struct {
	Timestamp     int64          //时间戳
	Transactions  []*Transaction //交易
	PrevBlockHash []byte         //前一个区块的hash，区块链就是一个个区块串联而成
	Hash          []byte         //当前区块hash
	Nonce         int            //工作量证明：难度值
	Height        int            //区块高度
}

// 创建并返回一个区块
func NewBlock(transactions []*Transaction, prevBlockHash []byte, height int) *Block {
	block := &Block{time.Now().Unix(), transactions, prevBlockHash, []byte{}, 0, height}
	pow := NewProofOfWork(block) // 工作量证明
	nonce, hash := pow.Run()

	block.Hash = hash[:] //难度值和当前区块hash，由工作量证明计算NewProofOfWork
	block.Nonce = nonce

	return block
}

//创建创世块：coinbase交易，第一个区块：只有一个coinbase交易；前一个区块的hash为空；高度为0
func NewGenesisBlock(coinbase *Transaction) *Block {
	return NewBlock([]*Transaction{coinbase}, []byte{}, 0)
}

// HashTransactions returns a hash of the transactions in the block
func (b *Block) HashTransactions() []byte {
	var transactions [][]byte

	for _, tx := range b.Transactions {
		transactions = append(transactions, tx.Serialize()) //所有交易
	}
	mTree := NewMerkleTree(transactions) //计算默克尔树

	return mTree.RootNode.Data
}

// Serialize serializes the block
func (b *Block) Serialize() []byte { //序列化
	var result bytes.Buffer
	encoder := gob.NewEncoder(&result)

	err := encoder.Encode(b)
	if err != nil {
		log.Panic(err)
	}

	return result.Bytes()
}

// DeserializeBlock deserializes a block
func DeserializeBlock(d []byte) *Block {
	var block Block

	decoder := gob.NewDecoder(bytes.NewReader(d)) //反序列化
	err := decoder.Decode(&block)
	if err != nil {
		log.Panic(err)
	}

	return &block
}
