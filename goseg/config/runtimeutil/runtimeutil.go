package runtimeutil

import (
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// NetCheck checks outbound tcp connectivity for an ip:port endpoint.
func NetCheck(netCheck string) bool {
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp", netCheck, timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// RandStringWithError generates a random secret string of the input length.
func RandStringWithError(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	randBytes := make([]byte, length)
	_, err := cryptorand.Read(randBytes)
	if err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(randBytes), nil
}

// RandString is a best-effort wrapper around RandStringWithError.
func RandString(length int) string {
	value, err := RandStringWithError(length)
	if err != nil {
		return ""
	}
	return value
}

// SHA256 returns the hex-encoded SHA256 for a file path.
func SHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
