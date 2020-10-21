package blockchain7

import (
	"fmt"
	"math/big"
	"reflect"
)

const (
	//HashLength 哈希长度
	HashLength = 32
)

var (
	hashT = reflect.TypeOf(Hash{})
)

// Hash 表示32字节Keccak256哈希
type Hash [HashLength]byte

// BytesToHash 将b转为hash
// 如果b的长度大于32字节, b将从左侧截断
func BytesToHash(b []byte) Hash {
	var h Hash
	h.SetBytes(b)
	return h
}

// Bytes 获得字节数组切
func (h Hash) Bytes() []byte { return h[:] }

// Big 将哈希转为 big integer
func (h Hash) Big() *big.Int { return new(big.Int).SetBytes(h[:]) }

// Hex 将哈希转为hex字符串
func (h Hash) Hex() string { return Encode(h[:]) }

// String  实现stringer接口，也会在需要写入日志时候用到
func (h Hash) String() string {
	return h.Hex()
}

// Format 实现fmt.Formatter，强制将字节切片转为指定的格式
func (h Hash) Format(s fmt.State, c rune) {
	fmt.Fprintf(s, "%"+string(c), h[:])
}

// SetBytes 将哈希转为二进制数组，并拷贝给二进制数组b.
// 如果b比len(h)大, b将被从左侧截断.
func (h *Hash) SetBytes(b []byte) {
	if len(b) > len(h) {
		b = b[len(b)-HashLength:]
	}

	copy(h[HashLength-len(b):], b)
}
