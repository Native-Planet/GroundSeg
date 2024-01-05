package click

import (
	"fmt"
	"groundseg/config"
	"groundseg/docker"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func createHoon(patp, file, hoon string) error {
	dockerDir := config.DockerDir
	hoonFile := filepath.Join(dockerDir, patp, "_data", fmt.Sprintf("%s.hoon", file))
	if err := ioutil.WriteFile(hoonFile, []byte(hoon), 0644); err != nil {
		return err
	}
	ClearLusCode(patp)
	return nil
}

func deleteHoon(patp, file string) {
	dockerDir := config.DockerDir
	hoonFile := filepath.Join(dockerDir, patp, "_data", fmt.Sprintf("%s.hoon", file))
	if _, err := os.Stat(hoonFile); !os.IsNotExist(err) {
		os.Remove(hoonFile)
	}
}

func clickExec(patp, file, dependency string) (string, error) {
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
	res, err := docker.ExecDockerCommand(patp, execCommand)
	if err != nil {
		return "", err
	}
	return res, nil
}

func joinGap(hoon []string) string {
	return strings.Join(hoon, "  ") // gap
}

func storageAction(key, value string) string {
	hoon := joinGap([]string{
		";<",
		"~",
		"bind:m",
		fmt.Sprintf("(poke [our %%storage] %%storage-action !>([%s '%s']))", key, value),
	})
	return hoon
}
