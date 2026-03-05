package devsvc

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"groundseg/backupsvc"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
)

func resetDevSeams() {
	ResetDevSeams()
}

func devMessage(t *testing.T, action string, opts ...func(*structs.WsDevAction)) []byte {
	t.Helper()
	payload := structs.WsDevPayload{
		Payload: structs.WsDevAction{Action: action},
	}
	for _, opt := range opts {
		opt(&payload.Payload)
	}
	msg, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal dev payload failed: %v", err)
	}
	return msg
}

func TestDevHandlerRequiresDevModeAndKnownAction(t *testing.T) {
	t.Cleanup(resetDevSeams)
	isDevFn = func() bool { return false }

	if err := DevHandler(devMessage(t, "print-mounts")); err == nil {
		t.Fatalf("expected dev-mode restriction")
	}

	isDevFn = func() bool { return true }
	if err := DevHandler(devMessage(t, "unknown-action")); err == nil {
		t.Fatalf("expected unknown action error")
	}
}

func TestDevHandlerPrintMountsAndBackupTlon(t *testing.T) {
	t.Cleanup(resetDevSeams)
	isDevFn = func() bool { return true }

	listHardDisksForDev = func() (structs.LSBLKDevice, error) {
		return structs.LSBLKDevice{}, errors.New("lsblk failed")
	}
	if err := DevHandler(devMessage(t, "print-mounts")); err != nil {
		t.Fatalf("print-mounts should not fail hard: %v", err)
	}

	backupRoot := t.TempDir()
	backupDirFn = func() string { return backupRoot }
	configForDev = func() structs.SysConfig {
		return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{"~zod"}}}
	}
	called := false
	createLocalBackupForDev = func(patp, path string) error {
		called = true
		if patp != "~zod" {
			t.Fatalf("unexpected patp: %s", patp)
		}
		if path != backupRoot {
			t.Fatalf("expected backup dir %s, got %s", backupRoot, path)
		}
		return nil
	}
	if err := DevHandler(devMessage(t, "backup-tlon")); err != nil {
		t.Fatalf("backup-tlon failed: %v", err)
	}
	if !called {
		t.Fatalf("expected CreateBackup to be called")
	}
}

func TestDevHandlerRemoteBackupAndRestore(t *testing.T) {
	t.Cleanup(resetDevSeams)
	isDevFn = func() bool { return true }

	oldBasePath := config.BasePath()
	config.SetBasePath(t.TempDir())
	t.Cleanup(func() { config.SetBasePath(oldBasePath) })

	patp := "~zod"
	backupDirFn = func() string { return backupsvc.ResolveBackupRoot(config.BasePath()) }

	configForDev = func() structs.SysConfig {
		return structs.SysConfig{Connectivity: structs.ConnectivityConfig{Piers: []string{patp}, RemoteBackupPassword: "pw"}}
	}
	uploadedPath := ""
	uploadLatestBackupForDev = func(ship, pw, backupRoot string) error {
		if ship != patp || pw != "pw" {
			t.Fatalf("unexpected upload args: %s %s %s", ship, pw, backupRoot)
		}
		uploadedPath = backupRoot
		return nil
	}
	if err := DevHandler(devMessage(t, "remote-backup-tlon")); err != nil {
		t.Fatalf("remote-backup-tlon failed: %v", err)
	}
	if uploadedPath != backupDirFn() {
		t.Fatalf("expected backup root %s, got %s", backupDirFn(), uploadedPath)
	}

	var restoreReq startram.RestoreBackupRequest
	restoreBackupWithRequestForDev = func(req startram.RestoreBackupRequest) error {
		restoreReq = req
		return nil
	}
	if err := DevHandler(devMessage(t, "restore-tlon", func(a *structs.WsDevAction) { a.Patp = patp; a.Remote = true })); err != nil {
		t.Fatalf("restore-tlon failed: %v", err)
	}
	if restoreReq.Ship != patp || restoreReq.Source != startram.RestoreBackupSourceRemote || restoreReq.Mode != startram.RestoreBackupModeDevelopment {
		t.Fatalf("unexpected restore request: %+v", restoreReq)
	}
}

func TestDevHandlerStartramReminderAndToggle(t *testing.T) {
	t.Cleanup(resetDevSeams)
	isDevFn = func() bool { return true }

	configForDev = func() structs.SysConfig {
		conf := structs.SysConfig{
			Connectivity: structs.ConnectivityConfig{
				WgRegistered: true,
				Piers:        []string{"~zod", "~bus"},
			},
		}
		conf.Startram.StartramSetReminder.One = false
		conf.Startram.StartramSetReminder.Three = false
		conf.Startram.StartramSetReminder.Seven = false
		return conf
	}
	retrieveStartramForDev = func() (structs.StartramRetrieve, error) {
		return structs.StartramRetrieve{Lease: "2026-01-10"}, nil
	}
	nowForDev = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	urbitConfForDev = func(patp string) structs.UrbitDocker {
		if patp == "~zod" {
			return structs.UrbitDocker{
				UrbitFeatureConfig: structs.UrbitFeatureConfig{
					StartramReminder: true,
				},
			}
		}
		return structs.UrbitDocker{
			UrbitFeatureConfig: structs.UrbitFeatureConfig{
				StartramReminder: false,
			},
		}
	}
	notified := ""
	sendNotificationForDev = func(patp string, noti structs.HarkNotification) error {
		notified = patp
		if noti.Type != "startram-reminder" || noti.StartramDaysLeft <= 0 {
			t.Fatalf("unexpected notification payload: %+v", noti)
		}
		return nil
	}

	if err := DevHandler(devMessage(t, "startram-reminder")); err != nil {
		t.Fatalf("startram-reminder failed: %v", err)
	}
	if notified != "~zod" {
		t.Fatalf("expected only opted-in ship to be notified, got %s", notified)
	}

	configForDev = func() structs.SysConfig {
		return structs.SysConfig{Connectivity: structs.ConnectivityConfig{WgRegistered: false}}
	}
	if err := DevHandler(devMessage(t, "startram-reminder")); err == nil {
		t.Fatalf("expected no startram registration error")
	}

	updateCalled := false
	updateConfTypedForDev = func(...config.ConfigUpdateOption) error {
		updateCalled = true
		return nil
	}
	if err := DevHandler(devMessage(t, "startram-reminder-toggle", func(a *structs.WsDevAction) { a.Reminded = true })); err != nil {
		t.Fatalf("startram-reminder-toggle failed: %v", err)
	}
	if !updateCalled {
		t.Fatalf("expected config update call for reminder toggle")
	}

	t.Logf("local devsvc tests run against package seams")
}

func TestCheckDevMode(t *testing.T) {
	original := os.Args
	t.Cleanup(func() {
		os.Args = original
	})

	os.Args = []string{"groundseg", "dev"}
	if !CheckDevMode() {
		t.Fatal("expected dev mode when dev arg is present")
	}

	os.Args = []string{"groundseg", "start"}
	if CheckDevMode() {
		t.Fatal("expected non-dev mode without dev arg")
	}
}
