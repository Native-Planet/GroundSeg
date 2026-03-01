package startram

// BackupInfrastructureService isolates backup transport/storage operations.
type BackupInfrastructureService interface {
	GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL string) (string, error)
	UploadBackup(ship, privateKey, filePath string) error
	Restore(req RestoreBackupRequest) error
}

type backupInfrastructureService struct{}

var defaultBackupInfrastructureService BackupInfrastructureService = backupInfrastructureService{}

func SetBackupInfrastructureService(service BackupInfrastructureService) {
	if service != nil {
		defaultBackupInfrastructureService = service
	}
}

func (backupInfrastructureService) GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL string) (string, error) {
	return getBackup(ship, timestamp, backupPassword, pubkey, endpointURL)
}

func (backupInfrastructureService) UploadBackup(ship, privateKey, filePath string) error {
	return uploadBackup(ship, privateKey, filePath)
}

func (backupInfrastructureService) Restore(req RestoreBackupRequest) error {
	return defaultRestoreWorker.Restore(req)
}

func GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL string) (string, error) {
	return defaultBackupInfrastructureService.GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL)
}

func UploadBackup(ship, privateKey, filePath string) error {
	return defaultBackupInfrastructureService.UploadBackup(ship, privateKey, filePath)
}

func RestoreBackupWithRequest(req RestoreBackupRequest) error {
	return defaultBackupInfrastructureService.Restore(req)
}
