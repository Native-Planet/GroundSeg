package startram

import (
	"archive/tar"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"groundseg/click"
	"groundseg/structs"
	"groundseg/system"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
	"go.uber.org/zap"
)

// get a backup download URL for a specific ship and backup timestamp from retrieve blob
func getBackup(ship, timestamp, backupPassword, pubkey, endpointUrl string) (string, error) {
	reqData := structs.GetBackupRequest{
		Ship:      ship,
		Pubkey:    pubkey,
		Timestamp: timestamp,
	}
	url := "https://" + endpointUrl + "/v1/backup/get"
	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request data: %w", err)
	}
	resp, err := apiPost(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make POST request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}
	var getBackupResp structs.GetBackupResponse
	if err := json.Unmarshal(body, &getBackupResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response data: %w", err)
	}
	return getBackupResp.Result, nil
}

// upload an encrypted backup blob to startram
func uploadBackup(ship, privateKey, filePath string) error {
	zap.L().Info(fmt.Sprintf("Uploading backup for %s", ship))
	settings := defaultConfigService.StartramSettingsSnapshot()
	url := "https://" + settings.EndpointURL + "/v1/backup/upload"
	// encrypt the file
	encFile, err := encryptFile(filePath, privateKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}
	// create the request body
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
	// make the request
	req, err := http.NewRequest("POST", url, body)
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

// encrypt an arbitrary file with a path and key
func encryptFile(filename string, keyString string) ([]byte, error) {
	// Read the file
	plaintext, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Derive a 256-bit key from the provided string
	key := sha256.Sum256([]byte(keyString))

	// Create a new AES cipher block
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create a new GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Create a nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to create nonce: %w", err)
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

func decryptFile(file []byte, keyString string) ([]byte, error) {
	// Derive a 256-bit key from the provided string
	key := sha256.Sum256([]byte(keyString))

	// Create a new AES cipher block
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create a new GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Get the nonce size
	nonceSize := aesGCM.NonceSize()
	if len(file) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract the nonce from the ciphertext
	nonce, ciphertext := file[:nonceSize], file[nonceSize:]

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

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

// prod version
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
	ship := req.Ship
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
	var data []byte
	var err error
	switch req.Source {
	case RestoreBackupSourceRemote:
		data, err = retrieveRemoteBackup(ship, req.Timestamp, req.MD5Hash)
		if err != nil {
			return fmt.Errorf("failed to retrieve remote backup: %w", err)
		}
		// Create restore directory if it doesn't exist
		restoreDir := filepath.Join("/opt/nativeplanet/groundseg/restore", ship)
		if err := os.MkdirAll(restoreDir, 0755); err != nil {
			return fmt.Errorf("failed to create restore directory: %w", err)
		}
		// Write backup to file
		err = os.WriteFile(filepath.Join(restoreDir, strconv.Itoa(req.Timestamp)), data, 0644)
		if err != nil {
			return fmt.Errorf("failed to write backup to file: %w", err)
		}
	case RestoreBackupSourceLocal:
		data, err = retrieveLocalBackup(ship, req.Timestamp, req.LocalBackupType)
		if err != nil {
			return fmt.Errorf("failed to retrieve local backup: %w", err)
		}
	default:
		return fmt.Errorf("unsupported restore source: %s", req.Source)
	}
	err = writeBackupToVolume(ship, data)
	if err != nil {
		return fmt.Errorf("failed to write backup to volume: %w", err)
	}
	// commit the base desk
	err = click.CommitDesk(ship, "base")
	if err != nil {
		return fmt.Errorf("failed to commit desk: %w", err)
	}
	err = click.RestoreTlon(ship)
	if err != nil {
		return fmt.Errorf("failed to restore tlon: %w", err)
	}
	zap.L().Info(fmt.Sprintf("Successfully restored backup for %s", ship))
	return nil
}

func retrieveRemoteBackup(ship string, timestamp int, md5hash string) ([]byte, error) {
	settings := defaultConfigService.StartramSettingsSnapshot()
	link, err := GetBackup(ship, strconv.Itoa(timestamp), settings.RemoteBackupPassword, settings.Pubkey, settings.EndpointURL)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to get backup: %w", err)
	}
	if link == "" {
		return []byte{}, fmt.Errorf("backup link is empty")
	}
	data, err := downloadAndVerify(link, md5hash)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to download and verify backup: %w", err)
	}
	// decrypt the file
	decryptedData, err := decryptFile(data, settings.RemoteBackupPassword)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to decrypt backup: %w", err)
	}
	return decryptedData, nil
}

func writeBackupToVolume(ship string, data []byte) error {
	// Get the Docker volume location for the ship
	cmd := exec.Command("docker", "inspect", "-f", "{{ range .Mounts }}{{ if eq .Type \"volume\" }}{{ .Source }}{{ end }}{{ end }}", ship)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get Docker volume location: %w", err)
	}

	volumePath := strings.TrimSpace(string(output))
	if volumePath == "" {
		return fmt.Errorf("no Docker volume found for container %s", ship)
	}
	deskDir := filepath.Join(volumePath, ship, "base")
	marDir := filepath.Join(deskDir, "mar")
	bakDir := filepath.Join(deskDir, "bak")

	// Create mar directory if it doesn't exist
	if _, err := os.Stat(marDir); os.IsNotExist(err) {
		err = os.MkdirAll(marDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create mar directory: %w", err)
		}
	}

	// Create backup directory if it doesn't exist
	if _, err := os.Stat(bakDir); os.IsNotExist(err) {
		err = os.MkdirAll(bakDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create backup directory: %w", err)
		}
	}

	// Create a temporary directory to decompress the backup
	tmpDir, err := os.MkdirTemp("", "backup-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Decompress the zstd data
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create zstd decoder: %w", err)
	}
	defer decoder.Close()

	// Create a tar reader
	tr := tar.NewReader(decoder)

	// Extract the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		target := filepath.Join(bakDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", target, err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file %s: %v", target, err)
			}
			f.Close()
		}
	}
	return nil
}

func retrieveLocalBackup(ship string, timestamp int, bakType string) ([]byte, error) {
	setBackupDir := func() string {
		basePath := defaultConfigService.BasePath()
		mmc, _ := system.IsMountedMMC(basePath)
		if mmc {
			return "/media/data/backup"
		} else {
			return filepath.Join(basePath, "backup")
		}
	}
	backupDir := setBackupDir()
	backupFile := filepath.Join(backupDir, ship, bakType, strconv.Itoa(timestamp))
	zap.L().Info(fmt.Sprintf("Restoring local backup for %s at %d to %s", ship, timestamp, backupFile))
	file, err := os.Stat(backupFile)
	if err != nil {
		return []byte{}, fmt.Errorf("backup file does not exist: %s", backupFile)
	}
	if file.IsDir() {
		return []byte{}, fmt.Errorf("backup is a directory: %s", backupFile)
	}
	// read the file
	data, err := os.ReadFile(backupFile)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read backup file: %w", err)
	}
	err = click.MountDesk(ship, "base")
	if err != nil {
		return []byte{}, fmt.Errorf("failed to mount base desk: %w", err)
	}
	return data, nil
}

// dev version
func restoreBackupDev(ship string) error {
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
	settings := defaultConfigService.StartramSettingsSnapshot()
	res, err := Retrieve()
	if err != nil {
		return fmt.Errorf("failed to retrieve StarTram information: %w", err)
	}
	for _, backup := range res.Backups {
		data, exists := backup[ship]
		if !exists {
			continue
		}
		var highestTimestamp int
		var highestMD5 string
		for _, item := range data {
			if item.Timestamp > highestTimestamp {
				highestTimestamp = item.Timestamp
				highestMD5 = item.MD5
			}
		}
		if highestTimestamp > 0 {
			link, err := GetBackup(ship, strconv.Itoa(highestTimestamp), settings.RemoteBackupPassword, settings.Pubkey, settings.EndpointURL)
			if err != nil {
				return fmt.Errorf("failed to get backup: %w", err)
			}
			if link == "" {
				return fmt.Errorf("backup link is empty")
			}
			data, err := downloadAndVerify(link, highestMD5)
			if err != nil {
				return fmt.Errorf("failed to download and verify backup: %w", err)
			}
			// decrypt the file
			decryptedData, err := decryptFile(data, settings.RemoteBackupPassword)
			if err != nil {
				return fmt.Errorf("failed to decrypt backup: %w", err)
			}
			// write to appropriate location
			loc := filepath.Join("/opt/nativeplanet/groundseg/restore", ship)
			// create the directory if it doesn't exist
			if _, err := os.Stat(loc); os.IsNotExist(err) {
				os.MkdirAll(loc, 0755)
			}
			err = os.WriteFile(filepath.Join(loc, strconv.Itoa(highestTimestamp)), decryptedData, 0644)
			if err != nil {
				return fmt.Errorf("failed to write backup to file: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("no backup found for %s", ship)
}

func downloadAndVerify(link, md5hash string) ([]byte, error) {
	// Download the file
	resp, err := apiGet(link)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Verify the MD5
	computedMD5 := fmt.Sprintf("%x", md5.Sum(data))
	if computedMD5 != md5hash {
		return nil, fmt.Errorf("MD5 mismatch: expected %s, got %s", md5hash, computedMD5)
	}

	// Return the file as bytes
	return data, nil
}
