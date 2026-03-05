package system

import (
	"context"
	"fmt"
	"time"

	"groundseg/internal/workflow"
	"groundseg/routines/backup"
	"groundseg/routines/chop"

	"go.uber.org/zap"
)

var (
	remoteBackupSleep = time.Sleep
	localBackupSleep  = time.Sleep
	sleepForChop      = time.Sleep
	waitIntervalsFn   = backup.WaitIntervals
	runRemoteBackupFn = backup.RunRemoteBackupPass
	runLocalBackupFn  = backup.RunLocalBackupPass
	runChopPassFn     = chop.RunAtLimitPass
)

func StartBackupRoutines() error {
	return backup.StartBackupRoutines()
}

// StartBackupRoutinesWithContext starts backup routines and returns immediately.
func StartBackupRoutinesWithContext(ctx context.Context) error {
	_, err := StartBackupRoutinesWithContextHandle(ctx)
	return err
}

// StartBackupRoutinesWithContextHandle starts backup routines and returns a
// handle for observing terminal worker errors.
func StartBackupRoutinesWithContextHandle(ctx context.Context) (*workflow.AsyncRunHandle, error) {
	handle := workflow.StartAsync(ctx, RunBackupRoutinesWithContext)
	return handle, nil
}

// RunBackupRoutinesWithContext blocks until context cancellation.
func RunBackupRoutinesWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	remoteInterval, localInterval := waitIntervalsFn()
	return workflow.RunUntilDoneOrWorkerResult(
		ctx,
		func(workerCtx context.Context) error {
			return runRoutineWithContext(workerCtx, remoteInterval, runRemoteBackupFn)
		},
		func(workerCtx context.Context) error {
			return runRoutineWithContext(workerCtx, localInterval, runLocalBackupFn)
		},
	)
}

func TlonBackupRemote() {
	remoteInterval, _ := waitIntervalsFn()
	workflow.RunForever(remoteInterval, runRemoteBackupFn, remoteBackupSleep, func(err error) {
		zap.L().Error(fmt.Sprintf("background routine pass failed: %v", err))
	})
}

func TlonBackupLocal() {
	_, localInterval := waitIntervalsFn()
	workflow.RunForever(localInterval, runLocalBackupFn, localBackupSleep, func(err error) {
		zap.L().Error(fmt.Sprintf("background routine pass failed: %v", err))
	})
}

func runRoutineWithContext(ctx context.Context, interval time.Duration, fn func() error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := fn(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func StartChopRoutines() error {
	return chop.StartChopRoutines()
}

// StartChopRoutinesWithContext starts chop routines and returns immediately.
func StartChopRoutinesWithContext(ctx context.Context) error {
	_, err := StartChopRoutinesWithContextHandle(ctx)
	return err
}

// StartChopRoutinesWithContextHandle starts chop routines and returns a handle
// for observing terminal worker errors.
func StartChopRoutinesWithContextHandle(ctx context.Context) (*workflow.AsyncRunHandle, error) {
	handle := workflow.StartAsync(ctx, RunChopRoutinesWithContext)
	return handle, nil
}

// RunChopRoutinesWithContext blocks until context cancellation.
func RunChopRoutinesWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return runChopWithContext(ctx)
}

func ChopAtLimit() {
	chop.StartLoop(sleepForChop)
}

func runChopWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		if err := runChopPassFn(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
