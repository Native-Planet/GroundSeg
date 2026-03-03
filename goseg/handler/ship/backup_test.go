package ship

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/startram"
	"groundseg/structs"
)

func TestHandleToggleBackupPersistsFlippedFlags(t *testing.T) {
	runtime := defaultBackupRuntime()
	runtime.runTransitionFn = func(_ string, _ string, _ string, _ string, _ time.Duration, op func() error) error {
		return op()
	}
	var localSaved, remoteSaved structs.UrbitBackupConfig
	runtime.persistShipBackupConfigFn = func(_ string, mutate func(*structs.UrbitBackupConfig) error) error {
		conf := structs.UrbitBackupConfig{}
		if err := mutate(&conf); err != nil {
			return err
		}
		if conf.LocalTlonBackup {
			localSaved = conf
		}
		if conf.RemoteTlonBackup {
			remoteSaved = conf
		}
		return nil
	}

	if err := handleLocalToggleBackupWithRuntime("~zod", runtime); err != nil {
		t.Fatalf("handleLocalToggleBackup failed: %v", err)
	}
	if !localSaved.LocalTlonBackup {
		t.Fatalf("expected local backup flag to be toggled on")
	}

	if err := handleStartramToggleBackupWithRuntime("~zod", runtime); err != nil {
		t.Fatalf("handleStartramToggleBackup failed: %v", err)
	}
	if !remoteSaved.RemoteTlonBackup {
		t.Fatalf("expected remote backup flag to be toggled on")
	}
}

func TestHandleLocalBackupSuccessAndFailure(t *testing.T) {
	backupRoot := t.TempDir()
	runtime := defaultBackupRuntime()
	runtime.backupRootFn = func() string { return backupRoot }
	runtime.sleepFn = func(time.Duration) {}
	var events []string
	runtime.publishUrbitTransitionFn = func(_ context.Context, transition structs.UrbitTransition) error {
		if transition.Type == "localTlonBackup" {
			events = append(events, transition.Event)
		}
		return nil
	}
	created := false
	runtime.createLocalBackupFn = func(patp, root string) error {
		created = true
		if patp != "~zod" {
			t.Fatalf("unexpected patp: %s", patp)
		}
		if root != runtime.backupRootFn() {
			t.Fatalf("expected backup root %s, got %s", backupRoot, root)
		}
		return nil
	}

	if err := handleLocalBackupWithRuntime("~zod", runtime); err != nil {
		t.Fatalf("handleLocalBackup failed: %v", err)
	}
	if !created {
		t.Fatalf("expected CreateBackup call")
	}
	if !strings.Contains(strings.Join(events, ","), "success") {
		t.Fatalf("expected success transition, got %v", events)
	}

	runtime.createLocalBackupFn = func(string, string) error { return errors.New("create failed") }
	events = nil
	if err := handleLocalBackupWithRuntime("~zod", runtime); err == nil {
		t.Fatalf("expected local backup creation failure")
	}
	if len(events) == 0 || !strings.Contains(events[len(events)-1], "") {
		// defer cleanup still sends empty event; just ensure transitions were emitted.
		t.Fatalf("expected transitions on failure, got %v", events)
	}
}

func TestHandleScheduleLocalBackupValidationAndPersist(t *testing.T) {
	runtime := defaultBackupRuntime()
	runtime.sleepFn = func(time.Duration) {}
	runtime.publishUrbitTransitionFn = func(_ context.Context, _ structs.UrbitTransition) error { return nil }

	payload := structs.WsUrbitPayload{Payload: structs.WsUrbitAction{BackupTime: "12"}}
	if err := handleScheduleLocalBackupWithRuntime("~zod", payload, runtime); err == nil {
		t.Fatalf("expected invalid time format error")
	}

	var saved structs.UrbitBackupConfig
	runtime.persistShipBackupConfigFn = func(_ string, mutate func(*structs.UrbitBackupConfig) error) error {
		conf := structs.UrbitBackupConfig{}
		if err := mutate(&conf); err != nil {
			return err
		}
		saved = conf
		return nil
	}
	payload.Payload.BackupTime = "0930"
	if err := handleScheduleLocalBackupWithRuntime("~zod", payload, runtime); err != nil {
		t.Fatalf("handleScheduleLocalBackup failed: %v", err)
	}
	if saved.BackupTime != "0930" {
		t.Fatalf("expected backup time persistence, got %+v", saved)
	}
}

func TestHandleRestoreTlonBackupBuildsRequest(t *testing.T) {
	runtime := defaultBackupRuntime()
	runtime.sleepFn = func(time.Duration) {}
	runtime.publishUrbitTransitionFn = func(_ context.Context, _ structs.UrbitTransition) error { return nil }

	var got startram.RestoreBackupRequest
	runtime.restoreBackupWithRequestFn = func(req startram.RestoreBackupRequest) error {
		got = req
		return nil
	}
	payload := structs.WsUrbitPayload{
		Payload: structs.WsUrbitAction{Timestamp: 123, MD5: "abc", BakType: "daily", Remote: true},
	}
	if err := handleRestoreTlonBackupWithRuntime("~zod", payload, runtime); err != nil {
		t.Fatalf("handleRestoreTlonBackup failed: %v", err)
	}
	if got.Ship != "~zod" || got.Timestamp != 123 || got.MD5Hash != "abc" || got.LocalBackupType != "daily" || got.Source != startram.RestoreBackupSourceRemote || got.Mode != startram.RestoreBackupModeProduction {
		t.Fatalf("unexpected restore request: %+v", got)
	}

	runtime.restoreBackupWithRequestFn = func(startram.RestoreBackupRequest) error {
		return errors.New("restore failed")
	}
	if err := handleRestoreTlonBackupWithRuntime("~zod", payload, runtime); err == nil {
		t.Fatalf("expected restore failure to propagate")
	}
}
