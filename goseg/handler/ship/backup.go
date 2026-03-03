package ship

import (
	"context"
	"fmt"
	"strings"
	"groundseg/backupsvc"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/logger"
	"groundseg/shipworkflow"
	"groundseg/startram"
	"groundseg/structs"
	"time"
)

type backupRuntime interface {
	runTransition(patp string, transitionType string, startEvent string, successEvent string, clearDelay time.Duration, operation func() error) error
	persistShipBackupConfig(patp string, persistFn func(*structs.UrbitBackupConfig) error) error
	publishUrbitTransition(ctx context.Context, transition structs.UrbitTransition) error
	sleepForBackup(delay time.Duration)
	createLocalBackupFor(patp string, backupRoot string) error
	backupRootPath() string
	restoreBackupWithRequest(req startram.RestoreBackupRequest) error
}

type backupRuntimeDependencies struct {
	runTransitionFn            func(string, string, string, string, time.Duration, func() error) error
	persistShipBackupConfigFn  func(string, func(*structs.UrbitBackupConfig) error) error
	publishUrbitTransitionFn   func(context.Context, structs.UrbitTransition) error
	sleepFn                    func(time.Duration)
	createLocalBackupFn        func(string, string) error
	backupRootFn               func() string
	restoreBackupWithRequestFn func(startram.RestoreBackupRequest) error
}

func (runtime backupRuntimeDependencies) runTransition(patp string, transitionType string, startEvent string, successEvent string, clearDelay time.Duration, operation func() error) error {
	return runtime.runTransitionFn(patp, transitionType, startEvent, successEvent, clearDelay, operation)
}

func (runtime backupRuntimeDependencies) persistShipBackupConfig(patp string, persistFn func(*structs.UrbitBackupConfig) error) error {
	return runtime.persistShipBackupConfigFn(patp, persistFn)
}

func (runtime backupRuntimeDependencies) publishUrbitTransition(ctx context.Context, transition structs.UrbitTransition) error {
	return runtime.publishUrbitTransitionFn(ctx, transition)
}

func (runtime backupRuntimeDependencies) sleepForBackup(delay time.Duration) {
	if runtime.sleepFn != nil {
		runtime.sleepFn(delay)
	}
}

func (runtime backupRuntimeDependencies) createLocalBackupFor(patp string, backupRoot string) error {
	if runtime.createLocalBackupFn == nil {
		return fmt.Errorf("backup runtime local backup callback is not configured")
	}
	return runtime.createLocalBackupFn(patp, backupRoot)
}

func (runtime backupRuntimeDependencies) backupRootPath() string {
	return runtime.backupRootFn()
}

func (runtime backupRuntimeDependencies) restoreBackupWithRequest(req startram.RestoreBackupRequest) error {
	return runtime.restoreBackupWithRequestFn(req)
}

func (runtime backupRuntimeDependencies) validate() error {
	missing := make([]string, 0, 7)
	if runtime.runTransitionFn == nil {
		missing = append(missing, "transition runtime")
	}
	if runtime.persistShipBackupConfigFn == nil {
		missing = append(missing, "config persistence")
	}
	if runtime.publishUrbitTransitionFn == nil {
		missing = append(missing, "transition publisher")
	}
	if runtime.sleepFn == nil {
		missing = append(missing, "sleep")
	}
	if runtime.createLocalBackupFn == nil {
		missing = append(missing, "local backup")
	}
	if runtime.backupRootFn == nil {
		missing = append(missing, "backup root")
	}
	if runtime.restoreBackupWithRequestFn == nil {
		missing = append(missing, "restore request")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("missing backup runtime callbacks: %s", strings.Join(missing, ", "))
}

func defaultBackupRuntime() backupRuntimeDependencies {
	return backupRuntimeDependencies{
		runTransitionFn: shipworkflow.RunTransitionedOperation,
		persistShipBackupConfigFn: func(patp string, persistFn func(*structs.UrbitBackupConfig) error) error {
			return config.UpdateUrbitSectionConfig(patp, config.UrbitConfigSectionBackup, persistFn)
		},
		publishUrbitTransitionFn: func(ctx context.Context, transition structs.UrbitTransition) error {
			return events.DefaultEventRuntime().PublishUrbitTransition(ctx, transition)
		},
		sleepFn:                    time.Sleep,
		createLocalBackupFn:        backupsvc.CreateLocalBackup,
		backupRootFn:               func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) },
		restoreBackupWithRequestFn: startram.RestoreBackupWithRequest,
	}
}

func resolveBackupRuntime(overrides ...backupRuntimeDependencies) (backupRuntimeDependencies, error) {
	runtime := defaultBackupRuntime()
	if len(overrides) == 0 {
		return runtime, nil
	}
	overridden := overrides[0]
	if overridden.runTransitionFn != nil {
		runtime.runTransitionFn = overridden.runTransitionFn
	}
	if overridden.persistShipBackupConfigFn != nil {
		runtime.persistShipBackupConfigFn = overridden.persistShipBackupConfigFn
	}
	if overridden.publishUrbitTransitionFn != nil {
		runtime.publishUrbitTransitionFn = overridden.publishUrbitTransitionFn
	}
	if overridden.sleepFn != nil {
		runtime.sleepFn = overridden.sleepFn
	}
	if overridden.createLocalBackupFn != nil {
		runtime.createLocalBackupFn = overridden.createLocalBackupFn
	}
	if overridden.backupRootFn != nil {
		runtime.backupRootFn = overridden.backupRootFn
	}
	if overridden.restoreBackupWithRequestFn != nil {
		runtime.restoreBackupWithRequestFn = overridden.restoreBackupWithRequestFn
	}
	if err := runtime.validate(); err != nil {
		return runtime, err
	}
	return runtime, nil
}

func handleLocalToggleBackup(patp string) error {
	return handleLocalToggleBackupWithRuntime(patp, backupRuntimeDependencies{})
}

func handleLocalToggleBackupWithRuntime(patp string, runtime backupRuntimeDependencies) error {
	resolvedRuntime, err := resolveBackupRuntime(runtime)
	if err != nil {
		return err
	}
	return resolvedRuntime.runTransition(patp, "localTlonBackupsEnabled", "loading", "", 0, func() error {
		conf := config.UrbitConf(patp)
		if err := resolvedRuntime.persistShipBackupConfig(patp, func(updated *structs.UrbitBackupConfig) error {
			updated.LocalTlonBackup = !conf.LocalTlonBackup
			return nil
		}); err != nil {
			return fmt.Errorf("couldn't set local backups: %w", err)
		}
		return nil
	})
}

func handleStartramToggleBackup(patp string) error {
	return handleStartramToggleBackupWithRuntime(patp, backupRuntimeDependencies{})
}

func handleStartramToggleBackupWithRuntime(patp string, runtime backupRuntimeDependencies) error {
	resolvedRuntime, err := resolveBackupRuntime(runtime)
	if err != nil {
		return err
	}
	return resolvedRuntime.runTransition(patp, "remoteTlonBackupsEnabled", "loading", "", 0, func() error {
		conf := config.UrbitConf(patp)
		if err := resolvedRuntime.persistShipBackupConfig(patp, func(updated *structs.UrbitBackupConfig) error {
			updated.RemoteTlonBackup = !conf.RemoteTlonBackup
			return nil
		}); err != nil {
			return fmt.Errorf("couldn't set remote backups: %w", err)
		}
		return nil
	})
}

func handleLocalBackup(patp string) error {
	return handleLocalBackupWithRuntime(patp, backupRuntimeDependencies{})
}

func handleLocalBackupWithRuntime(patp string, runtime backupRuntimeDependencies) error {
	resolvedRuntime, err := resolveBackupRuntime(runtime)
	if err != nil {
		return err
	}
	publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackup", "loading")
	defer func() {
		resolvedRuntime.sleepForBackup(3 * time.Second)
		publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackup", "")
	}()
	if err := resolvedRuntime.createLocalBackupFor(patp, resolvedRuntime.backupRootPath()); err != nil {
		err = fmt.Errorf("failed to backup tlon for %s: %w", patp, err)
		publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackup", err.Error())
		return err
	}
	publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackup", "success")
	return nil
}

func handleScheduleLocalBackup(patp string, urbitPayload structs.WsUrbitPayload) error {
	return handleScheduleLocalBackupWithRuntime(patp, urbitPayload, backupRuntimeDependencies{})
}

func handleScheduleLocalBackupWithRuntime(patp string, urbitPayload structs.WsUrbitPayload, runtime backupRuntimeDependencies) error {
	resolvedRuntime, err := resolveBackupRuntime(runtime)
	if err != nil {
		return err
	}
	publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackupSchedule", "loading")
	defer func() {
		resolvedRuntime.sleepForBackup(3 * time.Second)
		publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackupSchedule", "")
	}()
	backupTime := urbitPayload.Payload.BackupTime
	if len(backupTime) != 4 {
		err := fmt.Errorf("invalid time format")
		publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackupSchedule", err.Error())
		return err
	}
	if err := resolvedRuntime.persistShipBackupConfig(patp, func(conf *structs.UrbitBackupConfig) error {
		conf.BackupTime = backupTime
		return nil
	}); err != nil {
		err = fmt.Errorf("couldn't update urbit config: %w", err)
		text := err.Error()
		publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackupSchedule", text)
		return err
	}
	publishBackupUrbitTransition(resolvedRuntime, patp, "localTlonBackupSchedule", "success")
	return nil
}

func handleRestoreTlonBackup(patp string, urbitPayload structs.WsUrbitPayload) error {
	return handleRestoreTlonBackupWithRuntime(patp, urbitPayload, backupRuntimeDependencies{})
}

func handleRestoreTlonBackupWithRuntime(patp string, urbitPayload structs.WsUrbitPayload, runtime backupRuntimeDependencies) error {
	resolvedRuntime, err := resolveBackupRuntime(runtime)
	if err != nil {
		return err
	}
	publishBackupUrbitTransition(resolvedRuntime, patp, "handleRestoreTlonBackup", "loading")
	defer func() {
		resolvedRuntime.sleepForBackup(3 * time.Second)
		publishBackupUrbitTransition(resolvedRuntime, patp, "handleRestoreTlonBackup", "")
	}()
	restoreSource := startram.RestoreBackupSourceLocal
	if urbitPayload.Payload.Remote {
		restoreSource = startram.RestoreBackupSourceRemote
	}
	req := startram.RestoreBackupRequest{
		Ship:            patp,
		Timestamp:       urbitPayload.Payload.Timestamp,
		MD5Hash:         urbitPayload.Payload.MD5,
		LocalBackupType: urbitPayload.Payload.BakType,
		Mode:            startram.RestoreBackupModeProduction,
		Source:          restoreSource,
	}
	if err := resolvedRuntime.restoreBackupWithRequest(req); err != nil {
		err = fmt.Errorf("failed to restore backup for %s: %w", patp, err)
		publishBackupUrbitTransition(resolvedRuntime, patp, "handleRestoreTlonBackup", err.Error())
		return err
	}
	publishBackupUrbitTransition(resolvedRuntime, patp, "handleRestoreTlonBackup", "success")
	return nil
}

func publishBackupUrbitTransition(runtime backupRuntime, patp, transitionType, event string) {
	if err := runtime.publishUrbitTransition(context.Background(), structs.UrbitTransition{
		Patp:  patp,
		Type:  transitionType,
		Event: event,
	}); err != nil {
		logger.Warnf("failed to publish %s transition for %s: %v", transitionType, patp, err)
	}
}
