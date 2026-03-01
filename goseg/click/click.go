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

var (
	fixAcmeFn       = acme.Fix
	clearLusCodeFn  = luscode.ClearLusCode
	getLusCodeFn    = luscode.GetLusCode
	reviveDeskFn    = desk.ReviveDesk
	uninstallDeskFn = desk.UninstallDesk
	installDeskFn   = desk.InstallDesk
	getDeskFn       = desk.GetDesk
	mountDeskFn     = desk.MountDesk
	commitDeskFn    = desk.CommitDesk

	barExitFn = lifecycle.BarExit

	sendNotificationFn = notify.SendNotification
	sendPackFn         = pack.SendPack
	unlinkStorageFn    = storage.UnlinkStorage
	linkStorageFn      = storage.LinkStorage
	restoreTlonFn      = restore.RestoreTlon
	backupTlonFn       = backupdomain.BackupTlon
)

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
	if err := fixAcmeFn(patp); err != nil {
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
	clearLusCodeFn(patp)
}

// GetLusCode fetches +code from cache or executes click.
func GetLusCode(patp string) (string, error) {
	if err := validatePatp(patp); err != nil {
		return "", err
	}
	result, err := getLusCodeFn(patp)
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
	if err := reviveDeskFn(patp, deskName); err != nil {
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
	if err := uninstallDeskFn(patp, deskName); err != nil {
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
	if err := installDeskFn(patp, ship, deskName); err != nil {
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
	got, err := getDeskFn(patp, deskName, bypass)
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
	if err := mountDeskFn(patp, deskName); err != nil {
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
	if err := commitDeskFn(patp, deskName); err != nil {
		return fmt.Errorf("commit desk %s on %s: %w", deskName, patp, err)
	}
	return nil
}

// BarExit executes click drum-exit and clears +code cache.
func BarExit(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := barExitFn(patp); err != nil {
		return fmt.Errorf("bar exit for %s: %w", patp, err)
	}
	return nil
}

// SendNotification dispatches click hark notifications.
func SendNotification(patp string, payload structs.HarkNotification) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := sendNotificationFn(patp, payload); err != nil {
		return fmt.Errorf("send notification for %s: %w", patp, err)
	}
	return nil
}

// SendPack triggers a backup tarball command on the runtime.
func SendPack(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := sendPackFn(patp); err != nil {
		return fmt.Errorf("send pack for %s: %w", patp, err)
	}
	return nil
}

// UnlinkStorage clears remote storage configuration.
func UnlinkStorage(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := unlinkStorageFn(patp); err != nil {
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
	if err := linkStorageFn(patp, endpoint, svcAccount); err != nil {
		return fmt.Errorf("link storage for %s: %w", patp, err)
	}
	return nil
}

// RestoreTlon replays all known backup agents.
func RestoreTlon(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := restoreTlonFn(patp); err != nil {
		return fmt.Errorf("restore tlon for %s: %w", patp, err)
	}
	return nil
}

// BackupTlon exports the backup runtime integration.
func BackupTlon(patp string) error {
	if err := validatePatp(patp); err != nil {
		return err
	}
	if err := backupTlonFn(patp); err != nil {
		return fmt.Errorf("backup tlon for %s: %w", patp, err)
	}
	return nil
}
