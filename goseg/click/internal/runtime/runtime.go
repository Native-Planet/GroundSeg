package runtime

import (
	"fmt"
	"groundseg/click/internal/response"
	"groundseg/config"
	"groundseg/docker"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	execDockerCommandFn = func(container string, cmd []string) (string, error) {
		response, _, err := docker.ExecDockerCommand(container, cmd)
		return response, err
	}
)

func JoinGap(hoon []string) string {
	return strings.Join(hoon, "  ")
}

func CreateHoon(patp, file, hoon string) error {
	shipConf := config.UrbitConf(patp)
	location := filepath.Join(config.DockerDir, patp, "_data")
	if shipConf.CustomPierLocation != "" {
		location = shipConf.CustomPierLocation
	}
	hoonFile := filepath.Join(location, fmt.Sprintf("%s.hoon", file))
	return ioutil.WriteFile(hoonFile, []byte(hoon), 0644)
}

func DeleteHoon(patp, file string) {
	shipConf := config.UrbitConf(patp)
	location := filepath.Join(config.DockerDir, patp, "_data")
	if shipConf.CustomPierLocation != "" {
		location = shipConf.CustomPierLocation
	}
	hoonFile := filepath.Join(location, fmt.Sprintf("%s.hoon", file))
	if _, err := os.Stat(hoonFile); !os.IsNotExist(err) {
		os.Remove(hoonFile)
	}
}

func ClickExec(patp, file, dependency string) (string, error) {
	execCommand := []string{
		"click",
		"-b",
		"urbit",
		"-kp",
		"-i",
		fmt.Sprintf("%s.hoon", file),
		patp,
		dependency,
	}
	res, err := execDockerCommandFn(patp, execCommand)
	if err != nil {
		return "", err
	}
	return res, nil
}

func FilterResponse(resType string, pokeResponse string) (string, bool, error) {
	return response.ParsePokeResponse(resType, pokeResponse)
}

func ExecuteCommand(patp, file, hoon, sourcePath, successToken, operation string) (string, error) {
	if err := CreateHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("%s failed to create hoon: %v", operation, err)
	}
	defer DeleteHoon(patp, file)

	response, err := ClickExec(patp, file, sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s failed to execute hoon: %v", operation, err)
	}
	if successToken != "" {
		_, success, err := FilterResponse(successToken, response)
		if err != nil {
			return "", fmt.Errorf("%s failed to parse response: %v", operation, err)
		}
		if !success {
			return "", fmt.Errorf("%s failed poke", operation)
		}
	}
	return response, nil
}
