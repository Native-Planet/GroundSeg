package storage

import (
	"bufio"
	"encoding/json"
	"os/exec"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/anatol/smart.go"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
	"groundseg/structs"
)

type DiskSeams struct {
	RunDiskCommandFn    func(string, ...string) (string, error)
	ListPartitionsFn    func(bool) ([]disk.PartitionStat, error)
	CheckSataDriveFn    func(string) (bool, error)
	CheckNvmeDriveFn    func(string) (bool, error)
	RemoveMultipartFn   func(string) error
	MkdirAllFn          func(string, os.FileMode) error
	RemoveAllFn         func(string) error
	SymlinkFn           func(string, string) error
	LstatFn             func(string) (os.FileInfo, error)
	MkdirFn             func(string, os.FileMode) error
	OpenFn              func(string) (*os.File, error)
	OpenFileFn          func(string, int, os.FileMode) (*os.File, error)
	MountAllCommandFn   func() error
	MkfsExt4CommandFn   func(string, string) error
	StatFn              func(string) (os.FileInfo, error)
	ReadDirFn           func(string) ([]os.DirEntry, error)
	RemoveFn            func(string) error
}

// FstabRecord represents one line in /etc/fstab.
type FstabRecord struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
	Dump       string
	Pass       string
}

func (record FstabRecord) Line() string {
	return fmt.Sprintf("%s %s %s %s %s %s", record.Device, record.MountPoint, record.FSType, record.Options, record.Dump, record.Pass)
}

func DefaultSeams() DiskSeams {
	return DiskSeams{
		RunDiskCommandFn:    runDiskCommand,
		ListPartitionsFn:    disk.Partitions,
		CheckSataDriveFn:    CheckSataDrive,
		CheckNvmeDriveFn:    CheckNvmeDrive,
		RemoveMultipartFn:   func(path string) error { return RemoveMultipartFiles(path, DiskSeams{}) },
		MkdirAllFn:          os.MkdirAll,
		RemoveAllFn:         os.RemoveAll,
		SymlinkFn:           os.Symlink,
		LstatFn:             os.Lstat,
		MkdirFn:             os.Mkdir,
		OpenFn:              os.Open,
		OpenFileFn:          os.OpenFile,
		MountAllCommandFn:    func() error { return runCommand("mount", "-a") },
		MkfsExt4CommandFn:    func(uuid, devPath string) error { return runCommand("mkfs.ext4", "-U", uuid, "-F", devPath) },
		StatFn:              os.Stat,
		ReadDirFn:           os.ReadDir,
		RemoveFn:            os.Remove,
	}
}

func normalizeSeams(overrides DiskSeams) DiskSeams {
	seams := DefaultSeams()
	if overrides.RunDiskCommandFn != nil {
		seams.RunDiskCommandFn = overrides.RunDiskCommandFn
	}
	if overrides.ListPartitionsFn != nil {
		seams.ListPartitionsFn = overrides.ListPartitionsFn
	}
	if overrides.CheckSataDriveFn != nil {
		seams.CheckSataDriveFn = overrides.CheckSataDriveFn
	}
	if overrides.CheckNvmeDriveFn != nil {
		seams.CheckNvmeDriveFn = overrides.CheckNvmeDriveFn
	}
	if overrides.RemoveMultipartFn != nil {
		seams.RemoveMultipartFn = overrides.RemoveMultipartFn
	}
	if overrides.MkdirAllFn != nil {
		seams.MkdirAllFn = overrides.MkdirAllFn
	}
	if overrides.RemoveAllFn != nil {
		seams.RemoveAllFn = overrides.RemoveAllFn
	}
	if overrides.SymlinkFn != nil {
		seams.SymlinkFn = overrides.SymlinkFn
	}
	if overrides.LstatFn != nil {
		seams.LstatFn = overrides.LstatFn
	}
	if overrides.MkdirFn != nil {
		seams.MkdirFn = overrides.MkdirFn
	}
	if overrides.OpenFn != nil {
		seams.OpenFn = overrides.OpenFn
	}
	if overrides.OpenFileFn != nil {
		seams.OpenFileFn = overrides.OpenFileFn
	}
	if overrides.MountAllCommandFn != nil {
		seams.MountAllCommandFn = overrides.MountAllCommandFn
	}
	if overrides.MkfsExt4CommandFn != nil {
		seams.MkfsExt4CommandFn = overrides.MkfsExt4CommandFn
	}
	if overrides.StatFn != nil {
		seams.StatFn = overrides.StatFn
	}
	if overrides.ReadDirFn != nil {
		seams.ReadDirFn = overrides.ReadDirFn
	}
	if overrides.RemoveFn != nil {
		seams.RemoveFn = overrides.RemoveFn
	}
	return seams
}

var (
	smartResultsMu sync.RWMutex
	smartResults   = make(map[string]bool)
)

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

func ListHardDisks(seams DiskSeams) (structs.LSBLKDevice, error) {
	seams = normalizeSeams(seams)
	var dev structs.LSBLKDevice
	out, err := seams.RunDiskCommandFn("lsblk", "-f", "--json", "--bytes")
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

func SmartCheckAllDrives(devices structs.LSBLKDevice, seams DiskSeams) map[string]bool {
	seams = normalizeSeams(seams)
	results := make(map[string]bool)
	for _, disk := range devices.BlockDevices {
		if strings.Contains(disk.Name, "sd") {
			pass, err := seams.CheckSataDriveFn(disk.Name)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to do SMART check sata drive %v: %v", disk.Name, err))
				continue
			}
			results[disk.Name] = pass
		} else if strings.Contains(disk.Name, "nvme") {
			pass, err := seams.CheckNvmeDriveFn(disk.Name)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to do SMART check nvme drive %v: %v", disk.Name, err))
				continue
			}
			results[disk.Name] = pass
		}
	}
	return results
}

func CreateGroundSegFilesystem(sel string, seams DiskSeams) (string, error) {
	seams = normalizeSeams(seams)
	dirPath, err := nextGroundSegPath(seams)
	if err != nil {
		return "", err
	}
	if err := seams.MkdirFn(dirPath, 0755); err != nil {
		return "", fmt.Errorf("Failed to create directory %s: %w", dirPath, err)
	}
	fsID, err := formatGroundSegFilesystem(sel, seams)
	if err != nil {
		return "", err
	}
	if err := reconcileGroundSegFstab(sel, dirPath, fsID, seams); err != nil {
		return "", err
	}
	return dirPath, nil
}

func SetupTmpDir(seams DiskSeams) error {
	seams = normalizeSeams(seams)
	symlink := "/tmp"

	// remove old uploads
	if err := seams.RemoveMultipartFn(symlink); err != nil {
		zap.L().Warn(fmt.Sprintf("failed to remove multiparts: %v", err))
	}

	mmc, err := IsMountedMMC(symlink, seams)
	if err != nil {
		return fmt.Errorf("failed to check check /tmp mountpoint: %w", err)
	}

	if mmc {
		isSym := false
		tmpDir, err := seams.LstatFn(symlink)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to get /tmp info: %w", err)
			}
		} else {
			isSym = tmpDir.Mode()&os.ModeSymlink != 0
		}

		if !isSym {
			altDir := "/media/data/tmp"
			if err := seams.MkdirAllFn(altDir, 1777); err != nil {
				return fmt.Errorf("failed to create alternate tmp directory: %w", err)
			}
			if err := seams.RemoveAllFn(symlink); err != nil {
				return fmt.Errorf("failed to remove %v: %w", symlink, err)
			}
			if err := seams.SymlinkFn(altDir, symlink); err != nil {
				return fmt.Errorf("failed to create symlink from %v to %v: %w", altDir, symlink, err)
			}
		}
	}
	return nil
}

func IsMountedMMC(dirPath string, seams DiskSeams) (bool, error) {
	seams = normalizeSeams(seams)
	partitions, err := seams.ListPartitionsFn(true)
	if err != nil {
		return false, fmt.Errorf("failed to get list of partitions: %w", err)
	}
	for {
		for _, p := range partitions {
			if p.Mountpoint == dirPath {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return true, nil
				}
				return false, nil
			}
		}
		if dirPath == "/" {
			break
		}
		dirPath = path.Dir(dirPath)
	}
	return false, nil
}

func RemoveMultipartFiles(path string, seams DiskSeams) error {
	seams = normalizeSeams(seams)
	files, err := seams.ReadDirFn(path)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "multipart-") {
			filePath := filepath.Join(path, file.Name())
			err := seams.RemoveFn(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove file %s: %w", filePath, err)
			}
			zap.L().Debug(fmt.Sprintf("Removed file: %s", filePath))
		}
	}
	return nil
}

func ParseFstabLine(raw string) (FstabRecord, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return FstabRecord{}, false
	}
	parts := strings.Fields(trimmed)
	if len(parts) < 6 {
		return FstabRecord{}, false
	}
	return FstabRecord{
		Device:     parts[0],
		MountPoint: parts[1],
		FSType:     parts[2],
		Options:    parts[3],
		Dump:       parts[4],
		Pass:       parts[5],
	}, true
}

func ReconcileFstabLines(lines []string, desired FstabRecord) ([]string, bool) {
	if desired.Device == "" || desired.MountPoint == "" {
		return lines, false
	}

	desiredKey := desired.Device + "|" + desired.MountPoint
	updated := make([]string, 0, len(lines)+1)
	seen := false
	changed := false

	for _, line := range lines {
		record, ok := ParseFstabLine(line)
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
		recordLine := desired.Line()
		if record.Line() != recordLine {
			recordLine = recordLine
			changed = true
		} else {
			recordLine = line
		}
		updated = append(updated, recordLine)
		seen = true
	}

	if !seen {
		updated = append(updated, desired.Line())
		changed = true
	}
	return updated, changed
}

func ReadFstabLines(path string, seams DiskSeams) ([]string, error) {
	seams = normalizeSeams(seams)
	file, err := seams.OpenFn(path)
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

func WriteFstabLines(path string, lines []string, seams DiskSeams) error {
	seams = normalizeSeams(seams)
	file, err := seams.OpenFileFn(path, os.O_WRONLY|os.O_TRUNC, 0644)
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

func CheckSataDrive(name string) (bool, error) {
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
	if sectorCount, exists := log.Attrs[5]; exists && sectorCount.ValueRaw >= 50 {
		return false, nil
	}
	if powerOnHours, exists := log.Attrs[9]; exists && powerOnHours.ValueRaw >= 30000 {
		return false, nil
	}
	if errorCount, exists := log.Attrs[187]; exists && errorCount.ValueRaw >= 1 {
		return false, nil
	}
	if temperature, exists := log.Attrs[190]; exists && temperature.ValueRaw >= 55 {
		return false, nil
	}
	return true, nil
}

func CheckNvmeDrive(name string) (bool, error) {
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
	if log.CritWarning != 0 {
		return false, nil
	}
	if log.Temperature > 353 {
		return false, nil
	}
	if log.AvailSpare <= log.SpareThresh {
		return false, nil
	}
	if log.PercentUsed >= 100 {
		return false, nil
	}
	return true, nil
}

func runCommand(command string, args ...string) error {
	c := exec.Command(command, args...)
	if _, err := c.Output(); err != nil {
		return fmt.Errorf("run command %q: %w", command, err)
	}
	return nil
}

func runDiskCommand(command string, args ...string) (string, error) {
	c := exec.Command(command, args...)
	out, err := c.Output()
	if err != nil {
		return string(out), fmt.Errorf("run command %q: %w", command, err)
	}
	return string(out), nil
}

func reconcileGroundSegFstab(sel string, dirPath string, fsID string, seams DiskSeams) error {
	blockDevices, err := ListHardDisks(seams)
	if err != nil {
		return fmt.Errorf("Failed to retrieve block devices: %w", err)
	}
	for _, dev := range blockDevices.BlockDevices {
		if dev.Name != sel {
			continue
		}
		rawLines, err := ReadFstabLines("/etc/fstab", seams)
		if err != nil {
			return fmt.Errorf("Error opening fstab: %w", err)
		}
		reconciledLines, changed := ReconcileFstabLines(rawLines, FstabRecord{
			Device:     "UUID=" + fsID,
			MountPoint: dirPath,
			FSType:     "ext4",
			Options:    "defaults,nofail",
			Dump:       "0",
			Pass:       "2",
		})
		if changed {
			if err := WriteFstabLines("/etc/fstab", reconciledLines, seams); err != nil {
				return fmt.Errorf("Error writing to fstab: %w", err)
			}
		}
		if err := seams.MountAllCommandFn(); err != nil {
			return fmt.Errorf("Failed to mount filesystem: %w", err)
		}
		break
	}
	return nil
}

func nextGroundSegPath(seams DiskSeams) (string, error) {
	for i := 1; ; i++ {
		path := fmt.Sprintf("/groundseg-%d", i)
		exists, err := filePathExists(path, seams.StatFn)
		if err != nil {
			return "", err
		}
		if !exists {
			return path, nil
		}
	}
}

func filePathExists(path string, statFn func(string) (os.FileInfo, error)) (bool, error) {
	if _, err := statFn(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path %s: %w", path, err)
	}
	return true, nil
}

func formatGroundSegFilesystem(sel string, seams DiskSeams) (string, error) {
	devPath := "/dev/" + sel
	fsID := uuid.NewString()
	if err := seams.MkfsExt4CommandFn(fsID, devPath); err != nil {
		return "", fmt.Errorf("Failed to create ext4 filesystem: %w", err)
	}
	return fsID, nil
}
