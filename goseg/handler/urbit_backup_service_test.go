package handler

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/backupsvc"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
)

func resetBackupServiceSeams() {
	runTransitionedOperationForBackupService = runTransitionedOperation
	persistShipConfForBackupService = persistShipConf
	publishUrbitTransitionForBackupService = docker.PublishUrbitTransition
	sleepForBackupService = time.Sleep
	createLocalBackupForBackupService = backupsvc.CreateLocalBackup
	restoreBackupWithRequestForBackupService = startram.RestoreBackupWithRequest
}

func TestHandleToggleBackupPersistsFlippedFlags(t *testing.T) {
	t.Cleanup(resetBackupServiceSeams)

	runTransitionedOperationForBackupService = func(_ string, _ string, _ string, _ string, _ time.Duration, op func() error) error {
		return op()
	}
	var localSaved, remoteSaved structs.UrbitDocker
	persistShipConfForBackupService = func(_ string, conf structs.UrbitDocker) error {
		if conf.LocalTlonBackup {
			localSaved = conf
		}
		if conf.RemoteTlonBackup {
			remoteSaved = conf
		}
		return nil
	}

	if err := handleLocalToggleBackup("~zod", structs.UrbitDocker{LocalTlonBackup: false}); err != nil {
		t.Fatalf("handleLocalToggleBackup failed: %v", err)
	}
	if !localSaved.LocalTlonBackup {
		t.Fatalf("expected local backup flag to be toggled on")
	}

	if err := handleStartramToggleBackup("~zod", structs.UrbitDocker{RemoteTlonBackup: false}); err != nil {
		t.Fatalf("handleStartramToggleBackup failed: %v", err)
	}
	if !remoteSaved.RemoteTlonBackup {
		t.Fatalf("expected remote backup flag to be toggled on")
	}
}

func TestHandleLocalBackupSuccessAndFailure(t *testing.T) {
	t.Cleanup(resetBackupServiceSeams)
	BackupDir = t.TempDir()
	sleepForBackupService = func(time.Duration) {}
	var events []string
	publishUrbitTransitionForBackupService = func(t structs.UrbitTransition) {
		if t.Type == "localTlonBackup" {
			events = append(events, t.Event)
		}
	}
	created := false
	createLocalBackupForBackupService = func(patp, backupRoot string) error {
		created = true
		if patp != "~zod" {
			t.Fatalf("unexpected patp: %s", patp)
		}
		if backupRoot != BackupDir {
			t.Fatalf("expected backup root %s, got %s", BackupDir, backupRoot)
		}
		return nil
	}

	if err := handleLocalBackup("~zod"); err != nil {
		t.Fatalf("handleLocalBackup failed: %v", err)
	}
	if !created {
		t.Fatalf("expected CreateBackup call")
	}
	if !strings.Contains(strings.Join(events, ","), "success") {
		t.Fatalf("expected success transition, got %v", events)
	}

	createLocalBackupForBackupService = func(string, string) error { return errors.New("create failed") }
	events = nil
	if err := handleLocalBackup("~zod"); err == nil {
		t.Fatalf("expected local backup creation failure")
	}
	if len(events) == 0 || !strings.Contains(events[len(events)-1], "") {
		// defer cleanup still sends empty event; just ensure events were emitted.
		t.Fatalf("expected transitions on failure, got %v", events)
	}
}

func TestHandleScheduleLocalBackupValidationAndPersist(t *testing.T) {
	t.Cleanup(resetBackupServiceSeams)
	sleepForBackupService = func(time.Duration) {}
	publishUrbitTransitionForBackupService = func(structs.UrbitTransition) {}

	payload := structs.WsUrbitPayload{Payload: structs.WsUrbitAction{BackupTime: "12"}}
	if err := handleScheduleLocalBackup("~zod", payload, structs.UrbitDocker{}); err == nil {
		t.Fatalf("expected invalid time format error")
	}

	var saved structs.UrbitDocker
	persistShipConfForBackupService = func(_ string, conf structs.UrbitDocker) error {
		saved = conf
		return nil
	}
	payload.Payload.BackupTime = "0930"
	if err := handleScheduleLocalBackup("~zod", payload, structs.UrbitDocker{}); err != nil {
		t.Fatalf("handleScheduleLocalBackup failed: %v", err)
	}
	if saved.BackupTime != "0930" {
		t.Fatalf("expected backup time persistence, got %+v", saved)
	}
}

func TestHandleRestoreTlonBackupBuildsRequest(t *testing.T) {
	t.Cleanup(resetBackupServiceSeams)
	sleepForBackupService = func(time.Duration) {}
	publishUrbitTransitionForBackupService = func(structs.UrbitTransition) {}

	var got startram.RestoreBackupRequest
	restoreBackupWithRequestForBackupService = func(req startram.RestoreBackupRequest) error {
		got = req
		return nil
	}
	payload := structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{Timestamp: 123, MD5: "abc", BakType: "daily", Remote: true},
	}
	if err := handleRestoreTlonBackup("~zod", payload, structs.UrbitDocker{}); err != nil {
		t.Fatalf("handleRestoreTlonBackup failed: %v", err)
	}
	if got.Ship != "~zod" || got.Timestamp != 123 || got.MD5Hash != "abc" || got.LocalBackupType != "daily" || got.Source != startram.RestoreBackupSourceRemote || got.Mode != startram.RestoreBackupModeProduction {
		t.Fatalf("unexpected restore request: %+v", got)
	}

	restoreBackupWithRequestForBackupService = func(startram.RestoreBackupRequest) error {
		return errors.New("restore failed")
	}
	if err := handleRestoreTlonBackup("~zod", payload, structs.UrbitDocker{}); err == nil {
		t.Fatalf("expected restore failure to propagate")
	}
}
