package backup

import (
	"groundseg/startram/backup/transport"
)

// TransportClient abstracts backup transport calls for injection in tests.
type TransportClient = transport.TransportClient

// GetBackup retrieves a download link for an encrypted backup payload.
func GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL string, client TransportClient) (string, error) {
	return transport.GetBackup(ship, timestamp, backupPassword, pubkey, endpointURL, client)
}

// DownloadAndVerify downloads and validates a backup blob by MD5 checksum.
func DownloadAndVerify(link, md5hash string, client TransportClient) ([]byte, error) {
	return transport.DownloadAndVerify(link, md5hash, client)
}

// FetchRemoteBackup coordinates transport + integrity checks for a remote backup.
func FetchRemoteBackup(ship string, timestamp int, md5hash, password, pubkey, endpoint string, client TransportClient) ([]byte, error) {
	return transport.FetchRemoteBackup(ship, timestamp, md5hash, password, pubkey, endpoint, client)
}

// UploadBackup is the package-level transport entrypoint for backup upload.
func UploadBackup(ship, privateKey, endpointURL, pubkey, filePath string, client TransportClient) error {
	return transport.UploadBackup(ship, privateKey, endpointURL, pubkey, filePath, client)
}
