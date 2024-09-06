package startram

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"crypto/rand"
	"groundseg/config"
	"groundseg/structs"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"time"

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
func GetBackup(ship, timestamp, privateKey string) (string, error) {
	conf := config.Conf()
	reqData := structs.GetBackupRequest{
		Ship:      ship,
		Pubkey:    conf.Pubkey,
		Timestamp: timestamp,
	}
	url := "https://" + conf.EndpointUrl + "/v1/backup/get"
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
	if fmt.Sprintf("%v", &getBackupResp.Result) == "" {
		return "", fmt.Errorf("returned empty backup URL")
	}
	resp, err = http.Get(fmt.Sprintf("%v", &getBackupResp.Result))
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file: server returned %d", resp.StatusCode)
	}
	out, err := os.Create(timestamp)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}
	outputFile, err := decryptBackup(timestamp, privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt backup: %w", err)
	}
	return outputFile, nil
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

// decrypt for encryptBackup
func decryptBackup(filename string, keyString string) (string, error) {
	key := sha256.Sum256([]byte(keyString)) // Derive a 256-bit key from the provided string
	ciphertext, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	output := fmt.Sprintf("%s.dec", filename)
	if err = os.WriteFile(output, plaintext, 0644); err != nil {
		return "", err
	}
	return output, nil
}

func RestoreBackup(ship string) error {
	zap.L().Info(fmt.Sprintf("Restoring backup for %s", ship))
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
			zap.L().Info(fmt.Sprintf("Latest backup for %s: Timestamp: %d, MD5: %s", ship, highestTimestamp, highestMD5))
			// download the backup
			return nil
		}
	}
	return fmt.Errorf("no backup found for %s", ship)
}