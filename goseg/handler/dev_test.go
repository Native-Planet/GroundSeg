package handler

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"groundseg/backupsvc"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/system"
)

func resetDevSeams() {
	listHardDisksForDev = system.ListHardDisks
	confForDev = config.Conf
	createLocalBackupForDev = backupsvc.CreateLocalBackup
	uploadLatestBackupForDev = backupsvc.UploadLatestBackup
	restoreBackupWithRequestForDev = startram.RestoreBackupWithRequest
	retrieveStartramForDev = startram.Retrieve
	nowForDev = time.Now
	urbitConfForDev = config.UrbitConf
	sendNotificationForDev = click.SendNotification
	updateConfTypedForDev = config.UpdateConfTyped
	withStartramReminderAllForDev = config.WithStartramReminderAll
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
	isDev = false
	if err := DevHandler(devMessage(t, "print-mounts")); err == nil {
		t.Fatalf("expected dev-mode restriction")
	}

	isDev = true
	if err := DevHandler(devMessage(t, "unknown-action")); err == nil {
		t.Fatalf("expected unknown action error")
	}
}

func TestDevHandlerPrintMountsAndBackupTlon(t *testing.T) {
	t.Cleanup(resetDevSeams)
	isDev = true

	listHardDisksForDev = func() (structs.LSBLKDevice, error) {
		return structs.LSBLKDevice{}, errors.New("lsblk failed")
	}
	if err := DevHandler(devMessage(t, "print-mounts")); err != nil {
		t.Fatalf("print-mounts should not fail hard: %v", err)
	}

	backupRoot := t.TempDir()
	BackupDir = backupRoot
	confForDev = func() structs.SysConfig { return structs.SysConfig{Piers: []string{"~zod"}} }
	createCalled := false
	createLocalBackupForDev = func(patp, backupRoot string) error {
		createCalled = true
		if patp != "~zod" {
			t.Fatalf("unexpected patp: %s", patp)
		}
		if backupRoot != BackupDir {
			t.Fatalf("expected backup root %s, got %s", BackupDir, backupRoot)
		}
		return nil
	}
	if err := DevHandler(devMessage(t, "backup-tlon")); err != nil {
		t.Fatalf("backup-tlon failed: %v", err)
	}
	if !createCalled {
		t.Fatalf("expected CreateBackup to be called")
	}
}

func TestDevHandlerRemoteBackupAndRestore(t *testing.T) {
	t.Cleanup(resetDevSeams)
	isDev = true

	oldBasePath := config.BasePath
	config.BasePath = t.TempDir()
	t.Cleanup(func() { config.BasePath = oldBasePath })

	patp := "~zod"
	BackupDir = backupsvc.ResolveBackupRoot(config.BasePath)

	confForDev = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{patp}, RemoteBackupPassword: "pw"}
	}
	var uploadedPath string
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
	if uploadedPath != BackupDir {
		t.Fatalf("expected backup root %s, got %s", BackupDir, uploadedPath)
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
	isDev = true

	confForDev = func() structs.SysConfig {
		conf := structs.SysConfig{
			WgRegistered: true,
			Piers:        []string{"~zod", "~bus"},
		}
		conf.StartramSetReminder.One = false
		conf.StartramSetReminder.Three = false
		conf.StartramSetReminder.Seven = false
		return conf
	}
	retrieveStartramForDev = func() (structs.StartramRetrieve, error) {
		return structs.StartramRetrieve{Lease: "2026-01-10"}, nil
	}
	nowForDev = func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }
	urbitConfForDev = func(patp string) structs.UrbitDocker {
		if patp == "~zod" {
			return structs.UrbitDocker{StartramReminder: true}
		}
		return structs.UrbitDocker{StartramReminder: false}
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

	confForDev = func() structs.SysConfig { return structs.SysConfig{WgRegistered: false} }
	if err := DevHandler(devMessage(t, "startram-reminder")); err == nil {
		t.Fatalf("expected no startram registration error")
	}

	updateCalled := false
	updateConfTypedForDev = func(...config.ConfUpdateOption) error {
		updateCalled = true
		return nil
	}
	if err := DevHandler(devMessage(t, "startram-reminder-toggle", func(a *structs.WsDevAction) { a.Reminded = true })); err != nil {
		t.Fatalf("startram-reminder-toggle failed: %v", err)
	}
	if !updateCalled {
		t.Fatalf("expected config update call for reminder toggle")
	}
}
