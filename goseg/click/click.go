package click

import (
	"groundseg/click/acme"
	backupdomain "groundseg/click/backup"
	"groundseg/click/desk"
	"groundseg/click/lifecycle"
	"groundseg/click/luscode"
	"groundseg/click/notify"
	"groundseg/click/pack"
	"groundseg/click/restore"
	"groundseg/click/storage"
	"groundseg/structs"

	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type ClickRuntime interface {
	FixAcme(string) error
	ClearLusCode(string)
	GetLusCode(string) (string, error)
	ReviveDesk(string, string) error
	UninstallDesk(string, string) error
	InstallDesk(string, string, string) error
	GetDesk(string, string, bool) (string, error)
	MountDesk(string, string) error
	CommitDesk(string, string) error
	BarExit(string) error
	SendNotification(string, structs.HarkNotification) error
	SendPack(string) error
	UnlinkStorage(string) error
	LinkStorage(string, string, structs.MinIOServiceAccount) error
	RestoreTlon(string) error
	BackupTlon(string) error
}

type clickRuntime struct {
	fixAcmeFn       func(string) error
	clearLusCodeFn  func(string)
	getLusCodeFn    func(string) (string, error)
	reviveDeskFn    func(string, string) error
	uninstallDeskFn func(string, string) error
	installDeskFn   func(string, string, string) error
	getDeskFn       func(string, string, bool) (string, error)
	mountDeskFn     func(string, string) error
	commitDeskFn    func(string, string) error

	barExitFn func(string) error

	sendNotificationFn func(string, structs.HarkNotification) error
	sendPackFn         func(string) error
	unlinkStorageFn    func(string) error
	linkStorageFn      func(string, string, structs.MinIOServiceAccount) error
	restoreTlonFn      func(string) error
	backupTlonFn       func(string) error
}

func (runtime clickRuntime) FixAcme(patp string) error {
	return runtime.fixAcmeFn(patp)
}

func (runtime clickRuntime) ClearLusCode(patp string) {
	runtime.clearLusCodeFn(patp)
}

func (runtime clickRuntime) GetLusCode(patp string) (string, error) {
	return runtime.getLusCodeFn(patp)
}

func (runtime clickRuntime) ReviveDesk(patp, deskName string) error {
	return runtime.reviveDeskFn(patp, deskName)
}

func (runtime clickRuntime) UninstallDesk(patp, deskName string) error {
	return runtime.uninstallDeskFn(patp, deskName)
}

func (runtime clickRuntime) InstallDesk(patp, ship, deskName string) error {
	return runtime.installDeskFn(patp, ship, deskName)
}

func (runtime clickRuntime) GetDesk(patp, deskName string, bypass bool) (string, error) {
	return runtime.getDeskFn(patp, deskName, bypass)
}

func (runtime clickRuntime) MountDesk(patp, deskName string) error {
	return runtime.mountDeskFn(patp, deskName)
}

func (runtime clickRuntime) CommitDesk(patp, deskName string) error {
	return runtime.commitDeskFn(patp, deskName)
}

func (runtime clickRuntime) BarExit(patp string) error {
	return runtime.barExitFn(patp)
}

func (runtime clickRuntime) SendNotification(patp string, payload structs.HarkNotification) error {
	return runtime.sendNotificationFn(patp, payload)
}

func (runtime clickRuntime) SendPack(patp string) error {
	return runtime.sendPackFn(patp)
}

func (runtime clickRuntime) UnlinkStorage(patp string) error {
	return runtime.unlinkStorageFn(patp)
}

func (runtime clickRuntime) LinkStorage(patp, endpoint string, svcAccount structs.MinIOServiceAccount) error {
	return runtime.linkStorageFn(patp, endpoint, svcAccount)
}

func (runtime clickRuntime) RestoreTlon(patp string) error {
	return runtime.restoreTlonFn(patp)
}

func (runtime clickRuntime) BackupTlon(patp string) error {
	return runtime.backupTlonFn(patp)
}

func defaultClickRuntime() clickRuntime {
	return clickRuntime{
		fixAcmeFn:          acme.Fix,
		clearLusCodeFn:     luscode.ClearLusCode,
		getLusCodeFn:       luscode.GetLusCode,
		reviveDeskFn:       desk.ReviveDesk,
		uninstallDeskFn:    desk.UninstallDesk,
		installDeskFn:      desk.InstallDesk,
		getDeskFn:          desk.GetDesk,
		mountDeskFn:        desk.MountDesk,
		commitDeskFn:       desk.CommitDesk,
		barExitFn:          lifecycle.BarExit,
		sendNotificationFn: notify.SendNotification,
		sendPackFn:         pack.SendPack,
		unlinkStorageFn:    storage.UnlinkStorage,
		linkStorageFn:      storage.LinkStorage,
		restoreTlonFn:      restore.RestoreTlon,
		backupTlonFn:       backupdomain.BackupTlon,
	}
}

var runtimeRuntime ClickRuntime = defaultClickRuntime()

// SetRuntime replaces the internal click runtime used by exported wrappers.
func SetRuntime(handler ClickRuntime) {
	if handler == nil {
		runtimeRuntime = defaultClickRuntime()
		return
	}
	runtimeRuntime = handler
}

func resetClickRuntime() {
	SetRuntime(nil)
}

func getClickRuntime() ClickRuntime {
	return runtimeRuntime
}

func validatePatp(patp string) error {
	if strings.TrimSpace(patp) == "" {
		return errors.New("patp is required")
	}
	return nil
}

func validateArg(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// Fix acme dependencies for the given patp.
func FixAcme(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.FixAcme(patp); err != nil {
		return fmt.Errorf("fix acme for %s: %w", patp, err)
	}
	return nil
}

// ClearLusCode clears the cached +code value for a ship.
func ClearLusCode(patp string) {
	if err := validatePatp(patp); err != nil {
		zap.L().Warn(err.Error())
		return
	}
	runtimeRuntime.ClearLusCode(patp)
}

// GetLusCode fetches +code from cache or executes click.
func GetLusCode(patp string) (string, error) {
	if err := validatePatp(patp); err != nil {
		return "", err
	}
	result, err := runtimeRuntime.GetLusCode(patp)
	if err != nil {
		return "", fmt.Errorf("get +code for %s: %w", patp, err)
	}
	return result, nil
}

// ReviveDesk invokes the click command to revive a desk.
func ReviveDesk(patp, deskName string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := validateArg("desk", deskName); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := runtimeRuntime.ReviveDesk(patp, deskName); err != nil {
		return fmt.Errorf("revive desk %s on %s: %w", deskName, patp, err)
	}
	return nil
}

// UninstallDesk invokes the click command to uninstall a desk.
func UninstallDesk(patp, deskName string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := validateArg("desk", deskName); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := runtimeRuntime.UninstallDesk(patp, deskName); err != nil {
		return fmt.Errorf("uninstall desk %s on %s: %w", deskName, patp, err)
	}
	return nil
}

// InstallDesk invokes the click command to install a desk.
func InstallDesk(patp, ship, deskName string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := validateArg("ship", ship); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := validateArg("desk", deskName); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := runtimeRuntime.InstallDesk(patp, ship, deskName); err != nil {
		return fmt.Errorf("install desk %s on %s: %w", deskName, patp, err)
	}
	return nil
}

// GetDesk fetches desk state from cache or click.
func GetDesk(patp, deskName string, bypass bool) (string, error) {
	if err := validatePatp(patp); err != nil {
		return "", err
	}
	if err := validateArg("desk", deskName); err != nil {
		return "", fmt.Errorf("%w for %s", err, patp)
	}
	got, err := runtimeRuntime.GetDesk(patp, deskName, bypass)
	if err != nil {
		return "", fmt.Errorf("get desk %s on %s: %w", deskName, patp, err)
	}
	return got, nil
}

// MountDesk sends the hood mount command for a desk.
func MountDesk(patp, deskName string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := validateArg("desk", deskName); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := runtimeRuntime.MountDesk(patp, deskName); err != nil {
		return fmt.Errorf("mount desk %s on %s: %w", deskName, patp, err)
	}
	return nil
}

// CommitDesk sends the hood commit command for a desk.
func CommitDesk(patp, deskName string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := validateArg("desk", deskName); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := runtimeRuntime.CommitDesk(patp, deskName); err != nil {
		return fmt.Errorf("commit desk %s on %s: %w", deskName, patp, err)
	}
	return nil
}

// BarExit executes click drum-exit and clears +code cache.
func BarExit(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.BarExit(patp); err != nil {
		return fmt.Errorf("bar exit for %s: %w", patp, err)
	}
	return nil
}

// SendNotification dispatches click hark notifications.
func SendNotification(patp string, payload structs.HarkNotification) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.SendNotification(patp, payload); err != nil {
		return fmt.Errorf("send notification for %s: %w", patp, err)
	}
	return nil
}

// SendPack triggers a backup tarball command on the runtime.
func SendPack(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.SendPack(patp); err != nil {
		return fmt.Errorf("send pack for %s: %w", patp, err)
	}
	return nil
}

// UnlinkStorage clears remote storage configuration.
func UnlinkStorage(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.UnlinkStorage(patp); err != nil {
		return fmt.Errorf("unlink storage for %s: %w", patp, err)
	}
	return nil
}

// LinkStorage sets remote storage configuration.
func LinkStorage(patp, endpoint string, svcAccount structs.MinIOServiceAccount) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := validateArg("endpoint", endpoint); err != nil {
		return fmt.Errorf("%w for %s", err, patp)
	}
	if err := runtimeRuntime.LinkStorage(patp, endpoint, svcAccount); err != nil {
		return fmt.Errorf("link storage for %s: %w", patp, err)
	}
	return nil
}

// RestoreTlon replays all known backup agents.
func RestoreTlon(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.RestoreTlon(patp); err != nil {
		return fmt.Errorf("restore tlon for %s: %w", patp, err)
	}
	return nil
}

// BackupTlon exports the backup runtime integration.
func BackupTlon(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := runtimeRuntime.BackupTlon(patp); err != nil {
		return fmt.Errorf("backup tlon for %s: %w", patp, err)
	}
	return nil
}
