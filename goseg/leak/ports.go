package leak

import (
	"fmt"
	"goseg/logger"
	"net"
	"os"
	"path/filepath"
)

func makeSymlink(patp, sockLocation, symlinkPath string) (string, error) {
	err := os.MkdirAll(symlinkPath, 0755)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Failed to create directory: %v : %v", symlinkPath, err))
		return "", err
	}
	symlink := filepath.Join(symlinkPath, patp)
	// Check if the symlink already exists
	info, err := os.Lstat(symlink)
	if err == nil {
		// If it's a symlink
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(symlink)
			if err != nil {
				return "", fmt.Errorf("Error reading the symlink: %v", err)
			}
			// Check if it points to the desired target
			if target == sockLocation {
				return symlink, nil
			}
		}
		// Remove if it's a different symlink or a file
		err = os.Remove(symlink)
		if err != nil {
			return "", fmt.Errorf("Error removing existing file or symlink: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("Error checking symlink: %v", err)
	}
	// Create the symlink
	err = os.Symlink(sockLocation, symlink)
	if err != nil {
		return "", fmt.Errorf("Failed to symlink %v to %v: %v", sockLocation, symlink, err)
	} else {
		return symlink, nil
	}
}

func makeConnection(sockLocation string) net.Conn {
	conn, err := net.Dial("unix", sockLocation)
	if err != nil {
		return nil
	}
	return conn
}
