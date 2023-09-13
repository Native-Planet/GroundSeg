package handler

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/broadcast"
	"goseg/config"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"
	"goseg/system"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/gorilla/websocket"
)

// todo
// handle bug report stuff
func SupportHandler(msg []byte, payload structs.WsPayload, r *http.Request, conn *websocket.Conn) error {
	logger.Logger.Info("Support")
	return nil
}

func NewShipHandler(msg []byte) error {
	logger.Logger.Info("New ship")
	// Unmarshal JSON
	var shipPayload structs.WsNewShipPayload
	err := json.Unmarshal(msg, &shipPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal new ship payload: %v", err)
	}
	switch shipPayload.Payload.Action {
	case "boot":
		// Check if patp is valid
		patp := sigRemove(shipPayload.Payload.Patp)
		isValid := checkPatp(patp)
		if !isValid {
			return fmt.Errorf("Invalid @p provided: %v", patp)
		}
		go createUrbitShip(patp, shipPayload)
	case "reset":
		err := resetNewShip()
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown NewShip action: %v", shipPayload.Payload.Action)
	}
	return nil
}

// handle system events
func SystemHandler(msg []byte) error {
	logger.Logger.Info("System")
	var systemPayload structs.WsSystemPayload
	err := json.Unmarshal(msg, &systemPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal system payload: %v", err)
	}
	switch systemPayload.Payload.Action {
	case "power":
		switch systemPayload.Payload.Command {
		case "shutdown":
			logger.Logger.Info(fmt.Sprintf("Device shutdown requested"))
			if config.DebugMode {
				logger.Logger.Info(fmt.Sprintf("DebugMode detected, skipping shutdown. Exiting program."))
				os.Exit(0)
			} else {
				logger.Logger.Info(fmt.Sprintf("Turning off device.."))
				cmd := exec.Command("shutdown", "-h", "now")
				cmd.Run()
			}
		case "restart":
			logger.Logger.Info(fmt.Sprintf("Device restart requested"))
			if config.DebugMode {
				logger.Logger.Info(fmt.Sprintf("DebugMode detected, skipping restart. Exiting program."))
				os.Exit(0)
			} else {
				logger.Logger.Info(fmt.Sprintf("Restarting device.."))
				cmd := exec.Command("reboot")
				cmd.Run()
			}
		default:
			return fmt.Errorf("Unrecognized power command: %v", systemPayload.Payload.Command)
		}
	case "modify-swap":
		logger.Logger.Info(fmt.Sprintf("Updating swap with value %v",systemPayload.Payload.Value))
		broadcast.SysTransBus <- structs.SystemTransitionBroadcast{Swap: systemPayload.Payload.Value, Type: "swap"}
		swapfile := config.BasePath + "/swapfile"
		if err := system.ConfigureSwap(swapfile, systemPayload.Payload.Value); err != nil {
			logger.Logger.Error(fmt.Sprintf("Unable to set swap: %v", err))
			return fmt.Errorf("Unable to set swap: %v", err)
		}
		if err = config.UpdateConf(map[string]interface{}{
			"swapVal":  systemPayload.Payload.Value,
		}); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't update swap value: %v", err))
		}
	default:
		return fmt.Errorf("Unrecognized system action: %v", systemPayload.Payload.Action)
	}
	return nil
}

// handle urbit-type events
func UrbitHandler(msg []byte) error {
	logger.Logger.Info("Urbit")
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
				logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
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
			logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
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
				logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
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
				logger.Logger.Error(fmt.Sprintf("Unable to broadcast to clients: %v", err))
			}
			docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: ""}
		}
		return nil
	default:
		return fmt.Errorf("Unrecognized urbit action: %v", urbitPayload.Payload.Type)
	}
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
		logger.Logger.Info(fmt.Sprintf("Session %s logged in", loginPayload.Token.ID))
		return nil
	} else {
		return fmt.Errorf("Failed auth: %v", loginPayload.Payload.Password)
	}
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
	logger.Logger.Info("Sending unauth broadcast")
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
	// 	logger.Logger.Error(fmt.Sprintf("Error writing unauth response: %v", err))
	// 	return
	// }
	return resp, nil
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
		logger.Logger.Info("Setting new password")
		conf := config.Conf()
		if auth.Hasher(pwPayload.Payload.Old) == conf.PwHash {
			update := map[string]interface{}{
				"pwHash": auth.Hasher(pwPayload.Payload.Password),
			}
			if err := config.UpdateConf(update); err != nil {
				return fmt.Errorf("Unable to update password: %v", err)
			}
			LogoutHandler(msg)
		}
	default:
		return fmt.Errorf("Unrecognized password action: %v", pwPayload.Payload.Action)
	}
	return nil
}

// SigRemove removes the '~' prefix from patp if it exists
func sigRemove(patp string) string {
	if patp != "" {
		if strings.HasPrefix(patp, "~") {
			patp = patp[1:]
		}
	}
	return patp
}

// CheckPatp checks if patp is correct
func checkPatp(patp string) bool {
	// Handle undefined patp
	if patp == "" {
		return false
	}

	// Split the string by hyphen
	wordlist := strings.Split(patp, "-")

	// Define the regular expression pattern
	pattern := regexp.MustCompile("^[a-z]{6}$|^[a-z]{3}$")

	// Define pre and suf (truncated for brevity)
	pre := "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
	suf := "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

	for _, word := range wordlist {
		// Check regular expression match
		if !pattern.MatchString(word) {
			return false
		}

		// Check prefixes and suffixes
		if len(word) > 3 {
			if !strings.Contains(pre, word[0:3]) || !strings.Contains(suf, word[3:6]) {
				return false
			}
		} else {
			if !strings.Contains(suf, word) {
				return false
			}
		}
	}
	return true
}
