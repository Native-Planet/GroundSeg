package startram

import (
	"encoding/json"
	"fmt"
	"goseg/config"
	"goseg/structs"
	"io/ioutil"
	"net/http"
)

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
