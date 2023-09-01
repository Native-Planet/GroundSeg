package handler

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"
	"goseg/startram"
	"goseg/structs"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// for passing mutexes between packages
// not being used for now
type HandlerMuConn struct {
    *structs.MuConn
}

func (mc *HandlerMuConn) Write(messageType int, data []byte) error {
	mc.Mu.Lock()
	defer mc.Mu.Unlock()
	return mc.Conn.WriteMessage(messageType, data)
}
// 


// handle bug report stuff
func SupportHandler(msg []byte, payload structs.WsPayload, r *http.Request, conn *websocket.Conn) error {
	config.Logger.Info("Support")
	return nil
}

// handle system events
func SystemHandler(msg []byte) error {
	config.Logger.Info("System")
	var systemPayload structs.WsSystemPayload
	err := json.Unmarshal(msg, &systemPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal system payload: %v", err)
	}
	switch systemPayload.Payload.Action {
	case "power":
		switch systemPayload.Payload.Command {
		case "shutdown":
			config.Logger.Info(fmt.Sprintf("Device shutdown requested"))
			os.Exit(0) // todo: shutdown here

		case "restart":
			config.Logger.Info(fmt.Sprintf("Device restart requested"))
			os.Exit(0) // todo: restart here
		default:
			return fmt.Errorf("Unrecognized power command: %v", systemPayload.Payload.Command)
		}
	default:
		return fmt.Errorf("Unrecognized system action: %v", systemPayload.Payload.Action)
	}
	return nil
}

// handle urbit-type events
func UrbitHandler(msg []byte) error {
	config.Logger.Info("Urbit")
	var urbitPayload structs.WsUrbitPayload
	err := json.Unmarshal(msg, &urbitPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal urbit payload: %v", err)
	}
	patp := urbitPayload.Payload.Patp
	shipConf := config.UrbitConf(patp)
	switch urbitPayload.Payload.Action {
	case "toggle-network":
		currentNetwork := shipConf.Network
		conf := config.Conf()
		if currentNetwork == "wireguard" {
			shipConf.Network = "bridge"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
		} else if currentNetwork == "bridge" && conf.WgRegistered == true {
			shipConf.Network = "wireguard"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			if err := broadcast.BroadcastToClients(); err != nil {
				config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
			}
		} else {
			return fmt.Errorf("No remote registration")
		}
		if shipConf.BootStatus == "boot" {
			docker.StartContainer(patp, "vere")
		}
		return nil
	case "toggle-devmode":
		if shipConf.DevMode == true {
			shipConf.DevMode = false
		} else {
			shipConf.DevMode = true
		}
		update := make(map[string]structs.UrbitDocker)
		update[patp] = shipConf
		if err := config.UpdateUrbitConfig(update); err != nil {
			return fmt.Errorf("Couldn't update urbit config: %v", err)
		}
		if err := broadcast.BroadcastToClients(); err != nil {
			config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
		}
		docker.StartContainer(patp, "vere")
		return nil
	case "toggle-power":
		update := make(map[string]structs.UrbitDocker)
		if shipConf.BootStatus == "noboot" {
			shipConf.BootStatus = "boot"
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: "loading"}
			docker.StartContainer(patp, "vere")
			if err := broadcast.BroadcastToClients(); err != nil {
				config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
			}
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: ""}
		} else if shipConf.BootStatus == "boot" {
			shipConf.BootStatus = "noboot"
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: "loading"}
			docker.StopContainerByName(patp)
			if err := broadcast.BroadcastToClients(); err != nil {
				config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
			}
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: ""}
		}
		return nil
	default:
		return fmt.Errorf("Unrecognized urbit action: %v", urbitPayload.Payload.Type)
	}
	return nil
}

// validate password and add to auth session map
func LoginHandler(conn *structs.MuConn, msg []byte) error {
	// no real mutex here
    // connHandler := &structs.MuConn{Conn: conn}
	var loginPayload structs.WsLoginPayload
	err := json.Unmarshal(msg, &loginPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %v", err)
	}
	isAuthenticated := auth.AuthenticateLogin(loginPayload.Payload.Password)
	if isAuthenticated {
		token := map[string]string{
			"id":    loginPayload.Token.ID,
			"token": loginPayload.Token.Token,
		}
		if err := auth.AddToAuthMap(conn.Conn, token, true); err != nil {
			return fmt.Errorf("Unable to process login: %v", err)
		}
	} else {
		return fmt.Errorf("Failed auth: %v", loginPayload.Payload.Password)
	}
	config.Logger.Info(fmt.Sprintf("Session %s logged in", loginPayload.Token.ID))
	return nil
}

// take a guess
func LogoutHandler(msg []byte) error {
	var logoutPayload structs.WsLogoutPayload
	err := json.Unmarshal(msg, &logoutPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %v", err)
	}
	auth.RemoveFromAuthMap(logoutPayload.Token.ID, true)
	return nil
}

// return the unauth payload
func UnauthHandler() ([]byte, error) {
	config.Logger.Info("Sending unauth broadcast")
	blob := structs.UnauthBroadcast{
		Type:      "structure",
		AuthLevel: "unauthorized",
		Login: struct {
			Remainder int `json:"remainder"`
		}{
			Remainder: 0,
		},
	}
	resp, err := json.Marshal(blob)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling message: %v", err)
	}
	// if err := conn.Write(websocket.TextMessage, resp); err != nil {
	// 	config.Logger.Error(fmt.Sprintf("Error writing unauth response: %v", err))
	// 	return
	// }
	return resp, nil
}

// startram action handler
// gonna get confusing if we have varied startram structs
func StartramHandler(msg []byte) error {
	var startramPayload structs.WsStartramPayload
	err := json.Unmarshal(msg, &startramPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal startram payload: %v", err)
	}
	switch startramPayload.Payload.Action {
	case "register":
		regCode := startramPayload.Payload.Key
		region := startramPayload.Payload.Region
		if err := startram.Register(regCode,region); err != nil {
			return fmt.Errorf("Failed registration: %v",err)
		}
		if err := broadcast.BroadcastToClients(); err != nil {
			config.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
		}
	case "regions":
		if err := broadcast.LoadStartramRegions(); err != nil {
			return fmt.Errorf("%v",err)
		}
	default:
		return fmt.Errorf("Unrecognized startram action: %v", startramPayload.Payload.Action)
	}
	return nil
}

// password reset handler
func PwHandler(msg []byte) error {
	var pwPayload structs.WsPwPayload
	err := json.Unmarshal(msg, &pwPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal password payload: %v", err)
	}
	switch pwPayload.Payload.Action {
	case "modify":
		config.Logger.Info("Setting new password")
		conf := config.Conf()
		if auth.Hasher(pwPayload.Payload.Old) == conf.PwHash {
			update := map[string]interface{}{
				"pwHash": auth.Hasher(pwPayload.Payload.Password),
			}
			if err := config.UpdateConf(update); err != nil {
				return fmt.Errorf("Unable to update password: %v",err)
			}
			LogoutHandler(msg)
		}
	default:
		return fmt.Errorf("Unrecognized password action: %v",pwPayload.Payload.Action)
	}
	return nil
}