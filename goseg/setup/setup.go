package setup

import (
	"encoding/json"
	"fmt"
	"groundseg/auth/tokens"
	"groundseg/authsession"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"

	"go.uber.org/zap"
)

var (
	// stage and page are the same thing
	Stages = map[string]int{
		"start":    0,
		"profile":  1,
		"startram": 2,
		"complete": 3,
	}

	updateConfTypedForSetup  = config.UpdateConfigTyped
	hasherForSetup           = tokens.Hasher
	cycleWgKeyForSetup       = config.CycleWgKey
	startramRegisterForSetup = startram.Register
	addToAuthMapForSetup     = authsession.AddToAuthMap
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
	zap.L().Info("Setup")
	err := json.Unmarshal(msg, &setupPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal setup payload: %w", err)
	}
	for {
		switch setupPayload.Payload.Action {
		case "begin":
			if err = updateConfTypedForSetup(config.WithSetup("profile")); err != nil {
				return fmt.Errorf("Unable to begin profile setup: %w", err)
			}
		case "password":
			password := setupPayload.Payload.Password
			hashed := hasherForSetup(password)
			if err = updateConfTypedForSetup(
				config.WithSetup("startram"),
				config.WithPwHash(hashed),
			); err != nil {
				return fmt.Errorf("Unable to set password: %w", err)
			}
		case "startram":
			if err := cycleWgKeyForSetup(); err != nil {
				return fmt.Errorf("Failed to reset registration key: %w", err)
			}
			key := setupPayload.Payload.Key
			region := setupPayload.Payload.Region
			if err = startramRegisterForSetup(key, region); err != nil {
				return fmt.Errorf("Failed registration: %w", err)
			}
			if err = updateConfTypedForSetup(config.WithSetup("complete")); err != nil {
				return fmt.Errorf("Unable to update config: %w", err)
			}
			if err := addToAuthMapForSetup(conn.Conn, token, true); err != nil {
				return fmt.Errorf("Error moving session to auth: %w", err)
			}
		case "skip":
			if err = updateConfTypedForSetup(config.WithSetup("complete")); err != nil {
				return fmt.Errorf("Unable to update config: %w", err)
			}
			if err := addToAuthMapForSetup(conn.Conn, token, true); err != nil {
				return fmt.Errorf("Error moving session to auth: %w", err)
			}
		default:
			return fmt.Errorf("Invalid setup action: %v", setupPayload.Payload.Action)
		}
		return nil
	}
}
