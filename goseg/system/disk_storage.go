package system

import "groundseg/system/storage"

func CreateGroundSegFilesystem(sel string) (string, error) {
	return storage.CreateGroundSegFilesystem(sel, resolveDiskSeams())
}

func SetupTmpDir() error {
	return storage.SetupTmpDir(resolveDiskSeams())
}

func IsMountedMMC(dirPath string) (bool, error) {
	return storage.IsMountedMMC(dirPath, resolveDiskSeams())
}

func RemoveMultipartFiles(path string) error {
	return storage.RemoveMultipartFiles(path, resolveDiskSeams())
}
