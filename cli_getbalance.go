package blockchain7

import (
	"fmt"
	"log"
)

//GetBalance 获得账号余额
func (cli *CLI) getBalance(address string, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: 地址非法")
	}
	bc := NewBlockchain(nodeID)
	defer bc.Db.Close()

	balance := 0
	pubKeyHash := Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	UTXOSet := UTXOSet{bc}
	UTXOs := UTXOSet.FindUTXO(pubKeyHash)

	for _, output := range UTXOs {
		balance += output.Value
	}

	fmt.Printf("'%s'的账号余额是: %d\n", address, balance)
}
