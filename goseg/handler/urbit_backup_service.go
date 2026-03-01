package handler

import (
	"fmt"
	"groundseg/backupsvc"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"time"
)

var (
	runTransitionedOperationForBackupService = runTransitionedOperation
	persistShipConfForBackupService          = persistShipConf
	publishUrbitTransitionForBackupService   = docker.PublishUrbitTransition
	sleepForBackupService                    = time.Sleep
	createLocalBackupForBackupService        = backupsvc.CreateLocalBackup
	restoreBackupWithRequestForBackupService = startram.RestoreBackupWithRequest
)

func handleLocalToggleBackup(patp string, shipConf structs.UrbitDocker) error {
	return runTransitionedOperationForBackupService(patp, "localTlonBackupsEnabled", "loading", "", 0, func() error {
		shipConf.LocalTlonBackup = !shipConf.LocalTlonBackup
		if err := persistShipConfForBackupService(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't set local backups: %v", err)
		}
		return nil
	})
}

func handleStartramToggleBackup(patp string, shipConf structs.UrbitDocker) error {
	return runTransitionedOperationForBackupService(patp, "remoteTlonBackupsEnabled", "loading", "", 0, func() error {
		shipConf.RemoteTlonBackup = !shipConf.RemoteTlonBackup
		if err := persistShipConfForBackupService(patp, shipConf); err != nil {
			return fmt.Errorf("Couldn't set remote backups: %v", err)
		}
		return nil
	})
}

func handleLocalBackup(patp string) error {
	publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: "loading"})
	defer func() {
		sleepForBackupService(3 * time.Second)
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: ""})
	}()
	if err := createLocalBackupForBackupService(patp, BackupDir); err != nil {
		text := fmt.Sprintf("failed to backup tlon for %v: %v", patp, err)
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: text})
		return fmt.Errorf("%s", text)
	}
	publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackup", Event: "success"})
	return nil
}

func handleScheduleLocalBackup(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: "loading"})
	defer func() {
		sleepForBackupService(3 * time.Second)
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: ""})
	}()
	backupTime := urbitPayload.Payload.BackupTime
	if len(backupTime) != 4 {
		text := "invalid time format"
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: text})
		return fmt.Errorf("%s", text)
	}
	shipConf.BackupTime = backupTime
	if err := persistShipConfForBackupService(patp, shipConf); err != nil {
		text := fmt.Sprintf("couldn't update urbit config: %v", err)
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: text})
		return fmt.Errorf("%s", text)
	}
	publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "localTlonBackupSchedule", Event: "success"})
	return nil
}

func handleRestoreTlonBackup(patp string, urbitPayload structs.WsUrbitPayload, shipConf structs.UrbitDocker) error {
	publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: "loading"})
	defer func() {
		sleepForBackupService(3 * time.Second)
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: ""})
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
		publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: text})
		return fmt.Errorf("%s", text)
	}
	publishUrbitTransitionForBackupService(structs.UrbitTransition{Patp: patp, Type: "handleRestoreTlonBackup", Event: "success"})
	return nil
}
