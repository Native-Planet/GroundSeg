package runtime

import (
	"fmt"
	"groundseg/click/internal/response"
	"groundseg/config"
	"groundseg/docker/lifecycle"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var (
	execDockerCommandFn = func(container string, cmd []string) (string, error) {
		response, exitCode, err := lifecycle.DefaultRuntime().ExecDockerCommand(container, cmd)
		if err != nil {
			return "", err
		}
		if exitCode != 0 {
			return "", fmt.Errorf("click command exited with code %d", exitCode)
		}
		return response, nil
	}
)

func JoinGap(hoon []string) string {
	return strings.Join(hoon, "  ")
}

func CreateHoon(patp, file, hoon string) error {
	shipConf := config.UrbitConf(patp)
	location := filepath.Join(config.DockerDir(), patp, "_data")
	if shipConf.CustomPierLocation != "" {
		location = shipConf.CustomPierLocation
	}
	hoonFile := filepath.Join(location, fmt.Sprintf("%s.hoon", file))
	return ioutil.WriteFile(hoonFile, []byte(hoon), 0644)
}

func DeleteHoon(patp, file string) {
	shipConf := config.UrbitConf(patp)
	location := filepath.Join(config.DockerDir(), patp, "_data")
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
	response, _, success, err := ExecuteCommandWithResponse(patp, file, hoon, sourcePath, successToken, operation, nil)
	if err != nil {
		return "", err
	}
	if successToken != "" && !success {
		return "", fmt.Errorf("%s failed poke", operation)
	}
	return response, nil
}

func ExecuteCommandWithResponse(
	patp, file, hoon, sourcePath, responseToken, operation string,
	clearLusCode func(string),
) (string, string, bool, error) {
	response, err := executeClickCommand(patp, file, hoon, sourcePath, operation, clearLusCode)
	if err != nil {
		return "", "", false, err
	}
	if responseToken == "" {
		return response, "", true, nil
	}
	parsed, success, err := FilterResponse(responseToken, response)
	if err != nil {
		return response, "", false, fmt.Errorf("%s failed to parse response: %v", operation, err)
	}
	return response, parsed, success, nil
}

func ExecuteCommandWithLusInvalidation(
	patp, file, hoon, sourcePath, successToken, operation string,
	clearLusCode func(string),
) (string, error) {
	response, _, success, err := ExecuteCommandWithResponse(patp, file, hoon, sourcePath, successToken, operation, clearLusCode)
	if err != nil {
		return "", err
	}
	if !success {
		return "", fmt.Errorf("%s failed poke", operation)
	}
	return response, nil
}

func ExecuteCommandWithSuccess(
	patp, file, hoon, sourcePath, successToken, operation string,
	clearLusCode func(string),
) (string, error) {
	response, _, success, err := ExecuteCommandWithResponse(patp, file, hoon, sourcePath, successToken, operation, clearLusCode)
	if err != nil {
		return "", err
	}
	if successToken != "" && !success {
		return "", fmt.Errorf("%s failed poke", operation)
	}
	return response, nil
}

func executeClickCommand(
	patp, file, hoon, sourcePath, operation string,
	clearLusCode func(string),
) (string, error) {
	if err := CreateHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("%s failed to create hoon: %v", operation, err)
	}
	if clearLusCode != nil {
		clearLusCode(patp)
	}
	defer DeleteHoon(patp, file)

	response, err := ClickExec(patp, file, sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s failed to execute hoon: %v", operation, err)
	}
	return response, nil
}
