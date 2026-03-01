package docker

import (
	"encoding/hex"
	"fmt"
	"groundseg/structs"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.uber.org/zap"
)

type minioRuntime = dockerRuntime

func newMinIORuntime() minioRuntime {
	return newDockerRuntime()
}

func LoadMC() error {
	return loadMC(newMinIORuntime())
}

func loadMC(rt minioRuntime) error {
	conf := rt.conf()
	if conf.WgRegistered {
		zap.L().Info("Loading MC container")
		confPath := filepath.Join(rt.basePath(), "settings", "mc.json")
		_, err := rt.osOpen(confPath)
		if err != nil {
			// create a default if it doesn't exist
			err = rt.createDefaultMcConf()
			if err != nil {
				// error if we can't create it
				errmsg := fmt.Sprintf("Unable to create MC config! %v", err)
				zap.L().Error(errmsg)
			}
		}
		zap.L().Info("Running MC")
		info, err := rt.startContainer("mc", "miniomc")
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error starting MC: %v", err))
			return err
		}
		rt.updateContainerState("mc", info)
	}
	return nil
}

// iterate through each ship and create a minio
// version stuff is offloaded to version server struct
func LoadMinIOs() error {
	return loadMinIOs(newMinIORuntime())
}

func loadMinIOs(rt minioRuntime) error {
	conf := rt.conf()
	if conf.WgRegistered {
		zap.L().Info("Loading MinIO containers")
		for _, pier := range conf.Piers {
			label := "minio_" + pier
			info, err := rt.startContainer(label, "minio")
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error starting %s Minio: %v", pier, err))
				continue
			}
			rt.updateContainerState(label, info)
		}
	}
	return nil
}

// minio container config builder
func minioContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	return minioContainerConfWithRuntime(newMinIORuntime(), containerName)
}

func minioContainerConfWithRuntime(rt minioRuntime, containerName string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	shipName, err := getPatpFromMinIOName(containerName)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	err = rt.loadUrbitConfig(shipName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", shipName, err)
		return containerConfig, hostConfig, errmsg
	}
	shipConf := rt.urbitConf(shipName)
	// construct the container metadata from version server info
	containerInfo, err := rt.getLatestContainerInfo("minio")
	if err != nil {
		return containerConfig, hostConfig, err
	}
	desiredImage := fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"])
	// Create a byte slice of length 16
	randomBytes := make([]byte, 16)
	// Generate random bytes
	_, err = rt.randRead(randomBytes)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	// Convert to a 32-character long hex string
	minIOPwd := hex.EncodeToString(randomBytes)
	if err := rt.setMinIOPassword(containerName, minIOPwd); err != nil {
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
	for {
		status, err := rt.getContainerRunningStatus("wireguard")
		if err != nil {
			return containerConfig, hostConfig, err
		}
		if strings.Contains(status, "Up") {
			break
		}
		rt.sleep(rt.pollInterval)
	}
	return containerConfig, hostConfig, nil
}

// miniomc container config builder
func mcContainerConf() (container.Config, container.HostConfig, error) {
	return mcContainerConfWithRuntime(newMinIORuntime())
}

func mcContainerConfWithRuntime(rt minioRuntime) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	// construct the container metadata from version server info
	containerInfo, err := rt.getLatestContainerInfo("miniomc")
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
	for {
		status, err := rt.getContainerRunningStatus("wireguard")
		if err != nil {
			return containerConfig, hostConfig, err
		}
		if strings.Contains(status, "Up") {
			break
		}
		rt.sleep(rt.pollInterval)
	}
	return containerConfig, hostConfig, nil
}

func setMinIOAdminAccount(containerName string) error {
	return setMinIOAdminAccountWithRuntime(newMinIORuntime(), containerName)
}

func setMinIOAdminAccountWithRuntime(rt minioRuntime, containerName string) error {
	// get patp
	patp, err := getPatpFromMinIOName(containerName)
	if err != nil {
		return err
	}
	// get urbit config
	urbConf := rt.urbitConf(patp)
	// make sure mc is running
	for {
		status, _ := rt.getContainerRunningStatus("mc")
		if strings.Contains(status, "Up") {
			break
		}
		rt.sleep(rt.pollInterval)
	}
	// get password
	pwd, err := rt.getMinIOPassword(fmt.Sprintf("minio_%s", patp))
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
	if _, err := rt.execDockerCommand(containerName, aliasCommand); err != nil {
		return err
	}
	// make bucket
	createCommand := []string{
		"mc",
		"mb",
		"--ignore-existing",
		fmt.Sprintf("patp_%s/bucket", patp),
	}
	if _, err := rt.execDockerCommand(containerName, createCommand); err != nil {
		return err
	}

	// write the script
	scriptContent := fmt.Sprintf(
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`,
		"bucket",
	)
	scriptPath := filepath.Join(rt.dockerDir(), containerName, "_data", "policy.json")
	err = rt.writeFile(scriptPath, []byte(scriptContent), 0755) // make the script executable
	if err != nil {
		return err
	}
	policyCommand := []string{
		"mc",
		"anonymous",
		"set-json",
		"/data/policy.json",
		fmt.Sprintf("patp_%s/bucket", patp),
	}
	if _, err := rt.execDockerCommand(containerName, policyCommand); err != nil {
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
	return createMinIOServiceAccountWithRuntime(newMinIORuntime(), patp)
}

func createMinIOServiceAccountWithRuntime(rt minioRuntime, patp string) (structs.MinIOServiceAccount, error) {
	var svcAccount structs.MinIOServiceAccount
	svcAccount.AccessKey = "urbit_minio"
	svcAccount.Alias = fmt.Sprintf("patp_%s", patp)
	svcAccount.User = patp
	// create secret key
	bytes := make([]byte, 20) // 20 bytes * 2 (hex encoding) = 40 characters
	_, err := rt.randRead(bytes)
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
	response, err := rt.execDockerCommand(containerName, cmd)
	if err != nil {
		return svcAccount, fmt.Errorf("Failed to remove old service account for %s: %v", patp, err)
	}
	zap.L().Debug(fmt.Sprintf("Remove old service account response: %s", response))
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
		response, err := rt.execDockerCommand(containerName, cmd)
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
