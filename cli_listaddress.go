package blockchain7

import (
	"fmt"
	"log"
)

//listAddresses 列出所有钱包的地址
func (cli *CLI) listAddresses(nodeID string) {
	wallets, err := NewWallets(nodeID)
	if err != nil {
		log.Panic(err)
	}
	addresses := wallets.GetAddresses()

	for _, address := range addresses {
		fmt.Println(address)
	}
}
