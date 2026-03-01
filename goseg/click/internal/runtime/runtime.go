package runtime

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	execDockerCommandFn = docker.ExecDockerCommand
)

func JoinGap(hoon []string) string {
	return strings.Join(hoon, "  ")
}

func CreateHoon(patp, file, hoon string) error {
	shipConf := config.UrbitConf(patp)
	location := filepath.Join(config.DockerDir, patp, "_data")
	if shipConf.CustomPierLocation != nil {
		if str, ok := shipConf.CustomPierLocation.(string); ok {
			location = str
		}
	}
	hoonFile := filepath.Join(location, fmt.Sprintf("%s.hoon", file))
	return ioutil.WriteFile(hoonFile, []byte(hoon), 0644)
}

func DeleteHoon(patp, file string) {
	shipConf := config.UrbitConf(patp)
	location := filepath.Join(config.DockerDir, patp, "_data")
	if shipConf.CustomPierLocation != nil {
		if str, ok := shipConf.CustomPierLocation.(string); ok {
			location = str
		}
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

func FilterResponse(resType string, response string) (string, bool, error) {
	responseSlice := strings.Split(response, "\n")
	switch resType {
	case "success":
		for _, line := range responseSlice {
			if strings.Contains(line, "[0 %avow 0 %noun %success]") {
				return "", true, nil
			}
		}
		return "", false, nil
	case "code":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				endIndex := strings.Index(line, "]")
				lastPercentIndex := strings.LastIndex(line[:endIndex], "%")
				if lastPercentIndex != -1 && endIndex != -1 && lastPercentIndex < endIndex {
					code := line[lastPercentIndex+1 : endIndex]
					return code, false, nil
				}
			}
		}
	case "desk":
		for _, line := range responseSlice {
			if strings.Contains(line, "%avow") {
				if strings.Contains(line, "does not yet exist") {
					return "not-found", false, nil
				}
				regex := regexp.MustCompile(`app status:\s+([^\s]+)`)
				match := regex.FindStringSubmatch(line)
				if len(match) >= 2 {
					appStatus := strings.TrimSuffix(match[1], "]")
					return appStatus, false, nil
				}
				return "not-found", false, nil
			}
		}
	case "default":
		return "", false, fmt.Errorf("Unknown poke response")
	}
	return "", false, fmt.Errorf("+code not in poke response")
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
