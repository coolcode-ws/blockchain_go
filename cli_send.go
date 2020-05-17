package main

import (
	"fmt"
	"log"
)

// 转账
func (cli *CLI) send(from, to string, amount int, nodeID string, mineNow bool) {
	//校验转出和转入地址合法性
	if !ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}
	//加载区块链和utxo集合
	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()
	//加载钱包
	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)
	//创建转账交易
	tx := NewUTXOTransaction(&wallet, to, amount, &UTXOSet)
	//判断是否挖矿
	if mineNow {
		//创建coinbase交易：pubkey为随机值，签名为空
		cbTx := NewCoinbaseTX(from, "")
		txs := []*Transaction{cbTx, tx}
		//挖矿，产生新区块
		newBlock := bc.MineBlock(txs)
		//更新utxo集合
		UTXOSet.Update(newBlock)
	} else {
		//发送交易给其中一个节点
		sendTx(knownNodes[0], tx)
	}

	fmt.Println("Success!")
}
