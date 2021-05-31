package blockchain7

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const walletFile = "wallet_%s.dat"

// Wallets 保存钱包集合
type Wallets struct {
	Wallets map[string]*Wallet
}

// NewWallets 从文件读取生成Wallets
func NewWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)

	err := wallets.LoadFromFile(nodeID)

	return &wallets, err
}

// CreateWallet 添加一个钱包到Wallets
func (ws *Wallets) CreateWallet() string {
	wallet := NewWallet()
	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = wallet

	return address
}

// GetAddresses 从钱包文件中返回所有钱包地址
func (ws *Wallets) GetAddresses() []string {
	var addresses []string

	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// GetWallet 根据地址返回一个钱包
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// LoadFromFile 从文件读取wallets
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets

	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets

	return nil
}

// SaveToFile 保存wallets到文件
func (ws Wallets) SaveToFile(nodeID string) {
	var content bytes.Buffer

	walletFile := fmt.Sprintf(walletFile, nodeID)

	//Wallet的PrivateKey的结构体类型逐层分析下去，有一个结构体字段是priv.PublicKey.Curve，
	//其类型是elliptic.Curve，而elliptic.Curve是一个interface，实际上在产生wallet时候，
	//传递的具体实现类型是curve := elliptic.P256()
	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}
