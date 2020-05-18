package main

import (
	"bytes"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

var nodeAddress string
var miningAddress string
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = [][]byte{}
var mempool = make(map[string]Transaction)

type addr struct {
	AddrList []string //地址列表
}

type block struct {
	AddrFrom string //对端地址
	Block    []byte //区块
}

type getblocks struct {
	AddrFrom string //对端地址
}

type getdata struct {
	AddrFrom string //对端地址
	Type     string //类型
	ID       []byte //id
}

type inv struct {
	AddrFrom string   //对端地址
	Type     string   //类型
	Items    [][]byte //items
}

type tx struct {
	AddFrom     string //对端地址
	Transaction []byte //交易信息
}

type verzion struct {
	Version    int    //版本号
	BestHeight int    //最新区块高度
	AddrFrom   string //对端地址
}

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func extractCommand(request []byte) []byte {
	return request[:commandLength]
}

//响应addr交互命令
func requestBlocks() {
	//遍历所有节点，发送getBLocks交互命令
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

//发送addr请求
func sendAddr(address string) {
	//追加新的节点地址
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	//设置command+payload请求信息
	payload := gobEncode(nodes)
	request := append(commandToBytes("addr"), payload...)
	//发送数据请求
	sendData(address, request)
}

//发送block请求
func sendBlock(addr string, b *Block) {
	//区块序列化
	data := block{nodeAddress, b.Serialize()}
	//设置command+payload
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)
	//发送请求
	sendData(addr, request)
}

// 发送请求到指定地址：version、tx、inv、getblocks、getdata
func sendData(addr string, data []byte) {
	//连接节点
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string
		//连接失败，更新可连接节点列表
		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		knownNodes = updatedNodes

		return
	}
	defer conn.Close()
	//连接成功，广播数据
	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

//交互命令：Inv
func sendInv(address, kind string, items [][]byte) {
	//设置payload信息：节点地址、类型、区块链所有区块hash
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	//设置command+payload请求信息
	request := append(commandToBytes("inv"), payload...)
	//发送数据请求
	sendData(address, request)
}

//交互命令：getblocks
func sendGetBlocks(address string) {
	//设置当前节点地址
	payload := gobEncode(getblocks{nodeAddress})
	//设置command+payload请求信息
	request := append(commandToBytes("getblocks"), payload...)
	//发送数据请求
	sendData(address, request)
}

//交互命令： getdata
func sendGetData(address, kind string, id []byte) {
	//设置payload信息：当前节点地址、类型（block|tx）、id（区块hash|交易hash）
	payload := gobEncode(getdata{nodeAddress, kind, id})
	//command+payload请求信息
	request := append(commandToBytes("getdata"), payload...)
	//发送数据请求
	sendData(address, request)
}

//交互命令：tx
func sendTx(addr string, tnx *Transaction) {
	//交易序列化
	data := tx{nodeAddress, tnx.Serialize()}
	//设置command+payload请求信息
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)
	//发送数据请求
	sendData(addr, request)
}

//交互命令： version
func sendVersion(addr string, bc *Blockchain) {
	//获取区块最新高度
	bestHeight := bc.GetBestHeight()
	//gob序列化
	payload := gobEncode(verzion{nodeVersion, bestHeight, nodeAddress})
	//请求命令：command+payload
	request := append(commandToBytes("version"), payload...)
	//发送请求
	sendData(addr, request)
}

//处理addr交互命令
func handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr
	//解析command+payload请求命令
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	//追加地址列表
	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	//向其他节点发送getblocks命令
	requestBlocks()
}

//处理block请求
func handleBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload block
	//解析command+payload请求信息
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	//反序列化区块信息
	blockData := payload.Block
	block := DeserializeBlock(blockData)
	//接收新的区块
	fmt.Println("Recevied a new block!")
	bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)
	//若存在缺失的区块，则发送getdata，获取指定的区块
	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := UTXOSet{bc} //重建utxo索引
		UTXOSet.Reindex()
	}
}

//处理Inv请求
func handleInv(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload inv
	//解析command+payload请求信息
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)
	//请求类型：block区块信息
	if payload.Type == "block" {
		blocksInTransit = payload.Items
		//向对端节点发送getdata请求，获取最新区块信息
		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, "block", blockHash)
		//更新本地区块信息
		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}
	//请求类型：tx交易信息
	if payload.Type == "tx" {
		txID := payload.Items[0]
		//判断本地交易池汇总是否存在请求的交易信息，不存在，则向对端节点发送getdata请求，获取最新交易
		if mempool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

//处理getblocks请求
func handleGetBlocks(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getblocks
	//解析command+payload
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	//查询当前节点区块链中所有区块的hash
	blocks := bc.GetBlockHashes()
	//发送Inv请求：来源地址、类型、所有区块hash
	sendInv(payload.AddrFrom, "block", blocks)
}

//处理getdata请求
func handleGetData(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload getdata
	//解析command+payload请求信息
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	//请求类型：block
	if payload.Type == "block" {
		//by 区块hash 查询区块信息
		block, err := bc.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}
		//发送block请求
		sendBlock(payload.AddrFrom, &block)
	}
	//请求类型：tx
	if payload.Type == "tx" {
		//解析payload信息，by交易hash，从交易池中获取交易信息
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]
		//发送tx请求
		sendTx(payload.AddrFrom, &tx)
		// delete(mempool, txID)
	}
}

//处理tx请求
func handleTx(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload tx
	//解析command+payload请求信息
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	//反序列化交易信息
	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx
	//判断当前节点是否是新加入的节点
	//1）向区块链中的其他节点发送Inv命令，获取交易信息
	if nodeAddress == knownNodes[0] {
		for _, node := range knownNodes {
			if node != nodeAddress && node != payload.AddFrom {
				sendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else { //2)当前节点不是新加入的节点，判断交易池大小和挖矿地址
		if len(mempool) >= 2 && len(miningAddress) > 0 {
		MineTransactions:
			var txs []*Transaction
			//遍历交易池，验证交易，获取当前节点所有交易信息
			for id := range mempool {
				tx := mempool[id]
				if bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}
			//所有交易非法，则就绪等待新交易的到来
			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}
			//创建coinbase交易
			cbTx := NewCoinbaseTX(miningAddress, "")
			txs = append(txs, cbTx)
			//挖矿，产生新的区块
			newBlock := bc.MineBlock(txs)
			//重建utxo索引
			UTXOSet := UTXOSet{bc}
			UTXOSet.Reindex()

			fmt.Println("New block is mined!")
			//清空交易池中的所有交易信息
			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}
			//向其他节点广播最新区块
			for _, node := range knownNodes {
				if node != nodeAddress {
					sendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}
			//交易池中存在交易，则不断进行挖矿
			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

//处理version请求
func handleVersion(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload verzion
	//解析comand+payload
	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}
	//当前节点最新高度
	myBestHeight := bc.GetBestHeight()
	//对端节点最新高度
	foreignerBestHeight := payload.BestHeight
	//1、当前节点最新高度 小于 对端节点高度，则发送getblocks请求，向对端节点获取区块
	//2、当前节点兑现高度 大于 对端节点高度，则发送version骑牛，想对端节点同步当前节点的高度信息
	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	}

	// 添加新节点
	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

// 处理请求连接
func handleConnection(conn net.Conn, bc *Blockchain) {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	conn.Close()
}

// 启动一个节点
func StartServer(nodeID, minerAddress string) {
	//nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	nodeAddress = fmt.Sprintf("localhost:%s", "3000")
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	bc := NewBlockchain(nodeID)
	//不是第一个节点，发送version交互命令
	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}
	//监听
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

// gob打包
func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

//节点信息
func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}
