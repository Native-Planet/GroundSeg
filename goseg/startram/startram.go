package startram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/structs"
	"io/ioutil"
	"net/http"
)

// get available regions from endpoint
func GetRegions() (map[string]structs.StartramRegion, error) {
	var regions map[string]structs.StartramRegion
	conf := config.Conf()
	regionUrl := "https://" + conf.EndpointUrl + "/v1/regions"
	resp, err := http.Get(regionUrl)
	if err != nil {
		errmsg := fmt.Sprintf("Unable to connect to API server: %v", err)
		config.Logger.Warn(errmsg)
		return regions, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		errmsg := fmt.Sprintf("Error reading regions info: %v", err)
		config.Logger.Warn(errmsg)
		return regions, err
	}
	// unmarshal values into struct
	err = json.Unmarshal(body, &regions)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling regions json: %v", err)
		fmt.Println(string(body))
		config.Logger.Warn(errmsg)
		return regions, err
	}
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
		config.Logger.Warn(errmsg)
		return retrieve, err
	}
	// read response body
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		errmsg := fmt.Sprintf("Error reading retrieve info: %v", err)
		config.Logger.Warn(errmsg)
		return retrieve, err
	}
	// unmarshal values into struct
	err = json.Unmarshal(body, &retrieve)
	if err != nil {
		errmsg := fmt.Sprintf("Error unmarshalling retrieve json: %v", err)
		fmt.Println(string(body))
		config.Logger.Warn(errmsg)
		return retrieve, err
	}
	// pin that ho to the global vars
	config.StartramConfig = retrieve
	config.Logger.Info(fmt.Sprintf("StarTram info retrieved: %s", string(body)))
	return retrieve, nil
}

// register your pubkey
func Register(regCode string) error {
	conf := config.Conf()
	url := "https://" + conf.EndpointUrl + "/v1/register"
	var regObj structs.StartramRegister
	var respObj structs.StartramRegisterResp
	regObj.Pubkey = conf.Pubkey
	regObj.RegCode = regCode
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
		return fmt.Errorf("Error registering: %v", respObj.Debug)
	}
	return nil
}

// create a service
func SvcCreate(subdomain string, svcType string) error {
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
