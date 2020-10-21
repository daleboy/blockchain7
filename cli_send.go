package blockchain7

import (
	"fmt"
	"log"
)

//send 转账
func (cli *CLI) send(from, to string, amount int, nodeID string, mineNow bool) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: 发送地址非法")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: 接收地址非法")
	}

	bc := NewBlockchain(nodeID) //打开数据库，读取区块链并构建区块链实例
	UTXOSet := UTXOSet{bc}
	defer bc.Db.Close() //转账完毕，关闭数据库

	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	wallet := wallets.GetWallet(from)

	tx := NewUTXOTransaction(&wallet, to, amount, &UTXOSet)

	if mineNow { //当前是挖矿节点，有奖励
		cbTx := NewCoinbaseTX(from, "")
		txs := []*Transaction{cbTx, tx}

		newBlock := bc.MineBlock(txs)
		UTXOSet.Update(newBlock)
	} else { //非挖矿节点
		sendTx(knownNodes[0], tx)
	}

	fmt.Println("转账成功！")
}
