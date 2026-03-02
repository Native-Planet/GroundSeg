package startram

import (
	"fmt"
	"os/exec"

	"go.uber.org/zap"
	"groundseg/click"
	"groundseg/config"
	"groundseg/startram/backup"
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

type restoreBackupRuntime struct {
	fetchRemoteFn        func(ship string, timestamp int, md5hash string, settings config.StartramSettings) ([]byte, error)
	persistRemoteFn      func(ship string, timestamp int, data []byte) error
	fetchLocalFn         func(basePath string, ship string, bakType string, timestamp int) ([]byte, error)
	mountBaseDeskFn      func(ship string) error
	writeToVolumeFn      func(ship string, data []byte) error
	commitDeskFn         func(ship string, desk string) error
	restoreTlonFn        func(ship string) error
	getLocalBackupPathFn func(basePath string, ship, bakType string, timestamp int) string
	runtimeContextFn     func() config.RuntimeContext
}

var defaultRestoreBackupRuntime = restoreBackupRuntime{
	fetchRemoteFn:        fetchRemoteBackupWithAPI,
	persistRemoteFn:      backup.PersistRemoteBackup,
	fetchLocalFn:         backup.ReadLocalBackup,
	mountBaseDeskFn:      func(ship string) error { return click.MountDesk(ship, "base") },
	writeToVolumeFn:      writeBackupToVolumeWithAdapter,
	commitDeskFn:         click.CommitDesk,
	restoreTlonFn:        click.RestoreTlon,
	getLocalBackupPathFn: backup.ResolveLocalBackupPath,
	runtimeContextFn: func() config.RuntimeContext {
		return defaultConfigService.RuntimeContext()
	},
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

func restoreBackupProdWithRuntime(runtime restoreBackupRuntime, req RestoreBackupRequest) error {
	ship := req.Ship
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))

	data, err := func() ([]byte, error) {
		switch req.Source {
		case RestoreBackupSourceRemote:
			settings := defaultConfigService.StartramSettingsSnapshot()
			data, err := runtime.fetchRemoteFn(ship, req.Timestamp, req.MD5Hash, settings)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve remote backup: %w", err)
			}
			if err := runtime.persistRemoteFn(ship, req.Timestamp, data); err != nil {
				return nil, fmt.Errorf("failed to write backup to file: %w", err)
			}
			return data, nil
		case RestoreBackupSourceLocal:
			basePath := runtime.runtimeContextFn().BasePath
			data, err := runtime.fetchLocalFn(basePath, ship, req.LocalBackupType, req.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve local backup: %w", err)
			}
			if err := runtime.mountBaseDeskFn(ship); err != nil {
				return nil, fmt.Errorf("failed to mount base desk: %w", err)
			}
			return data, nil
		default:
			return nil, fmt.Errorf("unsupported restore source: %s", req.Source)
		}
	}()
	if err != nil {
		return err
	}

	if err := runtime.writeToVolumeFn(ship, data); err != nil {
		return fmt.Errorf("failed to write backup to volume: %w", err)
	}
	if err := runtime.commitDeskFn(ship, "base"); err != nil {
		return fmt.Errorf("failed to commit desk: %w", err)
	}
	if err := runtime.restoreTlonFn(ship); err != nil {
		return fmt.Errorf("failed to restore tlon: %w", err)
	}
	zap.L().Info(fmt.Sprintf("Successfully restored backup for %s", ship))
	return nil
}

func restoreBackupProd(req RestoreBackupRequest) error {
	return restoreBackupProdWithRuntime(defaultRestoreBackupRuntime, req)
}

func fetchRemoteBackupWithAPI(ship string, timestamp int, md5hash string, settings config.StartramSettings) ([]byte, error) {
	return backup.FetchRemoteBackup(ship, timestamp, md5hash, settings.RemoteBackupPassword, settings.Pubkey, settings.EndpointURL, defaultAPIClient)
}

func writeBackupToVolumeWithAdapter(ship string, data []byte) error {
	cmd := exec.Command("docker", "inspect", "-f", "{{ range .Mounts }}{{ if eq .Type \"volume\" }}{{ .Source }}{{ end }}{{ end }}", ship)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Docker volume location: %w", err)
	}
	return backup.WriteBackupToVolume(string(output), ship, data)
}

func retrieveRemoteBackup(ship string, timestamp int, md5hash string) ([]byte, error) {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return fetchRemoteBackupWithAPI(ship, timestamp, md5hash, settings)
}

func retrieveLocalBackup(ship string, timestamp int, bakType string) ([]byte, error) {
	basePath := defaultConfigService.RuntimeContext().BasePath
	backupPath := defaultRestoreBackupRuntime.getLocalBackupPathFn(basePath, ship, bakType, timestamp)
	zap.L().Info(fmt.Sprintf("Restoring local backup for %s at %d to %s", ship, timestamp, backupPath))
	return defaultRestoreBackupRuntime.fetchLocalFn(basePath, ship, bakType, timestamp)
}

func restoreBackupDev(ship string) error {
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
	settings := defaultConfigService.StartramSettingsSnapshot()
	res, err := Retrieve()
	if err != nil {
		return fmt.Errorf("failed to retrieve StarTram information: %w", err)
	}
	for _, backupData := range res.Backups {
		item, exists := backupData[ship]
		if !exists {
			continue
		}
		var highestTimestamp int
		var highestMD5 string
		for _, backupInfo := range item {
			if backupInfo.Timestamp > highestTimestamp {
				highestTimestamp = backupInfo.Timestamp
				highestMD5 = backupInfo.MD5
			}
		}
		if highestTimestamp <= 0 {
			continue
		}
		decrypted, err := fetchRemoteBackupWithAPI(ship, highestTimestamp, highestMD5, settings)
		if err != nil {
			return fmt.Errorf("failed to download and verify backup: %w", err)
		}
		if err := backup.PersistRemoteBackup(ship, highestTimestamp, decrypted); err != nil {
			return fmt.Errorf("failed to write backup to file: %w", err)
		}
		return nil
	}
	return fmt.Errorf("no backup found for %s", ship)
}
