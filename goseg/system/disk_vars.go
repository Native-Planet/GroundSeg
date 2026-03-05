package system

import (
	"os"
	"os/exec"

	"groundseg/system/storage"

	"github.com/shirou/gopsutil/disk"
)

var (
	runDiskCommandFn  = runCommand
	listPartitionsFn  = disk.Partitions
	checkSataDriveFn  = storage.CheckSataDrive
	checkNvmeDriveFn  = storage.CheckNvmeDrive
	removeMultipartFn = func(path string) error { return storage.RemoveMultipartFiles(path, storage.DiskSeams{}) }
	isMountedMMCFn    = func(path string) (bool, error) { return storage.IsMountedMMC(path, storage.DiskSeams{}) }
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
	readDirFn         = os.ReadDir
	removeFn          = os.Remove
)

func resolveDiskSeams() storage.DiskSeams {
	return storage.DiskSeams{
		RunDiskCommandFn:  runDiskCommandFn,
		ListPartitionsFn:  listPartitionsFn,
		CheckSataDriveFn:  checkSataDriveFn,
		CheckNvmeDriveFn:  checkNvmeDriveFn,
		RemoveMultipartFn: removeMultipartFn,
		MkdirAllFn:        mkdirAllFn,
		RemoveAllFn:       removeAllFn,
		SymlinkFn:         symlinkFn,
		LstatFn:           lstatFn,
		MkdirFn:           mkdirFn,
		OpenFn:            openFn,
		OpenFileFn:        openFileFn,
		MountAllCommandFn: mountAllCommandFn,
		MkfsExt4CommandFn: mkfsExt4CommandFn,
		StatFn:            statFn,
		ReadDirFn:         readDirFn,
		RemoveFn:          removeFn,
	}
}
