package blockchain7

import (
	"fmt"
	"log"
)

//createBlockchain 创建全新区块链
func (cli *CLI) createBlockchain(address string, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: 地址非法")
	}
	bc := CreatBlockchain(address, nodeID) //注意，这里调用的是blockchain.go中的函数
	//bc := NewBlockchain()
	defer bc.Db.Close()

	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex() //在数据库中建立UTXO

	fmt.Println("创建全新区块链完毕！")
}
