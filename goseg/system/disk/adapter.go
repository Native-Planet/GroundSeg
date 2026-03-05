package disk

import (
	"groundseg/structs"
	"groundseg/system/storage"
)

// FstabRecord remains source-compatible with storage while the system boundary migrates.
type FstabRecord = storage.FstabRecord

var ParseFstabLine = storage.ParseFstabLine
var ReconcileFstabLines = storage.ReconcileFstabLines
var SmartResultsSnapshot = storage.SmartResultsSnapshot
var SetSmartResults = storage.SetSmartResults
var IsDevMounted = storage.IsDevMounted

func ReadFstabLines(path string) ([]string, error) {
	return storage.ReadFstabLines(path, storage.DefaultSeams())
}

func WriteFstabLines(path string, lines []string) error {
	return storage.WriteFstabLines(path, lines, storage.DefaultSeams())
}

func ListHardDisks() (structs.LSBLKDevice, error) {
	return storage.ListHardDisks(storage.DefaultSeams())
}

func SmartCheckAllDrives(devices structs.LSBLKDevice) map[string]bool {
	return storage.SmartCheckAllDrives(devices, storage.DefaultSeams())
}

func CreateGroundSegFilesystem(selectedDevice string) (string, error) {
	return storage.CreateGroundSegFilesystem(selectedDevice, storage.DefaultSeams())
}

func SetupTmpDir() error {
	return storage.SetupTmpDir(storage.DefaultSeams())
}

func IsMountedMMC(path string) (bool, error) {
	return storage.IsMountedMMC(path, storage.DefaultSeams())
}

func RemoveMultipartFiles(path string) error {
	return storage.RemoveMultipartFiles(path, storage.DefaultSeams())
}
