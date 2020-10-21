package blockchain7

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/boltdb/bolt"
	"github.com/ethereum/go-ethereum/common"
)

//dbFile 区块链数据库文件名称
const dbFile = "blockchain_%s.db" //每个节点都有自己的数据库名称
const blocksBucket = "blocks"     //存储的内容的键
const genesisCoinbaseData = "The Times 14/Oct/2020 拯救世界，从今天开始。"

//Blockchain 区块链结构
//我们不在里面存储所有的区块了，而是仅存储区块链的 tip。
//另外，我们存储了一个数据库连接。因为我们想要一旦打开它的话，就让它一直运行，直到程序运行结束。
type Blockchain struct {
	Tip []byte   //区块链最后一块的哈希值
	Db  *bolt.DB //数据库
}

//MineBlock 挖出普通区块并将新区块加入到区块链中
//此方法通过区块链的指针调用，将修改区块链bc的内容
func (bc *Blockchain) MineBlock(transactions []*Transaction) *Block {
	var lastHash []byte //区块链最后一个区块的哈希
	var lashtHeght int  //区块链最后一个区块的高度
	//在将交易放入块之前进行签名验证
	for _, tx := range transactions {
		if bc.VerifyTransaction(tx) != true {
			log.Panic("ERROR: 非法交易")
		}
	}

	err := bc.Db.View(func(tx *bolt.Tx) error { //只读打开，读取最后一个区块的哈希，作为新区块的prevHash
		b := tx.Bucket([]byte(blocksBucket))
		lastHash = b.Get([]byte("1")) //最后一个区块的哈希的键是字符串"1"

		blockData := b.Get(lastHash)
		block := DeserializeBlock(blockData)
		lashtHeght = block.Height
		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	newBlock := NewBlock(transactions, lastHash, lashtHeght) //挖出区块

	err = bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		err := b.Put(newBlock.Hash, newBlock.Serialize()) //将新区块序列化后插入到数据库表中
		if err != nil {
			log.Panic(err)
		}

		err = b.Put([]byte("1"), newBlock.Hash) //更新区块链最后一个区块的哈希到数据库中
		if err != nil {
			log.Panic(err)
		}

		bc.Tip = newBlock.Hash //修改区块链实例的tip值

		return nil
	})

	return newBlock
}

//CreatBlockchain 创建一个全新的区块链数据库
//address用户发起创始交易，并挖矿，奖励也发给用户address
//注意，创建后，数据库是open状态，需要使用者负责close数据库
func CreatBlockchain(address string, nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExist(dbFile) {
		fmt.Println("区块链已经存在")
		os.Exit(1)
	}

	var tip []byte                          //存储最后一块的哈希
	db, err := bolt.Open(dbFile, 0600, nil) //打开数据库，如果不存在，则创建一个新的
	if err != nil {
		log.Panic(err)
	}

	err = db.Update(func(tx *bolt.Tx) error { //更新数据库，通过事务进行操作。一个数据文件同时只支持一个读-写事务
		cbtx := NewCoinbaseTX(address, genesisCoinbaseData) //创建创始交易
		genesis := NewGenesisBlock(cbtx)                    //创建创始区块

		b, err := tx.CreateBucket([]byte(blocksBucket))
		if err != nil {
			log.Panic(err)
		}

		d := genesis.Serialize()
		err = b.Put(genesis.Hash, d) //将创始区块序列化后插入到数据库表中
		if err != nil {
			log.Panic(err)
		}

		//插入Tip到数据库，没有用到事务
		err = b.Put([]byte("1"), genesis.Hash)
		if err != nil {
			log.Panic(err)
		}
		tip = genesis.Hash

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	BC := Blockchain{tip, db} //构建区块链实例

	return &BC //返回区块链实例的指针
}

//Iterator 每当需要对链中的区块进行迭代时候，我们就通过Blockchain创建迭代器
//注意，迭代器初始状态为链中的tip，因此迭代是从最新到最旧的进行获取
func (bc *Blockchain) Iterator() *BlockchainIterator {
	bci := &BlockchainIterator{bc.Tip, bc.Db}
	return bci
}

//FindUnspentTransaction 查找未花费的交易（即该交易的花费尚未花出，换句话说，
//及该交易的输出尚未被其他交易作为输入包含进去）
func (bc *Blockchain) FindUnspentTransaction(pubKeyHash []byte) []Transaction {
	var unspentTXs []Transaction //未花费交易

	//已花费输出，key是转化为字符串的当前交易的ID
	//value是该交易包含的引用输出的所有已花费输出值数组
	//一个交易可能有多个输出，在这种情况下，该交易将引用所有的输出：输出不可分规则，无法引用它的一部分，要么不用，要么一次性用完
	//在go中，映射的值可以是数组，所以这里创建一个映射来存储未花费输出
	spentTXOs := make(map[string][]int)

	//从区块链中取得所有已花费输出
	bci := bc.Iterator()
	for { //第一层循环，对区块链中的所有区块进行迭代查询
		block := bci.Next()

		for _, tx := range block.Transactions { //第二层循环，对单个区块中的所有交易进行循环：一个区块可能打包了多个交易
			//检查交易的输入，将所有可以解锁的引用的输出加入到已花费输出map中
			if tx.IsCoinbase() == false { //不适用于创始区块的交易，因为它没有引用输出
				for _, in := range tx.Vin { //第三层循环，对单个交易中的所有输入进行循环（一个交易可能有多个输入）
					if in.UsesKey(pubKeyHash) { //可以被pubKeyHash解锁，即属于pubKeyHash发起的交易（sender）
						inTxID := hex.EncodeToString(in.Txid)
						//in.Vout为引用输出在该交易所有输出中的一个索引
						spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout) //加入到已花费映射之中
					}
				}
			}
		}
		if len(block.PrevBlockHash) == 0 { //创始区块都检查完了，退出最外层循环
			break
		}
	}

	//获得未花费交易
	bci = bc.Iterator()
	for { //第一层循环，对区块链中的所有区块进行迭代查询
		block := bci.Next()

		for _, tx := range block.Transactions { //第二层循环，对单个区块中的所有交易进行循环：一个区块可能打包了多个交易
			txID := hex.EncodeToString(tx.ID) //交易ID转为字符串，便于比较

		Outputs:
			for outIdx, out := range tx.Vout { //第三层循环，对单个交易中的所有输出进行循环（一个交易可能有多个输出）
				//检查交易的输出，OutIdx为数组序号，实际上也是某个TxOutput的索引，out为TxOutput
				//一个交易，可能会有多个输出
				//输出是否已经花费了？
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] { //第四层循环，对前面获得的所有未花费输出进行循环，spentOut是value
						//根据输出引用不可再分规则，
						//只要有一个输出值被引用，那么该输出的所有值都被引用了
						//所以通过比较索引值，只要发现一个输出值被引用了，就不必查询下一个输出值了
						//说明该输出已经被引用（被包含在其它交易的输入之中，即被花费掉了）
						if spentOut == outIdx {
							continue Outputs //在 continue 语句后添加标签Outputs时，表示开始标签Outputs对应的循环
						}
					}
				}

				//输出没有被花费，且由pubKeyHash锁定（即归pubKeyHash用户所有）
				if out.IsLockedWithKey(pubKeyHash) {
					unspentTXs = append(unspentTXs, *tx) //将tx值加入到已花费交易数组中
				}
			}

		}
		if len(block.PrevBlockHash) == 0 { //创始区块都检查完了，退出最外层循环
			break
		}
	}
	return unspentTXs
}

//FindUTXO 从区块链中取得所有未花费输出及包含未花费输出的block
//只会区块链新创建后调用一次，其他时候不会调用
//不再需要调用者的公钥，因为我们保存到bucket的UTXO是所有的未花费输出
func (bc *Blockchain) FindUTXO() (map[string]TxOutputs, map[string]common.Hash) {
	UTXO := make(map[string]TxOutputs)         //已花费输出
	spentTXOs := make(map[string][]int)        //未花费输出
	UTXOBlocks := make(map[string]common.Hash) //含未花费输出的区块
	bci := bc.Iterator()

	//获得已花费输出
	for {
		block := bci.Next() //迭代区块链中所有的区块

		for _, tx := range block.Transactions {
			if tx.IsCoinbase() == false { //coninbase交易没有实际输入
				for _, in := range tx.Vin {
					inTxID := hex.EncodeToString(in.Txid)
					spentTXOs[inTxID] = append(spentTXOs[inTxID], in.Vout)
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	//获得未花费输出
	bci = bc.Iterator()
	for {
		block := bci.Next() //迭代区块链中所有的区块

		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Vout {
				// 该输出是否已经花费？
				if spentTXOs[txID] != nil {
					for _, spentOutIdx := range spentTXOs[txID] {
						if spentOutIdx == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs

				UTXOBlocks[txID] = common.BytesToHash(block.Hash)
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO, UTXOBlocks
}

//dbExists 判断数据库文件是否存在
func dbExist(dbFile string) bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}

//NewBlockchain 从数据库中取出最后一个区块的哈希，构建一个区块链实例
func NewBlockchain(nodeID string) *Blockchain {
	dbFile := fmt.Sprintf(dbFile, nodeID)
	if dbExist(dbFile) == false {
		fmt.Println("区块链不存在，请首先创建一个新的.")
		os.Exit(1)
	}

	var tip []byte
	db, err := bolt.Open(dbFile, 0600, nil)
	if err != nil {
		log.Panic(err)
	}

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket)) //通过名称获得bucket
		tip = b.Get([]byte("1"))             //获得最后区块的哈希

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	bc := Blockchain{Tip: tip, Db: db}

	return &bc
}

//AddBlock 将区块加入到本地区块链中
func (bc *Blockchain) AddBlock(block *Block) {
	err := bc.Db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockInDb := b.Get(block.Hash)

		if blockInDb != nil {
			return nil
		}

		blockData := block.Serialize()
		err := b.Put(block.Hash, blockData)
		if err != nil {
			log.Panic(err)
		}

		lastHash := b.Get([]byte("l"))
		lastBlockData := b.Get(lastHash)
		lastBlock := DeserializeBlock(lastBlockData)

		if block.Height > lastBlock.Height {
			err = b.Put([]byte("l"), block.Hash)
			if err != nil {
				log.Panic(err)
			}
			bc.Tip = block.Hash
		}

		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

// GetBestHeight 返回最后一个区块的高度
func (bc *Blockchain) GetBestHeight() int {
	var lastBlock Block

	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		lastHash := b.Get([]byte("l"))
		blockData := b.Get(lastHash)
		lastBlock = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		log.Panic(err)
	}

	return lastBlock.Height
}

// GetBlock 通过哈希返回一个区块
func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		blockData := b.Get(blockHash)

		if blockData == nil {
			return errors.New("没有找到区块。")
		}

		block = *DeserializeBlock(blockData)

		return nil
	})
	if err != nil {
		return block, err
	}

	return block, nil
}

// GetBlockHashes 返回区块链中的所有区块哈希列表
func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	bci := bc.Iterator()

	for {
		block := bci.Next()

		blocks = append(blocks, block.Hash)

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

//FindSpendableOutput 查找某个用户可以花费的输出，放到一个映射里面
//从未花费交易里取出未花费的输出，直至取出输出的币总数大于或等于需要send的币数为止
func (bc *Blockchain) FindSpendableOutput(pubKeyHash []byte, amount int) (int, map[string][]int) {
	unpsentOutputs := make(map[string][]int)
	unspentTXs := bc.FindUnspentTransaction(pubKeyHash)
	accumulated := 0 //sender发出的转出的全部币数

Work:
	for _, tx := range unspentTXs {
		txID := hex.EncodeToString(tx.ID)

		for outIdx, out := range tx.Vout {
			if out.IsLockedWithKey(pubKeyHash) && accumulated < amount {
				accumulated += out.Value
				unpsentOutputs[txID] = append(unpsentOutputs[txID], outIdx)

				if accumulated >= amount {
					break Work
				}
			}
		}
	}

	return accumulated, unpsentOutputs
}

// FindTransactionForUTXO 根据交易ID查询到一个交易，仅仅查询UTXOBlock的数据库，不需要迭代整个区块链
func (bc *Blockchain) FindTransactionForUTXO(txID []byte) (Transaction, error) {
	var tnx Transaction
	err := bc.Db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		blockb := tx.Bucket([]byte(utxoBlockBucket)) //UTXOBlock

		blockhash := blockb.Get(txID) //UTXOBlock
		blockData := b.Get(blockhash)
		block := *DeserializeBlock(blockData)
		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, txID) == 0 {
				tnx = *tx
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	if &tnx != nil {
		return tnx, nil
	}

	return Transaction{}, errors.New("未找到交易")
}

// FindTransaction 迭代整个区块链，根据交易ID查询到一个交易
func (bc *Blockchain) FindTransaction(txID []byte) (Transaction, error) {
	bci := bc.Iterator()

	for {
		block := bci.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, txID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("未找到交易")
}

// SignTransaction 对一个交易的所有输入引用的输出的交易进行签名
//注意，这里签名的不是参数tx（当前交易），而是tx输入所引用的输出的交易
func (bc *Blockchain) SignTransaction(tx *Transaction, privKey ecdsa.PrivateKey) {
	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransactionForUTXO(vin.Txid) //通过交易输入引用的输出交易ID获得输出交易
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	tx.Sign(privKey, prevTXs)
}

// VerifyTransaction 验证一个交易的所有输入的签名
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	prevTXs := make(map[string]Transaction)

	for _, vin := range tx.Vin {
		prevTX, err := bc.FindTransactionForUTXO(vin.Txid)
		if err != nil {
			log.Panic(err)
		}
		prevTXs[hex.EncodeToString(prevTX.ID)] = prevTX
	}

	return tx.Verify(prevTXs)
}
