package collectors

import (
	"groundseg/structs"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
)

type pierBackupSnapshot struct {
	remote       structs.Backup
	localDaily   structs.Backup
	localWeekly  structs.Backup
	localMonthly structs.Backup
}

func backupSnapshotForPiers(piers []string, remoteBackups []structs.Backup, runtime ...collectorRuntime) pierBackupSnapshot {
	return backupSnapshotForRuntime(
		collectorRuntimeOrDefault(runtime...),
		piers,
		remoteBackups,
	)
}

func backupSnapshotForRuntime(runtime collectorRuntime, piers []string, remoteBackups []structs.Backup) pierBackupSnapshot {
	return pierBackupSnapshot{
		remote:       flattenRemoteBackups(remoteBackups),
		localDaily:   localBackupsForPeriod(piers, "daily", runtime),
		localWeekly:  localBackupsForPeriod(piers, "weekly", runtime),
		localMonthly: localBackupsForPeriod(piers, "monthly", runtime),
	}
}

func flattenRemoteBackups(remoteBackups []structs.Backup) structs.Backup {
	remoteBackupMap := make(structs.Backup)
	for _, backup := range remoteBackups {
		for ship, backupInfo := range backup {
			remoteBackupMap[ship] = backupInfo
		}
	}
	return remoteBackupMap
}

func localBackupsForPeriod(piers []string, period string, runtime ...collectorRuntime) structs.Backup {
	return localBackupsForPeriodWithRuntime(collectorRuntimeOrDefault(runtime...), piers, period)
}

func localBackupsForPeriodWithRuntime(runtime collectorRuntime, piers []string, period string) structs.Backup {
	localBackups := make(structs.Backup)
	backupDir, ok := runtime.backupRoot()
	if !ok {
		zap.L().Warn("loading backup root failed: runtime dependency is not configured")
		return localBackups
	}
	for _, ship := range piers {
		if backupDir == "" {
			continue
		}
		shipBackups, err := filepath.Glob(filepath.Join(backupDir, ship, period, "*"))
		if err != nil {
			continue
		}
		for _, backup := range shipBackups {
			timestamp, err := strconv.Atoi(filepath.Base(backup))
			if err != nil {
				continue
			}
			localBackups[ship] = append(localBackups[ship], structs.BackupObject{Timestamp: timestamp, MD5: ""})
		}
	}
	return localBackups
}
