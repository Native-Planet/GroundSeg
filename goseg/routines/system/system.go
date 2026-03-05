package system

import (
	"context"
	"time"

	"groundseg/routines/backup"
	"groundseg/routines/chop"
)

var (
	remoteBackupSleep = time.Sleep
	localBackupSleep  = time.Sleep
	sleepForChop      = time.Sleep
)

func StartBackupRoutines() error {
	return backup.StartBackupRoutines()
}

func StartBackupRoutinesWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	remoteInterval, localInterval := backup.WaitIntervals()
	go runRoutineWithContext(ctx, remoteInterval, backup.RunRemoteBackupPass)
	go runRoutineWithContext(ctx, localInterval, backup.RunLocalBackupPass)
	<-ctx.Done()
	return nil
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

func runRoutineWithContext(ctx context.Context, interval time.Duration, fn func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		fn()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func StartChopRoutines() error {
	return chop.StartChopRoutines()
}

func StartChopRoutinesWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	go runChopWithContext(ctx)
	<-ctx.Done()
	return nil
}

func ChopAtLimit() {
	chop.StartLoop(sleepForChop)
}

func runChopWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		chop.RunAtLimitPass()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
