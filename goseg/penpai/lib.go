package penpai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"io"
	"os"
	"regexp"
)

var Locked = make(map[string]struct{})

func ExtractLlamafileName(url string) (string, error) {
	re := regexp.MustCompile(`([^/]+\.llamafile)`)
	match := re.FindStringSubmatch(url)
	if len(match) > 1 {
		return match[1], nil
	}
	return "", fmt.Errorf("invalid download link: %v", url)
}

func GetSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func SetExistence(m structs.Penpai, e bool, val int) error {
	conf := config.Conf()
	m.Exists = e
	conf.PenpaiModels[val] = m
	if err := config.UpdateConf(map[string]interface{}{
		"penpaiModels": conf.PenpaiModels,
	}); err != nil {
		return err
	}
	return nil
}

func Lock(hash string) {
	Locked[hash] = struct{}{}
}

func Unlock(hash string) {
	delete(Locked, hash)
}
