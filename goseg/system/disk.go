package system

import (
	"bufio"
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/anatol/smart.go"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

var (
	SmartResults = make(map[string]bool)
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

func RemoveMultipartFiles(path string) error {
	zap.L().Debug(fmt.Sprintf("Clearing multipart files from %v", path))
	// Read the contents of the directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}

	// Iterate through the contents
	for _, file := range files {
		// Check if the item is a file and its name starts with "multipart-"
		if !file.IsDir() && filepath.HasPrefix(file.Name(), "multipart-") {
			filePath := filepath.Join(path, file.Name())

			// Remove the file
			err := os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove file %s: %v", filePath, err)
			}
			zap.L().Debug(fmt.Sprintf("Removed file: %s", filePath))
		}
	}

	return nil
}

func SetupTmpDir() error {
	symlink := "/tmp"

	// remove old uploads
	if err := RemoveMultipartFiles(symlink); err != nil {
		zap.L().Warn(fmt.Sprintf("failed to remove multiparts: %v", err))
	}

	// check if /tmp is on emmc
	mmc, err := isMountedMMC(symlink)
	if err != nil {
		return fmt.Errorf("failed to check check /tmp mountpoint: %v", err)
	}

	// is mounted on emmc
	if mmc {
		isSym := false
		tmpDir, err := os.Lstat(symlink)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to get /tmp info: %v", err)
			}
		} else {
			isSym = tmpDir.Mode()&os.ModeSymlink != 0
		}

		// symlink?
		if !isSym {
			altDir := "/media/data/tmp"
			// make alt dir
			if err := os.MkdirAll(altDir, 1777); err != nil {
				return fmt.Errorf("failed to create alternate tmp directory: %v", err)
			}

			// delete /tmp
			if err := os.RemoveAll(symlink); err != nil {
				return fmt.Errorf("failed to remove %v: %v", symlink, err)
			}

			// create symlink
			if err := os.Symlink(altDir, symlink); err != nil {
				return fmt.Errorf("failed to create symlink from %v to %v: %v", altDir, symlink)
			}
		}
	}
	return nil
}

func isMountedMMC(dirPath string) (bool, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return false, fmt.Errorf("failed to get list of partitions")
	}
	/*
		the outer loop loops from child up the unix path
		until a mountpoint is found
	*/
OuterLoop:
	for {
		for _, p := range partitions {
			if p.Mountpoint == dirPath {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return true, nil
				} else {
					break OuterLoop
				}
			}
		}
		if dirPath == "/" {
			break
		}
		dirPath = path.Dir(dirPath) // Reduce the path by one level
	}
	return false, nil
}

func SmartCheckAllDrives(devices structs.LSBLKDevice) map[string]bool {
	results := make(map[string]bool)
	for _, disk := range devices.BlockDevices {
		if strings.Contains(disk.Name, "sd") {
			// sata
			pass, err := checkSataDrive(disk.Name)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to do SMART check sata drive %v: %v", disk.Name, err))
				continue
			} else {
				results[disk.Name] = pass
			}
		} else if strings.Contains(disk.Name, "nvme") {
			// nvme
			pass, err := checkNvmeDrive(disk.Name)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to do SMART check nvme drive %v: %v", disk.Name, err))
				continue
			} else {
				results[disk.Name] = pass
			}
		}
	}
	return results
}

func checkSataDrive(name string) (bool, error) {
	name = fmt.Sprintf("/dev/%v", name)
	zap.L().Info(fmt.Sprintf("running SMART check for sata drive %v", name))
	dev, err := smart.OpenSata(name)
	if err != nil {
		return false, err
	}
	log, err := dev.ReadSMARTData()
	if err != nil {
		return false, err
	}
	// #5 reallocated sector count < 50
	if sectorCount, exists := log.Attrs[5]; exists {
		if sectorCount.ValueRaw >= 50 {
			return false, nil
		}
	}

	// #9 power on hours < 30,000
	if powerOnHours, exists := log.Attrs[9]; exists {
		if powerOnHours.ValueRaw >= 30000 {
			return false, nil
		}
	}

	// #187 uncorrectable error count < 1
	if errorCount, exists := log.Attrs[187]; exists {
		if errorCount.ValueRaw >= 1 {
			return false, nil
		}
	}

	// #199 airflow temp celcius < 55
	if temperature, exists := log.Attrs[190]; exists {
		if temperature.ValueRaw >= 55 {
			return false, nil
		}
	}

	return true, nil
}

func checkNvmeDrive(name string) (bool, error) {
	name = fmt.Sprintf("/dev/%v", name)
	zap.L().Info(fmt.Sprintf("running SMART check for nvme drive %v", name))
	dev, err := smart.OpenNVMe(name)
	if err != nil {
		return false, err
	}
	log, err := dev.ReadSMART()
	if err != nil {
		return false, err
	}
	// crit warning
	if log.CritWarning != 0 {
		return false, nil
	}
	// temp
	if log.Temperature > 353 {
		return false, nil
	}
	// avail spare higher than spare threshold
	if log.AvailSpare <= log.SpareThresh {
		return false, nil
	}
	// percentage used below 100%
	if log.PercentUsed >= 100 {
		return false, nil
	}
	return true, nil
}
