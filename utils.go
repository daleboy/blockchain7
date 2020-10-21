package blockchain7

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"log"
	"strconv"
)

// Errors
var (
	ErrEmptyString   = &decError{"empty hex string"}
	ErrSyntax        = &decError{"invalid hex string"}
	ErrMissingPrefix = &decError{"hex string without 0x prefix"}
	ErrOddLength     = &decError{"hex string of odd length"}
	ErrEmptyNumber   = &decError{"hex string \"0x\""}
	ErrLeadingZero   = &decError{"hex number with leading zero digits"}
	ErrUint64Range   = &decError{"hex number > 64 bits"}
	ErrBig256Range   = &decError{"hex number > 256 bits"}
)

type decError struct{ msg string }

func (err decError) Error() string { return err.msg }

// IntToHex 将整型转为二进制数组
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// ReverseBytes 反转字节数组顺序
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

// Encode 编码字节数组为hex字符串（带0x前缀）
func Encode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// Decode 反编码一个hex字符串（包含0x前缀）
func Decode(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, ErrEmptyString
	}
	if !has0xPrefix(input) {
		return nil, ErrMissingPrefix
	}
	b, err := hex.DecodeString(input[2:])
	if err != nil {
		err = mapError(err)
	}
	return b, err
}

func has0xPrefix(input string) bool {
	return len(input) >= 2 && input[0] == '0' && (input[1] == 'x' || input[1] == 'X')
}

func mapError(err error) error {
	if err, ok := err.(*strconv.NumError); ok {
		switch err.Err {
		case strconv.ErrRange:
			return ErrUint64Range
		case strconv.ErrSyntax:
			return ErrSyntax
		}
	}
	if _, ok := err.(hex.InvalidByteError); ok {
		return ErrSyntax
	}
	if err == hex.ErrLength {
		return ErrOddLength
	}
	return err
}
