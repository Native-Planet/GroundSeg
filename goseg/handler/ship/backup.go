package ship

import (
	"context"
	"fmt"
	"groundseg/backupsvc"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/shipworkflow"
	"groundseg/startram"
	"groundseg/structs"
	"time"
)

var (
	runTransitionedOperationForBackupService = shipworkflow.RunTransitionedOperation
	persistShipBackupConfigForBackupService  = config.UpdateUrbitBackupConfig
	publishUrbitTransitionForBackupService   = func(ctx context.Context, transition structs.UrbitTransition) error {
		return events.DefaultEventRuntime().PublishUrbitTransition(ctx, transition)
	}
	sleepForBackupService                    = time.Sleep
	createLocalBackupForBackupService        = backupsvc.CreateLocalBackup
	backupDirForBackupService                = func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) }
	restoreBackupWithRequestForBackupService = startram.RestoreBackupWithRequest
)

func handleLocalToggleBackup(patp string) error {
	return runTransitionedOperationForBackupService(patp, "localTlonBackupsEnabled", "loading", "", 0, func() error {
		conf := config.UrbitConf(patp)
		if err := persistShipBackupConfigForBackupService(patp, func(updated *structs.UrbitBackupConfig) error {
			updated.LocalTlonBackup = !conf.LocalTlonBackup
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't set local backups: %v", err)
		}
		return nil
	})
}

func handleStartramToggleBackup(patp string) error {
	return runTransitionedOperationForBackupService(patp, "remoteTlonBackupsEnabled", "loading", "", 0, func() error {
		conf := config.UrbitConf(patp)
		if err := persistShipBackupConfigForBackupService(patp, func(updated *structs.UrbitBackupConfig) error {
			updated.RemoteTlonBackup = !conf.RemoteTlonBackup
			return nil
		}); err != nil {
			return fmt.Errorf("Couldn't set remote backups: %v", err)
		}
		return nil
	})
}

func handleLocalBackup(patp string) error {
	publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: "loading"})
	defer func() {
		sleepForBackupService(3 * time.Second)
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: ""})
	}()
	if err := createLocalBackupForBackupService(patp, backupDirForBackupService()); err != nil {
		text := fmt.Sprintf("failed to backup tlon for %v: %v", patp, err)
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: text})
		return fmt.Errorf("%s", text)
	}
	publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: "success"})
	return nil
}

func handleScheduleLocalBackup(patp string, urbitPayload structs.WsUrbitPayload) error {
	publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: "loading"})
	defer func() {
		sleepForBackupService(3 * time.Second)
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: ""})
	}()
	backupTime := urbitPayload.Payload.BackupTime
	if len(backupTime) != 4 {
		text := "invalid time format"
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: text})
		return fmt.Errorf("%s", text)
	}
	if err := persistShipBackupConfigForBackupService(patp, func(conf *structs.UrbitBackupConfig) error {
		conf.BackupTime = backupTime
		return nil
	}); err != nil {
		text := fmt.Sprintf("couldn't update urbit config: %v", err)
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: text})
		return fmt.Errorf("%s", text)
	}
	publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: "success"})
	return nil
}

func handleRestoreTlonBackup(patp string, urbitPayload structs.WsUrbitPayload) error {
	publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: "loading"})
	defer func() {
		sleepForBackupService(3 * time.Second)
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: ""})
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
	if err := restoreBackupWithRequestForBackupService(req); err != nil {
		text := fmt.Sprintf("failed to restore backup for %s: %v", patp, err)
		publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: text})
		return fmt.Errorf("%s", text)
	}
	publishUrbitTransitionForBackupService(context.Background(), structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: "success"})
	return nil
}
