package restore

import (
	"fmt"
	"groundseg/structs"
	"os/exec"

	"groundseg/config"
	"groundseg/startram/backup"
)

const (
	SourceLocal  = "local"
	SourceRemote = "remote"
)

// RestoreRuntime captures dependencies for restore orchestration.
type RestoreRuntime struct {
	FetchRemoteFn   func(ship string, timestamp int, md5hash string) ([]byte, error)
	PersistRemoteFn func(ship string, timestamp int, data []byte) error
	FetchLocalFn    func(basePath string, ship string, bakType string, timestamp int) ([]byte, error)
	MountBaseDeskFn func(ship string) error
	WriteToVolumeFn func(ship string, data []byte) error
	CommitDeskFn    func(ship string, desk string) error
	RestoreTlonFn   func(ship string) error
	GetBasePathFn   func() config.RuntimeContext
}

// RestoreBackupRequest describes restore input in the restore layer.
type RestoreBackupRequest struct {
	Ship            string
	Timestamp       int
	MD5Hash         string
	LocalBackupType string
	Source          string
}

// RestoreDevRuntime captures dependencies for the development restore flow.
type RestoreDevRuntime struct {
	FetchConfigFn   func() (structs.StartramRetrieve, error)
	FetchRemoteFn   func(ship string, timestamp int, md5hash string) ([]byte, error)
	PersistRemoteFn func(ship string, timestamp int, data []byte) error
}

// RestoreBackupProd restores backups in the production flow.
func RestoreBackupProd(runtime RestoreRuntime, req RestoreBackupRequest) error {
	ship := req.Ship
	data, err := func() ([]byte, error) {
		switch req.Source {
		case SourceRemote:
			data, err := runtime.FetchRemoteFn(ship, req.Timestamp, req.MD5Hash)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve remote backup: %w", err)
			}
			if err := runtime.PersistRemoteFn(ship, req.Timestamp, data); err != nil {
				return nil, fmt.Errorf("failed to write backup to file: %w", err)
			}
			return data, nil
		case SourceLocal:
			basePath := runtime.GetBasePathFn().BasePath
			data, err := runtime.FetchLocalFn(basePath, ship, req.LocalBackupType, req.Timestamp)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve local backup: %w", err)
			}
			if err := runtime.MountBaseDeskFn(ship); err != nil {
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

	if err := runtime.WriteToVolumeFn(ship, data); err != nil {
		return fmt.Errorf("failed to write backup to volume: %w", err)
	}
	if err := runtime.CommitDeskFn(ship, "base"); err != nil {
		return fmt.Errorf("failed to commit desk: %w", err)
	}
	if err := runtime.RestoreTlonFn(ship); err != nil {
		return fmt.Errorf("failed to restore tlon: %w", err)
	}
	return nil
}

// RestoreBackupDev restores backups in development mode.
func RestoreBackupDev(runtime RestoreDevRuntime, ship string) error {
	retrieve, err := runtime.FetchConfigFn()
	if err != nil {
		return fmt.Errorf("failed to retrieve StarTram information: %w", err)
	}
	for _, backupSet := range retrieve.Backups {
		backupInfo, exists := backupSet[ship]
		if !exists {
			continue
		}
		var highestTimestamp int
		var highestMD5 string
		for _, entry := range backupInfo {
			if entry.Timestamp > highestTimestamp {
				highestTimestamp = entry.Timestamp
				highestMD5 = entry.MD5
			}
		}
		if highestTimestamp <= 0 {
			break
		}
		decrypted, err := runtime.FetchRemoteFn(ship, highestTimestamp, highestMD5)
		if err != nil {
			return fmt.Errorf("failed to download and verify backup: %w", err)
		}
		if err := runtime.PersistRemoteFn(ship, highestTimestamp, decrypted); err != nil {
			return fmt.Errorf("failed to write backup to file: %w", err)
		}
		return nil
	}
	return fmt.Errorf("no backup found for %s", ship)
}

// WriteBackupToVolumeWithAdapter is the default adapter for restoring backup payloads into ship volumes.
func WriteBackupToVolumeWithAdapter(ship string, data []byte) error {
	cmd := exec.Command("docker", "inspect", "-f", "{{ range .Mounts }}{{ if eq .Type \"volume\" }}{{ .Source }}{{ end }}{{ end }}", ship)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Docker volume location: %w", err)
	}
	return backup.WriteBackupToVolume(string(output), ship, data)
}
