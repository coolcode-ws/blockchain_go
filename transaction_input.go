package main

import "bytes"

// 交易输入：引用了之前一笔交易的输出
type TXInput struct {
	Txid      []byte //之前一笔交易输出hash：coinbase交易为空
	Vout      int    //之前一笔交易输出的索引：coinbase交易为-1
	Signature []byte //签名，coinbase交易为空，交易输入引用了之前一笔交易的输出，交易是通过这个脚本来锁定某个交易输出，只有被锁定的人解锁后才能花费
	//提供了一个可作用于交易输出ScriptPubKey的数据，决定这笔交易输出能否解锁
	PubKey []byte //公钥，coinbase为随机值
}

// UsesKey checks whether the address initiated the transaction
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}
