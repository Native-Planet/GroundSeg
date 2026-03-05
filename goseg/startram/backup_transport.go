package startram

import (
	"fmt"
	"groundseg/startram/backup"

	"go.uber.org/zap"
)

// getBackup requests a signed download URL for a backup archive.
func getBackup(ship, timestamp, backupPassword, pubkey, endpointUrl string) (string, error) {
	client := defaultAPIClient
	return backup.GetBackup(ship, timestamp, backupPassword, pubkey, endpointUrl, client)
}

// uploadBackup uploads an encrypted backup blob to startram.
func uploadBackup(ship, privateKey, filePath string) error {
	zap.L().Info(fmt.Sprintf("Uploading backup for %s", ship))
	settings := defaultConfigService.StartramSettingsSnapshot()
	return backup.UploadBackup(ship, privateKey, settings.EndpointURL, settings.Pubkey, filePath, defaultAPIClient)
}

// downloadAndVerify downloads an artifact and validates integrity.
func downloadAndVerify(link, md5hash string) ([]byte, error) {
	client := defaultAPIClient
	return backup.DownloadAndVerify(link, md5hash, client)
}
