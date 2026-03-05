package storagefacade

import (
	"groundseg/structs"
	systemdisk "groundseg/system/disk"
	"groundseg/system/storage"
)

type FstabRecord = systemdisk.FstabRecord

func ParseFstabLine(raw string) (FstabRecord, bool) {
	return systemdisk.ParseFstabLine(raw)
}

func ReconcileFstabLines(lines []string, desired FstabRecord) ([]string, bool) {
	return systemdisk.ReconcileFstabLines(lines, desired)
}

func ReadFstabLines(path string, seams storage.DiskSeams) ([]string, error) {
	return storage.ReadFstabLines(path, seams)
}

func WriteFstabLines(path string, lines []string, seams storage.DiskSeams) error {
	return storage.WriteFstabLines(path, lines, seams)
}

func SmartResultsSnapshot() map[string]bool {
	return storage.SmartResultsSnapshot()
}

func SetSmartResults(results map[string]bool) {
	storage.SetSmartResults(results)
}

func ListHardDisks(seams storage.DiskSeams) (structs.LSBLKDevice, error) {
	return storage.ListHardDisks(seams)
}

func IsDevMounted(dev structs.BlockDev) bool {
	return storage.IsDevMounted(dev)
}

func SmartCheckAllDrives(devices structs.LSBLKDevice, seams storage.DiskSeams) map[string]bool {
	return storage.SmartCheckAllDrives(devices, seams)
}

func CreateGroundSegFilesystem(selectedDevice string, seams storage.DiskSeams) (string, error) {
	return storage.CreateGroundSegFilesystem(selectedDevice, seams)
}

func SetupTmpDir(seams storage.DiskSeams) error {
	return storage.SetupTmpDir(seams)
}

func IsMountedMMC(path string, seams storage.DiskSeams) (bool, error) {
	return storage.IsMountedMMC(path, seams)
}

func RemoveMultipartFiles(path string, seams storage.DiskSeams) error {
	return storage.RemoveMultipartFiles(path, seams)
}
