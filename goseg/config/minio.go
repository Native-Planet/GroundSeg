package config

import (
	"fmt"
)

func GetMinIOPassword(name string) (string, error) {
	minioPwdMutex.Lock()
	defer minioPwdMutex.Unlock()
	password, exists := minIOPasswords[name]
	if !exists {
		return "", fmt.Errorf("%v password does not exist!", name)
	}
	return password, nil
}

func SetMinIOPassword(name, password string) error {
	minioPwdMutex.Lock()
	defer minioPwdMutex.Unlock()
	minIOPasswords[name] = password
	return nil
}
