package blockchain7

import "fmt"

func (cli *CLI) reindexUTXO(nodeID string) {
	bc := NewBlockchain(nodeID)
	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("重建索引完成! 总共有%d个交易在UTXO集合中。\n", count)
}
