package startram

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"go.uber.org/zap"
	"groundseg/startram/backup"
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
	url := "https://" + settings.EndpointURL + "/v1/backup/upload"
	encFile, err := backup.EncryptFile(filePath, privateKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("ship", ship); err != nil {
		return fmt.Errorf("failed to write ship field: %w", err)
	}
	if err := writer.WriteField("pubkey", settings.Pubkey); err != nil {
		return fmt.Errorf("failed to write pubkey field: %w", err)
	}
	part, err := writer.CreateFormFile("file", "backup.enc")
	if err != nil {
		return err
	}
	_, err = io.Copy(part, bytes.NewReader(encFile))
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make POST request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	return nil
}

// downloadAndVerify downloads an artifact and validates integrity.
func downloadAndVerify(link, md5hash string) ([]byte, error) {
	client := defaultAPIClient
	return backup.DownloadAndVerify(link, md5hash, client)
}
