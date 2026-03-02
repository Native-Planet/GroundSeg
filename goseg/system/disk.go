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
	"sync"

	"github.com/anatol/smart.go"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

var (
	smartResultsMu    sync.RWMutex
	smartResults      = make(map[string]bool)
	runDiskCommandFn  = runCommand
	listPartitionsFn  = disk.Partitions
	checkSataDriveFn  = checkSataDrive
	checkNvmeDriveFn  = checkNvmeDrive
	removeMultipartFn = RemoveMultipartFiles
	isMountedMMCFn    = IsMountedMMC
	mkdirAllFn        = os.MkdirAll
	removeAllFn       = os.RemoveAll
	symlinkFn         = os.Symlink
	lstatFn           = os.Lstat
	mkdirFn           = os.Mkdir
	openFn            = os.Open
	openFileFn        = os.OpenFile
	mountAllCommandFn = func() error { return exec.Command("mount", "-a").Run() }
	mkfsExt4CommandFn = func(uuid, devPath string) error { return exec.Command("mkfs.ext4", "-U", uuid, "-F", devPath).Run() }
	statFn            = os.Stat
)

type fstabRecord struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
	Dump       string
	Pass       string
}

func (record fstabRecord) line() string {
	return fmt.Sprintf("%s %s %s %s %s %s", record.Device, record.MountPoint, record.FSType, record.Options, record.Dump, record.Pass)
}

func parseFstabLine(raw string) (fstabRecord, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return fstabRecord{}, false
	}
	parts := strings.Fields(trimmed)
	if len(parts) < 6 {
		return fstabRecord{}, false
	}
	return fstabRecord{
		Device:     parts[0],
		MountPoint: parts[1],
		FSType:     parts[2],
		Options:    parts[3],
		Dump:       parts[4],
		Pass:       parts[5],
	}, true
}

func reconcileFstabLines(lines []string, desired fstabRecord) ([]string, bool) {
	if desired.Device == "" || desired.MountPoint == "" {
		return lines, false
	}

	desiredKey := desired.Device + "|" + desired.MountPoint
	updated := make([]string, 0, len(lines)+1)
	seen := false
	changed := false

	for _, line := range lines {
		record, ok := parseFstabLine(line)
		if !ok {
			updated = append(updated, line)
			continue
		}
		if record.Device+"|"+record.MountPoint != desiredKey {
			updated = append(updated, line)
			continue
		}

		if seen {
			changed = true
			continue
		}

		recordLine := desired.line()
		if record.line() != recordLine {
			recordLine = recordLine
			changed = true
		} else {
			recordLine = line
		}
		updated = append(updated, recordLine)
		seen = true
	}

	if !seen {
		updated = append(updated, desired.line())
		changed = true
	}

	return updated, changed
}

func readFstabLines(path string) ([]string, error) {
	file, err := openFn(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func writeFstabLines(path string, lines []string) error {
	file, err := openFileFn(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}

func SmartResultsSnapshot() map[string]bool {
	smartResultsMu.RLock()
	defer smartResultsMu.RUnlock()

	clone := make(map[string]bool, len(smartResults))
	for device, healthy := range smartResults {
		clone[device] = healthy
	}
	return clone
}

func SetSmartResults(results map[string]bool) {
	smartResultsMu.Lock()
	defer smartResultsMu.Unlock()
	for device := range smartResults {
		delete(smartResults, device)
	}
	for device, healthy := range results {
		smartResults[device] = healthy
	}
}

func ListHardDisks() (structs.LSBLKDevice, error) {
	var dev structs.LSBLKDevice
	out, err := runDiskCommandFn("lsblk", "-f", "--json", "--bytes")
	if err != nil {
		return dev, fmt.Errorf("Failed to lsblk: %w", err)
	}
	err = json.Unmarshal([]byte(out), &dev)
	if err != nil {
		return dev, fmt.Errorf("Failed to unmarshal lsblk json: %w", err)
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
	dirPath, err := nextGroundSegPath()
	if err != nil {
		return "", err
	}
	if err := mkdirFn(dirPath, 0755); err != nil {
		return "", fmt.Errorf("Failed to create directory %s: %w", dirPath, err)
	}
	fsID, err := formatGroundSegFilesystem(sel)
	if err != nil {
		return "", err
	}
	if err := reconcileGroundSegFstab(sel, dirPath, fsID); err != nil {
		return "", err
	}
	return dirPath, nil
}

func nextGroundSegPath() (string, error) {
	for i := 1; ; i++ {
		path := fmt.Sprintf("/groundseg-%d", i)
		exists, err := filePathExists(path)
		if err != nil {
			return "", err
		}
		if !exists {
			return path, nil
		}
	}
}

func formatGroundSegFilesystem(sel string) (string, error) {
	devPath := "/dev/" + sel
	fsID := uuid.NewString()
	if err := mkfsExt4CommandFn(fsID, devPath); err != nil {
		return "", fmt.Errorf("Failed to create ext4 filesystem: %w", err)
	}
	return fsID, nil
}

func reconcileGroundSegFstab(sel string, dirPath string, fsID string) error {
	blockDevices, err := ListHardDisks()
	if err != nil {
		return fmt.Errorf("Failed to retrieve block devices: %w", err)
	}
	for _, dev := range blockDevices.BlockDevices {
		if dev.Name != sel {
			continue
		}
		rawLines, err := readFstabLines("/etc/fstab")
		if err != nil {
			return fmt.Errorf("Error opening fstab: %w", err)
		}
		reconciledLines, changed := reconcileFstabLines(rawLines, fstabRecord{
			Device:     "UUID=" + fsID,
			MountPoint: dirPath,
			FSType:     "ext4",
			Options:    "defaults,nofail",
			Dump:       "0",
			Pass:       "2",
		})
		if changed {
			if err := writeFstabLines("/etc/fstab", reconciledLines); err != nil {
				return fmt.Errorf("Error writing to fstab: %w", err)
			}
		}
		if err := mountAllCommandFn(); err != nil {
			return fmt.Errorf("Failed to mount filesystem: %w", err)
		}
		break
	}
	return nil
}

func filePathExists(path string) (bool, error) {
	if _, err := statFn(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path %s: %w", path, err)
	}
	return true, nil
}

func RemoveMultipartFiles(path string) error {
	zap.L().Debug(fmt.Sprintf("Clearing multipart files from %v", path))
	// Read the contents of the directory
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Iterate through the contents
	for _, file := range files {
		// Check if the item is a file and its name starts with "multipart-"
		if !file.IsDir() && filepath.HasPrefix(file.Name(), "multipart-") {
			filePath := filepath.Join(path, file.Name())

			// Remove the file
			err := os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove file %s: %w", filePath, err)
			}
			zap.L().Debug(fmt.Sprintf("Removed file: %s", filePath))
		}
	}

	return nil
}

func SetupTmpDir() error {
	symlink := "/tmp"

	// remove old uploads
	if err := removeMultipartFn(symlink); err != nil {
		zap.L().Warn(fmt.Sprintf("failed to remove multiparts: %v", err))
	}

	// check if /tmp is on emmc
	mmc, err := isMountedMMCFn(symlink)
	if err != nil {
		return fmt.Errorf("failed to check check /tmp mountpoint: %w", err)
	}

	// is mounted on emmc
	if mmc {
		isSym := false
		tmpDir, err := lstatFn(symlink)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to get /tmp info: %w", err)
			}
		} else {
			isSym = tmpDir.Mode()&os.ModeSymlink != 0
		}

		// symlink?
		if !isSym {
			altDir := "/media/data/tmp"
			// make alt dir
			if err := mkdirAllFn(altDir, 1777); err != nil {
				return fmt.Errorf("failed to create alternate tmp directory: %w", err)
			}

			// delete /tmp
			if err := removeAllFn(symlink); err != nil {
				return fmt.Errorf("failed to remove %v: %w", symlink, err)
			}

			// create symlink
			if err := symlinkFn(altDir, symlink); err != nil {
				return fmt.Errorf("failed to create symlink from %v to %v: %w", altDir, symlink, err)
			}
		}
	}
	return nil
}

func IsMountedMMC(dirPath string) (bool, error) {
	partitions, err := listPartitionsFn(true)
	if err != nil {
		return false, fmt.Errorf("failed to get list of partitions: %w", err)
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
			pass, err := checkSataDriveFn(disk.Name)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to do SMART check sata drive %v: %v", disk.Name, err))
				continue
			} else {
				results[disk.Name] = pass
			}
		} else if strings.Contains(disk.Name, "nvme") {
			// nvme
			pass, err := checkNvmeDriveFn(disk.Name)
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
