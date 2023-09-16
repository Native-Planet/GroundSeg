package startram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"io/ioutil"
	"math"
	"net/http"
	"time"
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
		errmsg := fmt.Sprintf("Unable to connect to API server: %v", err)
		logger.Logger.Warn(errmsg)
		return regions, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		errmsg := fmt.Sprintf("Error reading regions info: %v", err)
		logger.Logger.Warn(errmsg)
		return regions, err
	}
	// unmarshal values into struct
	err = json.Unmarshal(body, &regions)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling regions json: %v", err)
		fmt.Println(string(body))
		logger.Logger.Warn(errmsg)
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
		errmsg := fmt.Sprintf("Unable to connect to API server: %v", err)
		logger.Logger.Warn(errmsg)
		return retrieve, err
	}
	// read response body
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		errmsg := fmt.Sprintf("Error reading retrieve info: %v", err)
		logger.Logger.Warn(errmsg)
		return retrieve, err
	}
	// unmarshal values into struct
	err = json.Unmarshal(body, &retrieve)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling retrieve json: %v", err)
		fmt.Println(string(body))
		logger.Logger.Warn(errmsg)
		return retrieve, err
	}
	regStatus := true
	if retrieve.Status != "No record" {
		// pin that ho to the global vars
		config.StartramConfig = retrieve
		s := "<hidden>"
		if config.DebugMode {
			s = string(body)
		}
		logger.Logger.Info(fmt.Sprintf("StarTram info retrieved: %s", s))
	} else {
		regStatus = false
		return retrieve, fmt.Errorf("No registration record")
	}
	if conf.WgRegistered != regStatus {
		logger.Logger.Info("Updating registration status")
		err = config.UpdateConf(map[string]interface{}{
			"wgRegistered": regStatus,
		})
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("%v", err))
		}
	}
	err = fmt.Errorf("No registration")
	if regStatus {
		err = nil
	}
	EventBus <- structs.Event{Type: "retrieve", Data: nil}
	return retrieve, err
}

// register your pubkey
func Register(regCode string, region string) error {
	logger.Logger.Info(fmt.Sprintf("Submitting registration in %s", region))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/register"
	var regObj structs.StartramRegister
	var respObj structs.StartramRegisterResp
	regObj.Pubkey = conf.Pubkey
	regObj.RegCode = regCode
	regObj.Region = region
	regJSON, err := json.Marshal(regObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(regJSON))
	if err != nil {
		return fmt.Errorf("Unable to connect to API server: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf("Error unmarshalling response: %v", err)
	}
	if respObj.Error == 0 {
		err = config.UpdateConf(map[string]interface{}{
			"wgRegistered": true,
		})
		if err != nil {
			return fmt.Errorf("Error updating registration status: %v", err)
		}
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-registration: %v", err)
		}
	} else {
		err = config.UpdateConf(map[string]interface{}{
			"wgRegistered": false,
		})
		if err != nil {
			return fmt.Errorf("Error updating registration status: %v", err)
		}
		return fmt.Errorf("Error registering at %s: %v", url, respObj.Debug)
	}
	return nil
}

// create a service
func SvcCreate(subdomain string, svcType string) error {
	logger.Logger.Info(fmt.Sprintf("Creating new %s registrations: %s", svcType, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create"
	var createObj structs.StartramSvc
	var respObj structs.StartramSvcResp
	createObj.Pubkey = conf.Pubkey
	createObj.Subdomain = subdomain
	createObj.SvcType = svcType
	createJSON, err := json.Marshal(createObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(createJSON))
	if err != nil {
		return fmt.Errorf("Unable to connect to API server: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf("Error unmarshalling response: %v", err)
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-registration: %v", err)
		}
	} else {
		return fmt.Errorf("Error creating %s: %v", subdomain, respObj.Debug)
	}
	return nil
}

// delete a service
func SvcDelete(subdomain string, svcType string) error {
	logger.Logger.Info(fmt.Sprintf("Deleting %s registration: %s", svcType, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create"
	var delObj structs.StartramSvc
	var respObj structs.StartramSvcResp
	delObj.Pubkey = conf.Pubkey
	delObj.Subdomain = subdomain
	delObj.SvcType = svcType
	delJSON, err := json.Marshal(delObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %v", err)
	}
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(delJSON))
	if err != nil {
		return fmt.Errorf("Unable to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to connect to API server: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf("Error unmarshalling response: %v", err)
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-deletion: %v", err)
		}
	} else {
		return fmt.Errorf("Error deleting %s: %v", subdomain, respObj.Debug)
	}
	return nil
}

// create a custom domain
func AliasCreate(subdomain string, alias string) error {
	logger.Logger.Info(fmt.Sprintf("Registering alias %s for %s", alias, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create/alias"
	var aliasObj structs.StartramAlias
	var respObj structs.StartramAliasResp
	aliasObj.Pubkey = conf.Pubkey
	aliasObj.Subdomain = subdomain
	aliasObj.Alias = alias
	aliasJSON, err := json.Marshal(aliasObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal registration: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(aliasJSON))
	if err != nil {
		return fmt.Errorf("Unable to connect to API server: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf("Error unmarshalling response: %v", err)
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-creation: %v", err)
		}
	} else {
		return fmt.Errorf("Error aliasing %s: %v", alias, respObj.Debug)
	}
	return nil
}

// delete a custom domain
func AliasDelete(subdomain string, alias string) error {
	logger.Logger.Info(fmt.Sprintf("Deleting alias %s for %s", alias, subdomain))
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/create/alias"
	var delAliasObj structs.StartramAlias
	var respObj structs.StartramAliasResp
	delAliasObj.Pubkey = conf.Pubkey
	delAliasObj.Subdomain = subdomain
	delAliasObj.Alias = alias
	delAliasJSON, err := json.Marshal(delAliasObj)
	if err != nil {
		return fmt.Errorf("Couldn't marshal alias deletion: %v", err)
	}
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", url, bytes.NewBuffer(delAliasJSON))
	if err != nil {
		return fmt.Errorf("Unable to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to connect to API server: %v", err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("Error reading response: %v", err)
	}
	if err = json.Unmarshal(body, &respObj); err != nil {
		return fmt.Errorf("Error unmarshalling response: %v", err)
	}
	if respObj.Error == 0 {
		_, err := Retrieve()
		if err != nil {
			return fmt.Errorf("Error retrieving post-deletion: %v", err)
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
		res, err := Retrieve()
		if err != nil {
			return err
		}
		// return if all services are registered
		for _, remote := range res.Subdomains {
			if remote.Status != "ok" {
				logger.Logger.Warn(fmt.Sprintf("backoff: %v %v", remote.URL, remote.Status))
				break
			}
			// all "ok"
			return nil
		}
		// timeout after 5min
		if time.Since(startTime) > 5*time.Minute {
			errmsg := fmt.Errorf("Registration retrieval timed out")
			logger.Logger.Error(fmt.Sprintf("%v", errmsg))
			return errmsg
		}
		// linear cooldown
		logger.Logger.Warn(fmt.Sprintf("%v", duration))
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
				logger.Logger.Error(fmt.Sprintf("Couldn't register pier: %v: %v", ship, err))
			}
			if err := SvcCreate("s3."+ship, "minio"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't register S3: %v: %v", ship, err))
			}
		}
		if err := backoffRetrieve(); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Instance is not registered")
	}
	logger.Logger.Info("Registration retrieved")
	return nil
}
