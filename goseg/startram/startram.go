package startram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"groundseg/httpx"
	"groundseg/structs"
	"math"
	"net/http"
	"regexp"
	"time"

	"go.uber.org/zap"
)

var (
	eventBus = make(chan structs.Event, 100)
)

// FetchRegions gets available regions from endpoint without mutating package state.
func FetchRegions() (map[string]structs.StartramRegion, error) {
	var regions map[string]structs.StartramRegion
	command := buildFetchRegionsCommand()
	regionUrl := "https://" + command.EndpointURL + "/v1/regions"
	resp, err := apiGet(regionUrl)
	if err != nil {
		wrappedErr := wrapAPIConnectionError(err)
		errmsg := wrappedErr.Error()
		zap.L().Warn(errmsg)
		return regions, wrappedErr
	}
	if err := httpx.ReadJSON(resp, regionUrl, &regions); err != nil {
		zap.L().Warn(fmt.Sprintf("Error decoding regions response: %v", err))
		return regions, fmt.Errorf("decode regions response: %w", err)
	}
	return regions, nil
}

// SyncRegions refreshes the package-level region cache.
func SyncRegions() (map[string]structs.StartramRegion, error) {
	regions, err := FetchRegions()
	if err != nil {
		return nil, err
	}
	defaultRegionStore.Set(regions)
	return defaultRegionStore.Snapshot(), nil
}

// GetRegions is kept for compatibility and behaves as a read-only fetch.
func GetRegions() (map[string]structs.StartramRegion, error) {
	return FetchRegions()
}

// Retrieve fetches registration information for the local pubkey without side effects.
func Retrieve() (structs.StartramRetrieve, error) {
	var retrieve structs.StartramRetrieve
	command := buildRetrieveCommand()
	regionUrl := "https://" + command.EndpointURL + "/v1/retrieve?pubkey=" + command.Pubkey
	resp, err := apiGet(regionUrl)
	if err != nil {
		wrappedErr := wrapAPIConnectionError(err)
		errmsg := wrappedErr.Error()
		zap.L().Warn(errmsg)
		return retrieve, wrappedErr
	}
	if err := httpx.ReadJSON(resp, maskPubkey(regionUrl), &retrieve); err != nil {
		zap.L().Warn(fmt.Sprintf("Error decoding retrieve response: %v", err))
		return retrieve, fmt.Errorf("decode retrieve response: %w", err)
	}
	if retrieve.Status != "No record" {
		return retrieve, nil
	}
	return retrieve, fmt.Errorf("No registration record")
}

// SyncRetrieve updates in-memory and persisted registration state using Retrieve().
func SyncRetrieve() (structs.StartramRetrieve, error) {
	retrieve, err := Retrieve()
	if err != nil {
		return retrieve, err
	}
	if err := ApplyRetrieveState(retrieve); err != nil {
		return retrieve, err
	}
	return retrieve, nil
}

// register your pubkey
func register(regCode string, region string) error {
	zap.L().Info(fmt.Sprintf("Submitting registration in %s", region))
	command := buildRegisterCommand(regCode, region)
	url := "https://" + command.EndpointURL + "/v1/register"
	var regObj structs.StartramRegister
	var respObj structs.StartramRegisterResp
	regObj.Pubkey = command.Pubkey
	regObj.RegCode = command.RegCode
	regObj.Region = command.Region
	regJSON, err := json.Marshal(regObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %w", err)
	}
	resp, err := apiPost(url, "application/json", bytes.NewBuffer(regJSON))
	if err != nil {
		return wrapAPIConnectionError(err)
	}
	if err := httpx.ReadJSON(resp, url, &respObj); err != nil {
		return fmt.Errorf("decode registration response: %w", err)
	}
	if respObj.Error == 0 {
		err = defaultConfigService.SetWgRegistered(true)
		if err != nil {
			return fmt.Errorf("Error updating registration status: %w", err)
		}
		_, err := SyncRetrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-registration: %w", err)
		}
	} else {
		err = defaultConfigService.SetWgRegistered(false)
		if err != nil {
			return fmt.Errorf("Error updating registration status: %w", err)
		}
		return fmt.Errorf("Error registering at %s: %v", url, respObj.Debug)
	}
	return nil
}

// create a service
func svcCreate(subdomain string, svcType string) error {
	zap.L().Info(fmt.Sprintf("Creating new %s registrations: %s", svcType, subdomain))
	command := buildServiceCommand(subdomain, svcType)
	url := "https://" + command.EndpointURL + "/v1/create"
	var createObj structs.StartramSvc
	var respObj structs.StartramSvcResp
	createObj.Pubkey = command.Pubkey
	createObj.Subdomain = command.Subdomain
	createObj.SvcType = command.SvcType
	createJSON, err := json.Marshal(createObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %w", err)
	}
	resp, err := apiPost(url, "application/json", bytes.NewBuffer(createJSON))
	if err != nil {
		return wrapAPIConnectionError(err)
	}
	if err := httpx.ReadJSON(resp, url, &respObj); err != nil {
		return fmt.Errorf("decode create response: %w", err)
	}
	if respObj.Error == 0 {
		// _, err := Retrieve()
		// if err != nil {
		// 	return fmt.Errorf("Error retrieving post-registration: %w", err)
		// } // this can cause some fucked up infinite loops
		zap.L().Info(fmt.Sprintf("Service %v created", subdomain))
	} else {
		return fmt.Errorf("Error creating %v: %v", subdomain, respObj.Debug)
	}
	return nil
}

// delete a service
func svcDelete(subdomain string, svcType string) error {
	zap.L().Info(fmt.Sprintf("Deleting %s registration: %s", svcType, subdomain))
	command := buildServiceCommand(subdomain, svcType)
	url := "https://" + command.EndpointURL + "/v1/delete"
	var delObj structs.StartramSvc
	var respObj structs.StartramSvcResp
	delObj.Pubkey = command.Pubkey
	delObj.Subdomain = command.Subdomain
	delObj.SvcType = command.SvcType
	delJSON, err := json.Marshal(delObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %w", err)
	}
	resp, err := apiPost(url, "application/json", bytes.NewBuffer(delJSON))
	if err != nil {
		return wrapAPIConnectionError(err)
	}
	if err := httpx.ReadJSON(resp, url, &respObj); err != nil {
		return fmt.Errorf("decode delete response: %w", err)
	}
	if respObj.Error == 0 {
		_, err := SyncRetrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-deletion: %w", err)
		}
	} else {
		return fmt.Errorf("Error deleting %s: %v", subdomain, respObj.Debug)
	}
	return nil
}

// create a custom domain
func aliasCreate(subdomain string, alias string) error {
	zap.L().Info(fmt.Sprintf("Registering alias %s for %s", alias, subdomain))
	command := buildAliasCommand(subdomain, alias)
	url := "https://" + command.EndpointURL + "/v1/create/alias"
	var aliasObj structs.StartramAlias
	var respObj structs.StartramAliasResp
	aliasObj.Pubkey = command.Pubkey
	aliasObj.Subdomain = command.Subdomain
	aliasObj.Alias = command.Alias
	aliasJSON, err := json.Marshal(aliasObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %w", err)
	}
	resp, err := apiPost(url, "application/json", bytes.NewBuffer(aliasJSON))
	if err != nil {
		return wrapAPIConnectionError(err)
	}
	if err := httpx.ReadJSON(resp, url, &respObj); err != nil {
		return fmt.Errorf("decode alias create response: %w", err)
	}
	if respObj.Error == 0 {
		_, err := SyncRetrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-creation: %w", err)
		}
	} else {
		return fmt.Errorf("Error aliasing %s: %v", alias, respObj.Debug)
	}
	return nil
}

// delete a custom domain
func aliasDelete(subdomain string, alias string) error {
	zap.L().Info(fmt.Sprintf("Deleting alias %s for %s", alias, subdomain))
	command := buildAliasCommand(subdomain, alias)
	url := "https://" + command.EndpointURL + "/v1/create/alias"
	var delAliasObj structs.StartramAlias
	var respObj structs.StartramAliasResp
	delAliasObj.Pubkey = command.Pubkey
	delAliasObj.Subdomain = command.Subdomain
	delAliasObj.Alias = command.Alias
	delAliasJSON, err := json.Marshal(delAliasObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal alias deletion: %w", err)
	}
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(delAliasJSON))
	if err != nil {
		return fmt.Errorf("Unable to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return wrapAPIConnectionError(err)
	}
	if err := httpx.ReadJSON(resp, url, &respObj); err != nil {
		return fmt.Errorf("decode alias delete response: %w", err)
	}
	if respObj.Error == 0 {
		_, err := SyncRetrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-deletion: %w", err)
		}
	} else {
		return fmt.Errorf("Error deleting alias %s: %v", alias, respObj.Debug)
	}
	return nil
}

// call registration endpoint for 5 minutes or until all services are "ok"
func backoffRetrieve() error {
	startTime := time.Now()
	duration := 5 * time.Second
	for {
		res, err := SyncRetrieve()
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
func registerExistingShips() error {
	settings := defaultConfigService.StartramSettingsSnapshot()
	if settings.WgRegistered {
		for _, ship := range settings.Piers {
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

func registerNewShip(ship string) error {
	zap.L().Info(fmt.Sprintf("Registering service for new ship: %s", ship))
	if err := SvcCreate(ship, "urbit"); err != nil {
		return fmt.Errorf("couldn't register pier %s: %w", ship, err)
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
	command := buildCancelSubscriptionCommand(key)
	var respObj structs.CancelStartramSub
	url := "https://" + command.EndpointURL + "/v1/stripe/cancel"
	cancelObj := map[string]interface{}{
		"reg_key": command.RegKey,
	}
	cancelJSON, err := json.Marshal(cancelObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %w", err)
	}
	resp, err := apiPost(url, "application/json", bytes.NewBuffer(cancelJSON))
	if err != nil {
		return wrapAPIConnectionError(err)
	}
	if err := httpx.ReadJSON(resp, url, &respObj); err != nil {
		return fmt.Errorf("decode cancel subscription response: %w", err)
	}
	if respObj.Error == 1 {
		return fmt.Errorf("Couldn't cancel subscription: %v", &respObj.Message)
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
