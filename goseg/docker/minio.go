package docker

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
)

func LoadMC() error {
	conf := config.Conf()
	if conf.WgRegistered {
		logger.Logger.Info("Loading MC container")
		confPath := filepath.Join(config.BasePath, "settings", "mc.json")
		_, err := os.Open(confPath)
		if err != nil {
			// create a default if it doesn't exist
			err = config.CreateDefaultMcConf()
			if err != nil {
				// error if we can't create it
				errmsg := fmt.Sprintf("Unable to create MC config! %v", err)
				logger.Logger.Error(errmsg)
			}
		}
		logger.Logger.Info("Running MC")
		info, err := StartContainer("mc", "miniomc")
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Error starting MC: %v", err))
			return err
		}
		config.UpdateContainerState("mc", info)
	}
	return nil
}

// iterate through each ship and create a minio
// version stuff is offloaded to version server struct
func LoadMinIOs() error {
	conf := config.Conf()
	if conf.WgRegistered {
		logger.Logger.Info("Loading MinIO containers")
		for _, pier := range conf.Piers {
			label := "minio_" + pier
			info, err := StartContainer(label, "minio")
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Error starting %s Minio: %v", pier, err))
				continue
			}
			config.UpdateContainerState(label, info)
		}
	}
	return nil
}

// minio container config builder
func minioContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	shipName := strings.Split(containerName, "_")[1]
	err := config.LoadUrbitConfig(shipName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", shipName, err)
		return containerConfig, hostConfig, errmsg
	}
	shipConf := config.UrbitConf(shipName)
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("minio")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// Create a byte slice of length 16
	randomBytes := make([]byte, 16)
	// Generate random bytes
	_, err = rand.Read(randomBytes)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	// Convert to a 32-character long hex string
	minIOPwd := hex.EncodeToString(randomBytes)
	if err := config.SetMinIOPassword(containerName, minIOPwd); err != nil {
		return containerConfig, hostConfig, err
	}

	environment := []string{
		fmt.Sprintf("MINIO_ROOT_USER=%s", shipName),
		fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", minIOPwd), //shipConf.MinioPassword),
		fmt.Sprintf("MINIO_DOMAIN=s3.%s", shipConf.WgURL),
		fmt.Sprintf("MINIO_SERVER_URL=https://s3.%s", shipConf.WgURL),
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: containerName,
			Target: "/data",
		},
	}
	containerConfig = container.Config{
		Image:      desiredImage,
		Entrypoint: []string{"minio"},
		Cmd: []string{
			"server",
			"/data",
			"--address",
			fmt.Sprintf(":%v", shipConf.WgS3Port),
			//fmt.Sprintf("\":%v\"", shipConf.WgS3Port),
			"--console-address",
			fmt.Sprintf(":%v", shipConf.WgConsolePort),
			//fmt.Sprintf("\":%v\"", shipConf.WgConsolePort),
		},
		Env: environment,
	}
	// always on wg nw
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
		Mounts:      mounts,
	}
	ticker := time.NewTicker(500 * time.Millisecond)
minioNetworkLoop:
	for {
		select {
		case <-ticker.C:
			status, err := GetContainerRunningStatus("wireguard")
			if err != nil {
				return containerConfig, hostConfig, err
			}
			if strings.Contains(status, "Up") {
				break minioNetworkLoop
			}
		}
	}
	return containerConfig, hostConfig, nil
}

// miniomc container config builder
func mcContainerConf() (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := GetLatestContainerInfo("miniomc")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// construct the container config struct
	containerConfig = container.Config{
		Image:      desiredImage,
		Entrypoint: []string{"/bin/bash"},
		Tty:        true,
		OpenStdin:  true,
	}
	// always on wg nw
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
	}
	ticker := time.NewTicker(500 * time.Millisecond)
mcNetworkLoop:
	for {
		select {
		case <-ticker.C:
			status, err := GetContainerRunningStatus("wireguard")
			if err != nil {
				return containerConfig, hostConfig, err
			}
			if strings.Contains(status, "Up") {
				break mcNetworkLoop
			}
		}
	}
	return containerConfig, hostConfig, nil
}

func setMinIOAdminAccount(containerName string) error {
	// get patp
	patp, err := getPatpFromMinIOName(containerName)
	if err != nil {
		return err
	}
	// get urbit config
	urbConf := config.UrbitConf(patp)
	// make sure mc is running
	ticker := time.NewTicker(500 * time.Millisecond)
mcRunning:
	for {
		select {
		case <-ticker.C:
			status, _ := GetContainerRunningStatus("mc")
			if strings.Contains(status, "Up") {
				break mcRunning
			}
		}
	}
	// get password
	pwd, err := config.GetMinIOPassword(fmt.Sprintf("minio_%s", patp))
	if err != nil {
		return err
	}
	// set alias
	aliasCommand := []string{
		"mc",
		"alias",
		"set",
		fmt.Sprintf("patp_%s", patp),
		fmt.Sprintf("http://localhost:%v", urbConf.WgS3Port),
		patp,
		pwd,
	}
	if _, err := ExecDockerCommand(containerName, aliasCommand); err != nil {
		return err
	}
	// make bucket
	createCommand := []string{
		"mc",
		"mb",
		"--ignore-existing",
		fmt.Sprintf("patp_%s/bucket", patp),
	}
	if _, err := ExecDockerCommand(containerName, createCommand); err != nil {
		return err
	}
	publicCommand := []string{
		"mc",
		"anonymous",
		"set",
		"download",
		fmt.Sprintf("patp_%s/bucket", patp),
	}
	if _, err := ExecDockerCommand(containerName, publicCommand); err != nil {
		return err
	}
	return nil
}

func getPatpFromMinIOName(containerName string) (string, error) {
	// Check if string starts with "minio_"
	if !strings.HasPrefix(containerName, "minio_") {
		return "", fmt.Errorf("Invalid MinIO container name")
	}
	// Split the string
	splitStr := strings.SplitN(containerName, "_", 2)
	return splitStr[1], nil
}

func CreateMinIOServiceAccount(patp string) (structs.MinIOServiceAccount, error) {
	var svcAccount structs.MinIOServiceAccount
	svcAccount.AccessKey = "urbit_minio"
	svcAccount.Alias = fmt.Sprintf("patp_%s", patp)
	svcAccount.User = patp
	// create secret key
	bytes := make([]byte, 20) // 20 bytes * 2 (hex encoding) = 40 characters
	_, err := rand.Read(bytes)
	if err != nil {
		return svcAccount, err
	}
	svcAccount.SecretKey = hex.EncodeToString(bytes)
	// send command
	containerName := fmt.Sprintf("minio_%s", patp)
	cmd := []string{
		"mc",
		"admin",
		"user",
		"svcacct",
		"rm",
		svcAccount.Alias,
		fmt.Sprintf("%s", svcAccount.AccessKey),
	}
	response, err := ExecDockerCommand(containerName, cmd)
	if err != nil {
		return svcAccount, fmt.Errorf("Failed to remove old service account for %s: %v", patp, err)
	}
	logger.Logger.Debug(fmt.Sprintf("Remove old service account response: %s", response))
	// couldn't edit, add new instead
	if strings.Contains(response, "successfully") {
		cmd = []string{
			"mc",
			"admin",
			"user",
			"svcacct",
			"add",
			"--access-key",
			fmt.Sprintf("%s", svcAccount.AccessKey),
			"--secret-key",
			fmt.Sprintf("%s", svcAccount.SecretKey),
			svcAccount.Alias,
			svcAccount.User,
		}
		response, err := ExecDockerCommand(containerName, cmd)
		if err != nil {
			return svcAccount, fmt.Errorf("Failed to create service account for %s: %v", patp, err)
		}
		if strings.Contains(response, "ERROR") {
			return svcAccount, fmt.Errorf("Failed to create service account for %s: %v", patp, response)
		}
	} else {
		return svcAccount, fmt.Errorf("Failed to remove old service account for %s: %v", patp, response)
	}
	return svcAccount, nil
}
