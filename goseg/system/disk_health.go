package system

import (
	"groundseg/structs"
	"groundseg/system/storage"
)

func SmartResultsSnapshot() map[string]bool {
	return storage.SmartResultsSnapshot()
}

func SetSmartResults(results map[string]bool) {
	storage.SetSmartResults(results)
}

func ListHardDisks() (structs.LSBLKDevice, error) {
	// keep public signature from root package while delegating to storage
	return storage.ListHardDisks(resolveDiskSeams())
}

func IsDevMounted(dev structs.BlockDev) bool {
	return storage.IsDevMounted(dev)
}

func SmartCheckAllDrives(devices structs.LSBLKDevice) map[string]bool {
	return storage.SmartCheckAllDrives(devices, resolveDiskSeams())
}
