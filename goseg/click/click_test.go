package click

import (
	"errors"
	"fmt"
	"groundseg/click/acme"
	"groundseg/click/backup"
	"groundseg/click/desk"
	"groundseg/click/lifecycle"
	"groundseg/click/luscode"
	"groundseg/click/notify"
	"groundseg/click/pack"
	"groundseg/click/restore"
	"groundseg/click/storage"
	"groundseg/structs"
	"strings"
	"testing"
)

func resetClickSeams() {
	fixAcmeFn = acme.Fix
	clearLusCodeFn = luscode.ClearLusCode
	getLusCodeFn = luscode.GetLusCode
	reviveDeskFn = desk.ReviveDesk
	uninstallDeskFn = desk.UninstallDesk
	installDeskFn = desk.InstallDesk
	getDeskFn = desk.GetDesk
	mountDeskFn = desk.MountDesk
	commitDeskFn = desk.CommitDesk
	barExitFn = lifecycle.BarExit
	sendNotificationFn = notify.SendNotification
	sendPackFn = pack.SendPack
	unlinkStorageFn = storage.UnlinkStorage
	linkStorageFn = storage.LinkStorage
	restoreTlonFn = restore.RestoreTlon
	backupTlonFn = backup.BackupTlon
}

func TestValidatePatpAndArg(t *testing.T) {
	t.Cleanup(resetClickSeams)

	if err := validatePatp(""); err == nil {
		t.Fatalf("expected validatePatp to fail on empty input")
	}
	if err := validateArg("ship", ""); err == nil {
		t.Fatalf("expected validateArg to fail on empty input")
	}
	if err := validateArg("ship", "urbit"); err != nil {
		t.Fatalf("expected validateArg to pass for non-empty input")
	}
}

func TestClearLusCodeValidations(t *testing.T) {
	t.Cleanup(resetClickSeams)

	called := false
	clearLusCodeFn = func(patp string) {
		called = true
	}
	ClearLusCode("")
	if called {
		t.Fatalf("expected clearLusCode to skip invalid patp")
	}

	ClearLusCode("~zod")
	if !called {
		t.Fatalf("expected clearLusCode to be called for valid patp")
	}
}

func TestFixAcmePaths(t *testing.T) {
	t.Cleanup(resetClickSeams)

	called := ""
	fixAcmeFn = func(patp string) error {
		called = patp
		return nil
	}
	if err := FixAcme("~zod"); err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if called != "~zod" {
		t.Fatalf("expected fix acme call with ~zod, got %q", called)
	}

	expected := "fixing failed"
	fixAcmeFn = func(string) error { return errors.New(expected) }
	err := FixAcme("~zod")
	if err == nil || !strings.Contains(err.Error(), "fix acme for ~zod") || !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestGetLusCodePaths(t *testing.T) {
	t.Cleanup(resetClickSeams)

	getLusCodeFn = func(patp string) (string, error) {
		return "code-" + patp, nil
	}
	got, err := GetLusCode("~zod")
	if err != nil || got != "code-~zod" {
		t.Fatalf("unexpected lus code response: got=%q err=%v", got, err)
	}

	getLusCodeFn = func(string) (string, error) {
		return "", errors.New("lookup failed")
	}
	_, err = GetLusCode("~zod")
	if err == nil || !strings.Contains(err.Error(), "get +code for ~zod") {
		t.Fatalf("expected wrapped lookup error, got %v", err)
	}
}

func TestDeskFacadeValidationAndDelegation(t *testing.T) {
	t.Cleanup(resetClickSeams)

	var gotPatp, gotDesk, gotShip string
	var gotBypass bool
	reviveDeskFn = func(patp, desk string) error {
		gotPatp, gotDesk = patp, desk
		return nil
	}
	uninstallDeskFn = func(patp, desk string) error {
		gotPatp, gotDesk = patp, desk
		return nil
	}
	installDeskFn = func(patp, ship, desk string) error {
		gotPatp, gotShip, gotDesk = patp, ship, desk
		return nil
	}
	getDeskFn = func(patp, desk string, bypass bool) (string, error) {
		gotPatp, gotDesk, gotBypass = patp, desk, bypass
		return "ok", nil
	}
	mountDeskFn = func(patp, desk string) error {
		gotPatp, gotDesk = patp, desk
		return nil
	}
	commitDeskFn = func(patp, desk string) error {
		gotPatp, gotDesk = patp, desk
		return nil
	}

	if err := ReviveDesk("", "base"); err == nil {
		t.Fatalf("expected validation error for empty patp")
	}
	if err := UninstallDesk("~zod", ""); err == nil {
		t.Fatalf("expected validation error for empty desk")
	}
	if err := InstallDesk("~zod", "", "base"); err == nil {
		t.Fatalf("expected validation error for empty ship")
	}
	if err := UninstallDesk("~zod", "base"); err != nil {
		t.Fatalf("expected uninstall success, got %v", err)
	}
	if gotPatp != "~zod" || gotDesk != "base" {
		t.Fatalf("unexpected uninstall args: patp=%s desk=%s", gotPatp, gotDesk)
	}
	if err := InstallDesk("~bus", "~zod", "base"); err != nil {
		t.Fatalf("expected install success, got %v", err)
	}
	if gotShip != "~zod" || gotDesk != "base" || gotPatp != "~bus" {
		t.Fatalf("unexpected install args: patp=%s ship=%s desk=%s", gotPatp, gotShip, gotDesk)
	}
	if _, err := GetDesk("~mar", "garden", true); err != nil {
		t.Fatalf("expected get desk success, got %v", err)
	}
	if gotPatp != "~mar" || gotDesk != "garden" || !gotBypass {
		t.Fatalf("unexpected getDesk args: %+v %+v %t", gotPatp, gotDesk, gotBypass)
	}
	if err := MountDesk("~mar", "garden"); err != nil || gotPatp != "~mar" || gotDesk != "garden" {
		t.Fatalf("unexpected mount result: patp=%s desk=%s err=%v", gotPatp, gotDesk, err)
	}
	if err := CommitDesk("~mar", "garden"); err != nil {
		t.Fatalf("unexpected commit result: %v", err)
	}

	getDeskFn = func(string, string, bool) (string, error) { return "", errors.New("desk failed") }
	err := func() error {
		_, got := GetDesk("~zod", "base", false)
		return got
	}()
	if err == nil || !strings.Contains(err.Error(), "get desk base on ~zod") {
		t.Fatalf("expected wrapped getDesk error, got %v", err)
	}
}

func TestLifecycleAndNotificationFacade(t *testing.T) {
	t.Cleanup(resetClickSeams)

	var gotPatp string
	barExitFn = func(patp string) error {
		gotPatp = patp
		return nil
	}
	if err := BarExit("~zod"); err != nil {
		t.Fatalf("expected bar exit success, got %v", err)
	}
	if gotPatp != "~zod" {
		t.Fatalf("unexpected bar exit patp: %s", gotPatp)
	}

	sendNotificationFn = func(patp string, _ structs.HarkNotification) error { return nil }
	if err := SendNotification("~bus", structs.HarkNotification{Type: "test"}); err != nil {
		t.Fatalf("expected sendNotification wrapper success, got %v", err)
	}

	sendPackFn = func(patp string) error {
		gotPatp = patp
		return errors.New("pack failed")
	}
	err := SendPack("~nec")
	if err == nil || !strings.Contains(err.Error(), "send pack for ~nec") {
		t.Fatalf("expected wrapped send pack error, got %v", err)
	}
}

func TestStorageAndRuntimeFacade(t *testing.T) {
	t.Cleanup(resetClickSeams)

	var storagePatp string
	var unlinkPatp string
	var gotEndpoint string
	unlinkStorageFn = func(patp string) error {
		unlinkPatp = patp
		return nil
	}
	if err := UnlinkStorage("~zod"); err != nil {
		t.Fatalf("expected unlink success, got %v", err)
	}
	if unlinkPatp != "~zod" {
		t.Fatalf("unexpected unlink patp: %s", unlinkPatp)
	}

	linkStorageFn = func(patp, endpoint string, _ structs.MinIOServiceAccount) error {
		storagePatp = patp
		gotEndpoint = endpoint
		return nil
	}
	account := structs.MinIOServiceAccount{AccessKey: "ak", SecretKey: "sk", Alias: "alias", User: "user"}
	if err := LinkStorage("~bus", "https://storage", account); err != nil {
		t.Fatalf("expected link success, got %v", err)
	}
	if storagePatp != "~bus" || gotEndpoint != "https://storage" {
		t.Fatalf("unexpected link args: patp=%s endpoint=%s", storagePatp, gotEndpoint)
	}
	if err := LinkStorage("~bus", "", account); err == nil || !strings.Contains(err.Error(), "endpoint is required") {
		t.Fatalf("expected endpoint validation error, got %v", err)
	}

	restoreTlonFn = func(patp string) error {
		if !strings.HasPrefix(patp, "~") {
			return errors.New("bad patp")
		}
		return nil
	}
	if err := RestoreTlon("~bus"); err != nil {
		t.Fatalf("expected restore success, got %v", err)
	}

	backupTlonFn = func(patp string) error {
		if patp == "~zod" {
			return errors.New("backup failed")
		}
		return nil
	}
	err := BackupTlon("~zod")
	if err == nil || !strings.Contains(err.Error(), "backup tlon for ~zod") {
		t.Fatalf("expected wrapped backup error, got %v", err)
	}
}
func TestClearLusCodeUsesPatpValidation(t *testing.T) {
	t.Cleanup(resetClickSeams)
	called := false
	clearLusCodeFn = func(string) { called = true }

	ClearLusCode("  ")
	if called {
		t.Fatalf("expected invalid patp to be blocked")
	}
	ClearLusCode("~zod")
	if !called {
		t.Fatalf("expected clearLusCodeFn to be invoked for valid patp")
	}
}

func TestSendNotificationForwardsPayload(t *testing.T) {
	t.Cleanup(resetClickSeams)
	var gotPatp string
	var gotNotification structs.HarkNotification
	sendNotificationFn = func(patp string, n structs.HarkNotification) error {
		gotPatp = patp
		gotNotification = n
		return nil
	}

	notification := structs.HarkNotification{
		Type:             "startram-reminder",
		StartramDaysLeft: 7,
	}
	if err := SendNotification("~poc", notification); err != nil {
		t.Fatalf("expected notification forward success, got %v", err)
	}
	if gotPatp != "~poc" || gotNotification.Type != notification.Type {
		t.Fatalf("unexpected payload forwarding: patp=%q type=%q", gotPatp, gotNotification.Type)
	}
}

func TestPackAndRuntimeWrappers(t *testing.T) {
	t.Cleanup(resetClickSeams)
	if err := SendPack("   "); err == nil {
		t.Fatalf("expected patp validation error for empty patp")
	}

	var gotPatp string
	sendPackFn = func(patp string) error {
		gotPatp = patp
		return nil
	}
	if err := SendPack("~nov"); err != nil {
		t.Fatalf("expected SendPack success: %v", err)
	}
	if gotPatp != "~nov" {
		t.Fatalf("unexpected SendPack patp: %s", gotPatp)
	}

	sendPackFn = func(string) error { return errors.New("pack failed") }
	err := SendPack("~zod")
	if err == nil || !strings.Contains(err.Error(), "send pack for ~zod") {
		t.Fatalf("expected wrapped SendPack error, got %v", err)
	}

	restoreTlonFn = func(string) error { return fmt.Errorf("restore failed") }
	if err := RestoreTlon("~zod"); err == nil || !strings.Contains(err.Error(), "restore tlon for ~zod") {
		t.Fatalf("expected wrapped restore error, got %v", err)
	}
	backupTlonFn = func(string) error { return errors.New("backup failed") }
	if err := BackupTlon("~zod"); err == nil || !strings.Contains(err.Error(), "backup tlon for ~zod") {
		t.Fatalf("expected wrapped backup error, got %v", err)
	}
}
