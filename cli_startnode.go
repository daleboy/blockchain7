package blockchain7

import (
	"fmt"
	"log"
)

func (cli *CLI) startNode(nodeID, minerAddress string) {
	fmt.Printf("开始节点 %s\n", nodeID)
	if len(minerAddress) > 0 {
		if ValidateAddress(minerAddress) {
			fmt.Println("挖矿正在进行中. 接收挖矿奖励的地址: ", minerAddress)
		} else {
			log.Panic("错误的挖矿地址!")
		}
	}
	StartServer(nodeID, minerAddress) //启动节点服务器：区块链中每一个节点都是服务器
}
