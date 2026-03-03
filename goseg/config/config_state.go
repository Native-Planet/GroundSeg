package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"

	"go.uber.org/zap"
	"groundseg/structs"
)

// we keep map[string]structs.ContainerState in memory to keep track of the containers
// eg if they're running and whether they should be

// modify the desired/actual state of containers
func UpdateContainerState(name string, containerState structs.ContainerState) {
	contMutex.Lock()
	GSContainers[name] = containerState
	logMsg := "<hidden>"
	res, _ := json.Marshal(containerState)
	contMutex.Unlock()
	zap.L().Info(fmt.Sprintf("%s state:%s", name, logMsg))
	zap.L().Debug(fmt.Sprintf("%s state:%s", name, string(res)))
}

// delete a container from the config map
func DeleteContainerState(name string) {
	contMutex.Lock()
	delete(GSContainers, name)
	contMutex.Unlock()
	zap.L().Debug(fmt.Sprintf("%s removed from container state map", name))
}

// get the current container state
func GetContainerState() map[string]structs.ContainerState {
	contMutex.RLock()
	defer contMutex.RUnlock()
	stateCopy := make(map[string]structs.ContainerState, len(GSContainers))
	for name, state := range GSContainers {
		stateCopy[name] = state
	}
	return stateCopy
}

// check outbound tcp connectivity
// takes ip:port
func NetCheck(netCheck string) bool {
	internet := false
	timeout := 3 * time.Second
	conn, err := net.DialTimeout("tcp", netCheck, timeout)
	if err != nil {
		errmsg := fmt.Sprintf("Check internet access error: %v", err)
		zap.L().Error(errmsg)
	} else {
		internet = true
		if err := conn.Close(); err != nil {
			errmsg := fmt.Sprintf("Check internet access: unable to close connection probe to %s: %v", netCheck, err)
			zap.L().Warn(errmsg)
		}
	}
	return internet
}

// generates a random secret string of the input length
func RandString(length int) string {
	randBytes := make([]byte, length)
	_, err := rand.Read(randBytes)
	if err != nil {
		zap.L().Warn("Random error :s")
		return ""
	}
	return base64.URLEncoding.EncodeToString(randBytes)
}

func GetSHA256(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create a new SHA256 hash
	hasher := sha256.New()

	// Copy the file content to the hasher
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	// Get the final hash value
	hashValue := hasher.Sum(nil)

	// Convert the hash to a hexadecimal string
	hashString := hex.EncodeToString(hashValue)

	return hashString, nil
}
