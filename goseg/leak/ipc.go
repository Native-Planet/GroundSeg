package leak

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"goseg/logger"
	"math/big"
	"net"

	"github.com/stevelacy/go-urbit/noun"
)

func connectToIPC(socketPath string) (net.Conn, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func toBytes(num *big.Int) []byte {
	var padded []byte
	// version: 0
	version := []byte{0}

	// length: 4 bytes
	length := noun.ByteLen(num)
	lenBytes, err := int64ToLittleEndianBytes(length)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("%v", err))
	}
	bytes := makeBytes(num)
	padded = append(padded, version...)
	padded = append(padded, lenBytes...)
	padded = append(padded, bytes...)

	return padded
}

func makeBytes(num *big.Int) []byte {
	byteSlice := num.Bytes()
	// Reverse the slice for little-endian
	for i, j := 0, len(byteSlice)-1; i < j; i, j = i+1, j-1 {
		byteSlice[i], byteSlice[j] = byteSlice[j], byteSlice[i]
	}
	return byteSlice
}

func int64ToLittleEndianBytes(num int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	// uint32 for 4 bytes
	err := binary.Write(buf, binary.LittleEndian, uint32(num))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
