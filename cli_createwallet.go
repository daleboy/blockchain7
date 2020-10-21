package blockchain7

import "fmt"

func (cli *CLI) createWallet(nodeID string) {
	wallets, _ := NewWallets(nodeID)  //从钱包文件读取所有的钱包
	address := wallets.CreateWallet() //创建新钱包
	wallets.SaveToFile(nodeID)        //创建完成后，保存到本地，不参与网络共享，必须自己保管好！

	fmt.Printf("你的新钱包地址是: %s\n", address)
}
