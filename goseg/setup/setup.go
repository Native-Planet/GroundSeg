package setup

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/config"
	"goseg/logger"
	"goseg/startram"
	"goseg/structs"
)

var (
	// stage and page are the same thing
	Stages = map[string]int{
		"start":    0,
		"profile":  1,
		"startram": 2,
		"complete": 3,
	}
)

/*
   "type":"setup",
   "action":"begin/password/startram/skip",
   "password": "this is only for the password action, this is a str",
   "key": "startram reg key, for startram action",
   "region":"also startram"
*/

func Setup(msg []byte, conn *structs.MuConn, token map[string]string) error {
	var setupPayload structs.WsSetupPayload
	logger.Logger.Info("Setup")
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
				"setup":  "startram",
				"pwHash": hashed,
			}); err != nil {
				return fmt.Errorf("Unable to set password: %v", err)
			}
		case "startram":
			key := setupPayload.Payload.Key
			region := setupPayload.Payload.Region
			if err = startram.Register(key, region); err != nil {
				return fmt.Errorf("Failed registration: %v", err)
			}
			if err = config.UpdateConf(map[string]interface{}{
				"setup": "complete",
			}); err != nil {
				return fmt.Errorf("Unable to update config: %v", err)
			}
			if err := auth.AddToAuthMap(conn.Conn, token, true); err != nil {
				return fmt.Errorf("Error moving session to auth: %v", err)
			}
		case "skip":
			if err = config.UpdateConf(map[string]interface{}{
				"setup": "complete",
			}); err != nil {
				return fmt.Errorf("Unable to update config: %v", err)
			}
			if err := auth.AddToAuthMap(conn.Conn, token, true); err != nil {
				return fmt.Errorf("Error moving session to auth: %v", err)
			}
		default:
			return fmt.Errorf("Invalid setup action: %v", setupPayload.Payload.Action)
		}
		return nil
	}
}
