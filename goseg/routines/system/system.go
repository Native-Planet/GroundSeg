package system

import (
	"time"

	"groundseg/routines/backup"
	"groundseg/routines/chop"
)

var (
	remoteBackupSleep = time.Sleep
	localBackupSleep  = time.Sleep
	sleepForChop      = time.Sleep
)

func StartBackupRoutines() {
	go TlonBackupRemote()
	go TlonBackupLocal()
}

func TlonBackupRemote() {
	remoteInterval, _ := backup.WaitIntervals()
	runRoutine(remoteInterval, backup.RunRemoteBackupPass, remoteBackupSleep)
}

func TlonBackupLocal() {
	_, localInterval := backup.WaitIntervals()
	runRoutine(localInterval, backup.RunLocalBackupPass, localBackupSleep)
}

func runRoutine(interval time.Duration, fn func(), sleep func(time.Duration)) {
	for {
		fn()
		sleep(interval)
	}
}

func StartChopRoutines() {
	go ChopAtLimit()
}

func ChopAtLimit() {
	chop.StartLoop(sleepForChop)
}
