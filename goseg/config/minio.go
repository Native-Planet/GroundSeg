package config

import (
	"fmt"
	"goseg/logger"
)

func GetMinIOPassword(name string) (string, error) {
	minioPwdMutex.Lock()
	defer minioPwdMutex.Unlock()
	logger.Logger.Debug(fmt.Sprintf("get - minio passwords: %+v", name))
	password, exists := minIOPasswords[name]
	if !exists {
		return "", fmt.Errorf("%v password does not exist!", name)
	}
	return password, nil
}

func SetMinIOPassword(name, password string) error {
	minioPwdMutex.Lock()
	defer minioPwdMutex.Unlock()
	logger.Logger.Debug(fmt.Sprintf("set - minio passwords: %+v", name))
	minIOPasswords[name] = password
	return nil
}
