package setup

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/config"
	"goseg/startram"
	"goseg/structs"
)

var (
	// stage and page are the same thing
	Stages = map[string]int{
		"start":0,
		"profile":1,
		"startram":2,
		"complete":3,
	}
)

/*
    "type":"setup",
    "action":"begin/password/startram/skip",
    "password": "this is only for the password action, this is a str",
    "key": "startram reg key, for startram action",
    "region":"also startram"
*/

func Setup(msg []byte) error {
	var setupPayload structs.WsSetupPayload
	err := json.Unmarshal(msg, &setupPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal setup payload: %v", err)
	}
	for {
		switch setupPayload.Payload.Action {
		case "begin":
			if err = config.UpdateConf(map[string]interface{}{
				"setup": "profile",
			}); err != nil {
				return fmt.Errorf("Unable to begin profile setup: %v", err)
			}
		case "password":
			password := setupPayload.Payload.Password
			hashed := auth.Hasher(password)
			if err = config.UpdateConf(map[string]interface{}{
				"setup": "startram",
				"password": hashed,
			}); err != nil {
				return fmt.Errorf("Unable to set password: %v", err)
			}
		case "startram":
			key := setupPayload.Payload.Key
			region := setupPayload.Payload.Region
			if err = startram.Register(key,region); err != nil {
				return fmt.Errorf("Failed registration: %v", err)
			}
			if err = config.UpdateConf(map[string]interface{}{
				"setup": "complete",
				"firstBoot": false,
			}); err != nil {
				return fmt.Errorf("Unable to update config: %v", err)
			}
		case "skip":
			if err = config.UpdateConf(map[string]interface{}{
				"setup": "complete",
				"firstBoot": false,
			}); err != nil {
				return fmt.Errorf("Unable to update config: %v", err)
			}
		default:
			return fmt.Errorf("Invalid setup action: %v",setupPayload.Payload.Action)
		}
		return nil
	}
}