package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"math/big"
)

var (
	maxNonce = math.MaxInt64
)

const targetBits = 16

// ProofOfWork represents a proof-of-work
type ProofOfWork struct {
	block  *Block   //即将生成的区块
	target *big.Int //生成区块的难度值
}

// NewProofOfWork builds and returns a ProofOfWork target = 1 << 240; nonce < target
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	pow := &ProofOfWork{b, target}

	return pow
}

func (pow *ProofOfWork) prepareData(nonce int) []byte {
	data := bytes.Join(
		[][]byte{
			pow.block.PrevBlockHash,       //前一个区块hash
			pow.block.HashTransactions(),  //区块所有交易
			IntToHex(pow.block.Timestamp), //时间戳
			IntToHex(int64(targetBits)),   //难度系数
			IntToHex(int64(nonce)),        //难度值
		},
		[]byte{},
	)

	return data
}

// Run performs a proof-of-work
// 不断计算noce和hash值，直到找到一个nonce使得满足hash值小于target
func (pow *ProofOfWork) Run() (int, []byte) {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0 //从零开始递增

	fmt.Printf("Mining a new block")
	for nonce < maxNonce {
		//参与计算hash的数据
		data := pow.prepareData(nonce)
		//计算hash值
		hash = sha256.Sum256(data)
		//步长100000打印一次hash值
		if math.Remainder(float64(nonce), 100000) == 0 {
			fmt.Printf("\r%x", hash)
		}
		hashInt.SetBytes(hash[:])
		//满足条件
		if hashInt.Cmp(pow.target) == -1 {
			break
		} else {
			nonce++ //递增，继续计算
		}
	}
	fmt.Print("\n\n")

	return nonce, hash[:]
}

// Validate validates block's PoW
func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int

	data := pow.prepareData(pow.block.Nonce) //参与计算hash的数据
	hash := sha256.Sum256(data)              //计算hash
	hashInt.SetBytes(hash[:])

	isValid := hashInt.Cmp(pow.target) == -1 //判断是否满足工作量证明

	return isValid
}
