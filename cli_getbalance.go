package main

import (
	"fmt"
	"log"
)

// 查询余额
func (cli *CLI) getBalance(address, nodeID string) {
	// 校验地址是否合法
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	// 加载db文件
	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()
	// 从地址中解析出公钥hash
	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]
	// by公钥hash查找未花费utxo集合：公钥hash与ouput中的公钥hash相等
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)
	// 统计未花费utxo集合的总额
	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf("Balance of '%s': %d\n", address, balance)
}
