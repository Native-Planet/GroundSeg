package handler

import (
	"context"
	"fmt"
	"goseg/logger"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

func restructureDirectory(patp string) error {
	// get docker volume path for patp
	volDir := ""
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	volumes, err := cli.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		return err
	}
	for _, vol := range volumes.Volumes {
		if vol.Name == patp {
			volDir = vol.Mountpoint
			break
		}
	}
	if volDir == "" {
		return fmt.Errorf("No docker volume or %d!", patp)
	}
	dataDir := filepath.Join(volDir, "_data")
	// find .urb
	var urbLoc []string
	_ = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && filepath.Base(path) == ".urb" && !strings.Contains(path, "__MACOSX") {
			urbLoc = append(urbLoc, filepath.Dir(path))
		}
		return nil
	})
	// there can only be one
	if len(urbLoc) > 1 {
		return fmt.Errorf("%v ships detected in pier directory", len(urbLoc))
	}
	if len(urbLoc) < 1 {
		return fmt.Errorf("No ship found in pier directory")
	}
	logger.Logger.Debug(fmt.Sprintf(".urb subdirectory in %v", urbLoc[0]))
	pierDir := filepath.Join(dataDir, patp)
	tempDir := filepath.Join(dataDir, "temp_dir")
	unusedDir := filepath.Join(dataDir, "unused")
	// move it into the right place
	if filepath.Join(pierDir, ".urb") != filepath.Join(urbLoc[0], ".urb") {
		logger.Logger.Info(".urb location incorrect! Restructuring directory structure")
		logger.Logger.Debug(fmt.Sprintf(".urb found in %v", urbLoc[0]))
		logger.Logger.Debug(fmt.Sprintf("Moving to %v", tempDir))
		if dataDir == urbLoc[0] { // .urb in root
			_ = os.MkdirAll(tempDir, 0755)
			items, _ := ioutil.ReadDir(urbLoc[0])
			for _, item := range items {
				if item.Name() != patp {
					os.Rename(filepath.Join(urbLoc[0], item.Name()), filepath.Join(tempDir, item.Name()))
				}
			}
		} else {
			os.Rename(urbLoc[0], tempDir)
		}
		unused := []string{}
		dirs, _ := ioutil.ReadDir(dataDir)
		for _, dir := range dirs {
			dirName := dir.Name()
			if dirName != "temp_dir" && dirName != "unused" {
				unused = append(unused, dirName)
			}
		}
		if len(unused) > 0 {
			_ = os.MkdirAll(unusedDir, 0755)
			for _, u := range unused {
				os.Rename(filepath.Join(dataDir, u), filepath.Join(unusedDir, u))
			}
		}
		os.Rename(tempDir, pierDir)
		logger.Logger.Debug(fmt.Sprintf("%v restructuring done!", patp))
	} else {
		logger.Logger.Debug("No restructuring needed")
	}
	return nil
}
