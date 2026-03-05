package startram

import (
	"fmt"

	"groundseg/click"
	"groundseg/config"
	"groundseg/startram/backup"
	"groundseg/startram/backup/restore"

	"go.uber.org/zap"
)

type RestoreBackupMode string

const (
	RestoreBackupModeProduction  RestoreBackupMode = "production"
	RestoreBackupModeDevelopment RestoreBackupMode = "development"
)

type RestoreBackupSource string

const (
	RestoreBackupSourceLocal  RestoreBackupSource = "local"
	RestoreBackupSourceRemote RestoreBackupSource = "remote"
)

type RestoreBackupRequest struct {
	Ship            string
	Timestamp       int
	MD5Hash         string
	LocalBackupType string
	Mode            RestoreBackupMode
	Source          RestoreBackupSource
}

func RestoreBackup(ship string, remote bool, timestamp int, md5hash string, dev bool, bakType string) error {
	req := RestoreBackupRequest{
		Ship:            ship,
		Timestamp:       timestamp,
		MD5Hash:         md5hash,
		LocalBackupType: bakType,
		Mode:            RestoreBackupModeProduction,
		Source:          RestoreBackupSourceLocal,
	}
	if dev {
		req.Mode = RestoreBackupModeDevelopment
	}
	if remote {
		req.Source = RestoreBackupSourceRemote
	}
	return RestoreBackupWithRequest(req)
}

func restoreBackupProd(req RestoreBackupRequest) error {
	restoreRuntime := restore.RestoreRuntime{
		FetchRemoteFn: func(ship string, timestamp int, md5hash string) ([]byte, error) {
			settings := defaultConfigService.StartramSettingsSnapshot()
			return backup.FetchRemoteBackup(ship, timestamp, md5hash, settings.RemoteBackupPassword, settings.Pubkey, settings.EndpointURL, defaultAPIClient)
		},
		PersistRemoteFn: backup.PersistRemoteBackup,
		FetchLocalFn:    backup.ReadLocalBackup,
		MountBaseDeskFn: func(ship string) error { return click.MountDesk(ship, "base") },
		WriteToVolumeFn: restore.WriteBackupToVolumeWithAdapter,
		CommitDeskFn:    click.CommitDesk,
		RestoreTlonFn:   click.RestoreTlon,
		GetBasePathFn: func() config.RuntimeContext {
			return defaultConfigService.RuntimeContext()
		},
	}

	zap.L().Info(fmt.Sprintf("Restoring backup for %s", req.Ship))
	err := restore.RestoreBackupProd(restoreRuntime, restore.RestoreBackupRequest{
		Ship:            req.Ship,
		Timestamp:       req.Timestamp,
		MD5Hash:         req.MD5Hash,
		LocalBackupType: req.LocalBackupType,
		Source:          string(req.Source),
	})
	if err != nil {
		return err
	}
	zap.L().Info(fmt.Sprintf("Successfully restored backup for %s", req.Ship))
	return nil
}

func restoreBackupDev(ship string) error {
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
	return restore.RestoreBackupDev(restore.RestoreDevRuntime{
		FetchConfigFn: Retrieve,
		FetchRemoteFn: func(s string, timestamp int, md5hash string) ([]byte, error) {
			settings := defaultConfigService.StartramSettingsSnapshot()
			return fetchRemoteBackupWithAPI(s, timestamp, md5hash, settings)
		},
		PersistRemoteFn: backup.PersistRemoteBackup,
	}, ship)
}

func fetchRemoteBackupWithAPI(ship string, timestamp int, md5hash string, settings config.StartramSettings) ([]byte, error) {
	return backup.FetchRemoteBackup(ship, timestamp, md5hash, settings.RemoteBackupPassword, settings.Pubkey, settings.EndpointURL, defaultAPIClient)
}

func retrieveRemoteBackup(ship string, timestamp int, md5hash string) ([]byte, error) {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return fetchRemoteBackupWithAPI(ship, timestamp, md5hash, settings)
}

func retrieveLocalBackup(ship string, timestamp int, bakType string) ([]byte, error) {
	basePath := defaultConfigService.RuntimeContext().BasePath
	backupPath := backup.ResolveLocalBackupPath(basePath, ship, bakType, timestamp)
	zap.L().Info(fmt.Sprintf("Restoring local backup for %s at %d to %s", ship, timestamp, backupPath))
	return backup.ReadLocalBackup(basePath, ship, bakType, timestamp)
}
