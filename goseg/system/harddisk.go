package system

import (
	"bufio"
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"os"
	"os/exec"

	"github.com/google/uuid"
)

func ListHardDisks() (structs.LSBLKDevice, error) {
	var dev structs.LSBLKDevice
	out, err := runCommand("lsblk", "-f", "--json", "--bytes")
	if err != nil {
		return dev, fmt.Errorf("Failed to lsblk: %v", err)
	}
	err = json.Unmarshal([]byte(out), &dev)
	if err != nil {
		return dev, fmt.Errorf("Failed to unmarshal lsblk json: %v", err)
	}
	return dev, nil
}

func IsDevMounted(dev structs.BlockDev) bool {
	for _, t := range dev.Mountpoints {
		if t != "" {
			return true
		}
	}
	return false
}

func CreateGroundSegFilesystem(sel string) (string, error) {
	// Check for the existence of /groundseg-1 and increment if it exists
	var dirName string
	var dirPath string
	for i := 1; ; i++ {
		dirName = fmt.Sprintf("groundseg-%d", i)
		dirPath = "/" + dirName
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			break
		}
	}
	// Create the directory since it doesn't exist
	err := os.Mkdir(dirPath, 0755)
	if err != nil {
		return "", fmt.Errorf("Failed to create directory %s: %v", dirPath, err)
	}
	// Create an ext4 filesystem on this drive using it in its entirety.
	devPath := "/dev/" + sel
	uuid := uuid.NewString()
	cmd := exec.Command("mkfs.ext4", "-U", uuid, "-F", devPath)

	// Run the command and wait for it to complete
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Failed to create ext4 filesystem: %v", err)
	}

	// make sure to retrieve blockDevices AFTER creating the new fs!
	// this is so that the UUID is updated
	blockDevices, err := ListHardDisks()
	if err != nil {
		return "", fmt.Errorf("Failed to retrieve block devices: %v", err)
	}
	for _, dev := range blockDevices.BlockDevices {
		if dev.Name == sel {
			fstabEntry := fmt.Sprintf("UUID=%s %s %s %s %s %s\n", uuid, dirPath, "ext4", "defaults,nofail", "0", "2")
			// Read the existing fstab file
			file, err := os.Open("/etc/fstab")
			if err != nil {
				return "", fmt.Errorf("Error opening fstab: %v", err)
			}
			defer file.Close()

			var lines []string
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("Error reading fstab: %v", err)
			}

			// Append the new entry
			lines = append(lines, fstabEntry)

			// Write the updated content back to /etc/fstab
			file, err = os.OpenFile("/etc/fstab", os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return "", fmt.Errorf("Error opening fstab for writing: %v", err)
			}
			defer file.Close()

			writer := bufio.NewWriter(file)
			for _, line := range lines {
				_, err := writer.WriteString(line + "\n")
				if err != nil {
					return "", fmt.Errorf("Error writing to fstab: %v", err)
				}
			}
			writer.Flush()
			// Mount the newly created ext4 filesystem at /groundseg-<n>
			cmd = exec.Command("mount", "-a")
			err = cmd.Run()
			if err != nil {
				return "", fmt.Errorf("Failed to mount filesystem: %v", err)
			}
		}
	}
	return dirPath, nil
}
