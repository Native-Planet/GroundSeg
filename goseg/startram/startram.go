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
	"groundseg/config"
	"groundseg/structs"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
)

var (
	EventBus = make(chan structs.Event, 100)
	Regions  = make(map[string]structs.StartramRegion)
)

// get available regions from endpoint
func GetRegions() (map[string]structs.StartramRegion, error) {
	var regions map[string]structs.StartramRegion
	conf := config.Conf()
	regionUrl := "https://" + conf.EndpointUrl + "/v1/regions"
	resp, err := http.Get(regionUrl)
	if err != nil {
		errmsg := maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err))
		zap.L().Warn(errmsg)
		return regions, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		errmsg := fmt.Sprintf("Error reading regions info: %v", err)
		zap.L().Warn(errmsg)
		return regions, err
	}
	// unmarshal values into struct
	err = json.Unmarshal(body, &regions)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling regions json: %v", err)
		fmt.Println(string(body))
		zap.L().Warn(errmsg)
		return regions, err
	}
	Regions = regions
	return regions, nil
}

// retrieve the reg info for the local pubkey
func Retrieve() (structs.StartramRetrieve, error) {
	var retrieve structs.StartramRetrieve
	conf := config.Conf()
	regionUrl := "https://" + conf.EndpointUrl + "/v1/retrieve?pubkey=" + conf.Pubkey
	resp, err := http.Get(regionUrl)
	if err != nil {
		errmsg := maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err))
		zap.L().Warn(errmsg)
		return retrieve, err
	}
	// read response body
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		errmsg := fmt.Sprintf("Error reading retrieve info: %v", err)
		zap.L().Warn(errmsg)
		return retrieve, err
	}
	// unmarshal values into struct
	err = json.Unmarshal(body, &retrieve)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling retrieve json: %v", err)
		//fmt.Println(string(body))
		zap.L().Warn(errmsg)
		return retrieve, err
	}
	regStatus := true
	if retrieve.Status != "No record" {
		// pin that ho to the global vars
		config.StartramConfig = retrieve
		zap.L().Info(fmt.Sprintf("StarTram info retrieved"))
		zap.L().Debug(fmt.Sprintf("StarTram info: %s", string(body)))
	} else {
		regStatus = false
		return retrieve, fmt.Errorf(fmt.Sprintf("No registration record"))
	}
	if conf.WgRegistered != regStatus {
		zap.L().Info("Updating registration status")
		err = config.UpdateConf(map[string]interface{}{
			"wgRegistered": regStatus,
		})
		if err != nil {
			zap.L().Error(fmt.Sprintf("%v", err))
		}
	}
	err = fmt.Errorf("No registration")
	if regStatus {
		err = nil
		EventBus <- structs.Event{Type: "retrieve", Data: nil}
	}
	return retrieve, err
}

// register your pubkey
func Register(regCode string, region string) error {
	zap.L().Info(fmt.Sprintf("Submitting registration in %s", region))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/register"
	var regObj structs.StartramRegister
	var respObj structs.StartramRegisterResp
	regObj.Pubkey = conf.Pubkey
	regObj.RegCode = regCode
	regObj.Region = region
	regJSON, err := json.Marshal(regObj)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't marshal registration: %v", err))
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(regJSON))
	if err != nil {
		return fmt.Errorf(maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err)))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error reading response: %v", err))
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf(fmt.Sprintf("Error unmarshalling response: %v", err))
	}
	if respObj.Error == 0 {
		err = config.UpdateConf(map[string]interface{}{
			"wgRegistered": true,
		})
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error updating registration status: %v", err))
		}
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error retrieving post-registration: %v", err))
		}
	} else {
		err = config.UpdateConf(map[string]interface{}{
			"wgRegistered": false,
		})
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error updating registration status: %v", err))
		}
		return fmt.Errorf(fmt.Sprintf("Error registering at %s: %v", url, respObj.Debug))
	}
	return nil
}

// create a service
func SvcCreate(subdomain string, svcType string) error {
	zap.L().Info(fmt.Sprintf("Creating new %s registrations: %s", svcType, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create"
	var createObj structs.StartramSvc
	var respObj structs.StartramSvcResp
	createObj.Pubkey = conf.Pubkey
	createObj.Subdomain = subdomain
	createObj.SvcType = svcType
	createJSON, err := json.Marshal(createObj)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't marshal registration: %v", err))
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(createJSON))
	if err != nil {
		return fmt.Errorf(maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err)))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error reading response: %v", err))
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf(fmt.Sprintf("Error unmarshalling response: %v", err))
	}
	if respObj.Error == 0 {
		// _, err := Retrieve()
		// if err != nil {
		// 	return fmt.Errorf("Error retrieving post-registration: %v", err)
		// } // this can cause some fucked up infinite loops
		zap.L().Info(fmt.Sprintf("Service %v created", subdomain))
	} else {
		return fmt.Errorf(fmt.Sprintf("Error creating %v: %v", subdomain, respObj.Debug))
	}
	return nil
}

// delete a service
func SvcDelete(subdomain string, svcType string) error {
	zap.L().Info(fmt.Sprintf("Deleting %s registration: %s", svcType, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/delete"
	var delObj structs.StartramSvc
	var respObj structs.StartramSvcResp
	delObj.Pubkey = conf.Pubkey
	delObj.Subdomain = subdomain
	delObj.SvcType = svcType
	delJSON, err := json.Marshal(delObj)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't marshal registration: %v", err))
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(delJSON))
	if err != nil {
		return fmt.Errorf(maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err)))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error reading response: %v", err))
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf(fmt.Sprintf("Error unmarshalling response: %v", err))
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error retrieving post-deletion: %v", err))
		}
	} else {
		return fmt.Errorf(fmt.Sprintf("Error deleting %s: %v", subdomain, respObj.Debug))
	}
	return nil
}

// create a custom domain
func AliasCreate(subdomain string, alias string) error {
	zap.L().Info(fmt.Sprintf("Registering alias %s for %s", alias, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create/alias"
	var aliasObj structs.StartramAlias
	var respObj structs.StartramAliasResp
	aliasObj.Pubkey = conf.Pubkey
	aliasObj.Subdomain = subdomain
	aliasObj.Alias = alias
	aliasJSON, err := json.Marshal(aliasObj)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't marshal registration: %v", err))
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(aliasJSON))
	if err != nil {
		return fmt.Errorf(maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err)))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error reading response: %v", err))
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf(fmt.Sprintf("Error unmarshalling response: %v", err))
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error retrieving post-creation: %v", err))
		}
	} else {
		return fmt.Errorf(fmt.Sprintf("Error aliasing %s: %v", alias, respObj.Debug))
	}
	return nil
}

// delete a custom domain
func AliasDelete(subdomain string, alias string) error {
	zap.L().Info(fmt.Sprintf("Deleting alias %s for %s", alias, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create/alias"
	var delAliasObj structs.StartramAlias
	var respObj structs.StartramAliasResp
	delAliasObj.Pubkey = conf.Pubkey
	delAliasObj.Subdomain = subdomain
	delAliasObj.Alias = alias
	delAliasJSON, err := json.Marshal(delAliasObj)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't marshal alias deletion: %v", err))
	}
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(delAliasJSON))
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Unable to create request: %v", err))
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf(maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err)))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error reading response: %v", err))
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf(fmt.Sprintf("Error unmarshalling response: %v", err))
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf(fmt.Sprintf("Error retrieving post-deletion: %v", err))
		}
	} else {
		return fmt.Errorf(fmt.Sprintf("Error deleting alias %s: %v", alias, respObj.Debug))
	}
	return nil
}

// call registration endpoint for 5 minutes or until all services are "ok"
func backoffRetrieve() error {
	startTime := time.Now()
	duration := 5 * time.Second
	for {
		res, err := Retrieve()
		if err != nil {
			return err
		}
		// return if all services are registered
		for _, remote := range res.Subdomains {
			if remote.Status != "ok" {
				zap.L().Warn(fmt.Sprintf("backoff: %v %v", remote.URL, remote.Status))
				break
			}
			// all "ok"
			return nil
		}
		// timeout after 5min
		if time.Since(startTime) > 5*time.Minute {
			return fmt.Errorf("Registration retrieval timed out")
		}
		// linear cooldown
		zap.L().Warn(fmt.Sprintf("%v", duration))
		time.Sleep(duration)
		if duration.Seconds() < 60 {
			duration = time.Duration(math.Min(duration.Seconds()*2, 60)) * time.Second
		} else {
			duration += 60 * time.Second
		}
	}
}

// submit existing ships on registration
func RegisterExistingShips() error {
	conf := config.Conf()
	if conf.WgRegistered {
		for _, ship := range conf.Piers {
			if err := SvcCreate(ship, "urbit"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't register pier: %v: %v", ship, err))
				continue
			}
			if err := SvcCreate("s3."+ship, "minio"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't register S3: %v: %v", ship, err))
			}
		}
		if err := backoffRetrieve(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Instance is not registered")
	}
	zap.L().Info("Registration retrieved")
	return nil
}

func RegisterNewShip(ship string) error {
	zap.L().Info(fmt.Sprintf("Registering service for new ship: %s", ship))
	if err := SvcCreate(ship, "urbit"); err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't register pier: %v: %v", ship, err))
	}
	if err := SvcCreate("s3."+ship, "minio"); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't register S3: %v: %v", ship, err))
	}
	if err := backoffRetrieve(); err != nil {
		return err
	}
	return nil
}

// cancel a startram subscription with reg code
func CancelSub(key string) error {
	zap.L().Info(fmt.Sprintf("Cancelling StarTram registration"))
	conf := config.Conf()
	var respObj structs.CancelStartramSub
	url := "https://" + conf.EndpointUrl + "/v1/stripe/cancel"
	cancelObj := map[string]interface{}{
		"reg_key": key,
	}
	cancelJSON, err := json.Marshal(cancelObj)
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Couldn't marshal registration: %v", err))
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(cancelJSON))
	if err != nil {
		return fmt.Errorf(maskPubkey(fmt.Sprintf("Unable to connect to API server: %v", err)))
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("Error reading response: %v", err))
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf(fmt.Sprintf("Error unmarshalling response: %v", err))
	}
	if respObj.Error == 1 {
		return fmt.Errorf(fmt.Sprintf("Couldn't cancel subscription: %v", &respObj.Message))
	}
	return nil
}

func maskPubkey(input string) string {
	// Regular expression pattern to match text between "pubkey=" and "0K", including letters and numbers
	re := regexp.MustCompile(`(?s)(pubkey=)[a-zA-Z0-9]+(0K)`)

	// Replace the matched text with the same prefix and suffix, and "x" for each letter or number in between
	output := re.ReplaceAllStringFunc(input, func(s string) string {
		// Extract the prefix "pubkey=" and suffix "0K"
		prefix := "pubkey="
		suffix := "0K"

		// Get the length of the part to be replaced with "x"
		length := len(s) - len(prefix) - len(suffix)

		// Create the replacement string with "x" for each character
		replacement := prefix + string(make([]rune, length, length)) + suffix

		// Replace all characters in between with "x"
		for i := 0; i < length; i++ {
			replacement = replacement[:len(prefix)+i] + "x" + replacement[len(prefix)+i+1:]
		}

		return replacement
	})

	return output
}

// get a backup download URL for a specific ship and backup timestamp from retrieve blob
func GetBackup(ship, timestamp, backupPassword, pubkey, endpointUrl string) (string, error) {
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
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
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
func UploadBackup(ship, privateKey, filePath string) error {
	zap.L().Info(fmt.Sprintf("Uploading backup for %s", ship))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/backup/upload"
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
	if err := writer.WriteField("pubkey", conf.Pubkey); err != nil {
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

// prod version
func RestoreBackup(ship string, remote bool, timestamp int, md5hash string, dev bool) error {
	if dev {
		return restoreBackupDev(ship)
	}
	return restoreBackupProd(ship, remote, timestamp, md5hash)
}

func restoreBackupProd(ship string, remote bool, timestamp int, md5hash string) error {
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
	var data []byte
	var err error
	if remote {
		data, err = retrieveRemoteBackup(ship, timestamp, md5hash)
		if err != nil {
			return fmt.Errorf("failed to retrieve remote backup: %w", err)
		}
		// Create restore directory if it doesn't exist
		restoreDir := filepath.Join("/opt/nativeplanet/groundseg/restore", ship)
		if err := os.MkdirAll(restoreDir, 0755); err != nil {
			return fmt.Errorf("failed to create restore directory: %w", err)
		}
		// Write backup to file
		err = os.WriteFile(filepath.Join(restoreDir, strconv.Itoa(timestamp)), data, 0644)
		if err != nil {
			return fmt.Errorf("failed to write backup to file: %w", err)
		}
	} else {
		// local restore
		data, err = retrieveLocalBackup(ship, timestamp)
		if err != nil {
			return fmt.Errorf("failed to retrieve local backup: %w", err)
		}
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
	conf := config.Conf()
	link, err := GetBackup(ship, strconv.Itoa(timestamp), conf.RemoteBackupPassword, conf.Pubkey, conf.EndpointUrl)
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
	decryptedData, err := decryptFile(data, conf.RemoteBackupPassword)
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

	// 	// Always write egg-any.hoon file
	// 	eggAnyPath := filepath.Join(marDir, "egg-any.hoon")
	// 	eggAnyContent := `|_  =egg-any:gall
	// ++  grad  %noun
	// ++  grow
	//   |%
	//   ++  noun  egg-any
	//   --
	// ++  grab
	//   |%
	//   ++  noun  egg-any:gall
	//   --
	// --`
	// 	err = os.WriteFile(eggAnyPath, []byte(eggAnyContent), 0644)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to create egg-any.hoon: %w", err)
	// 	}

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
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			f.Close()
		}
	}
	return nil
}

func retrieveLocalBackup(ship string, timestamp int) ([]byte, error) {
	setBackupDir := func() string {
		mmc, _ := isMountedMMC(config.BasePath)
		if mmc {
			return "/media/data/backup"
		} else {
			return filepath.Join(config.BasePath, "backup")
		}
	}
	backupDir := setBackupDir()
	backupFile := filepath.Join(backupDir, ship, strconv.Itoa(timestamp))
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

func isMountedMMC(dirPath string) (bool, error) {
	partitions, err := disk.Partitions(true)
	if err != nil {
		return false, fmt.Errorf("failed to get list of partitions")
	}
	/*
		the outer loop loops from child up the unix path
		until a mountpoint is found
	*/
OuterLoop:
	for {
		for _, p := range partitions {
			if p.Mountpoint == dirPath {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return true, nil
				} else {
					break OuterLoop
				}
			}
		}
		if dirPath == "/" {
			break
		}
		dirPath = path.Dir(dirPath) // Reduce the path by one level
	}
	return false, nil
}

// dev version
func restoreBackupDev(ship string) error {
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
	conf := config.Conf()
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
			link, err := GetBackup(ship, strconv.Itoa(highestTimestamp), conf.RemoteBackupPassword, conf.Pubkey, conf.EndpointUrl)
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
			decryptedData, err := decryptFile(data, conf.RemoteBackupPassword)
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
	resp, err := http.Get(link)
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
