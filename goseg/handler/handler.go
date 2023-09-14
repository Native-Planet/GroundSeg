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
	"goseg/startram"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	failedLogins	int
	remainder	    int
	loginMu			sync.Mutex
)

const (
	MaxFailedLogins = 5
	LockoutDuration = 2 * time.Minute
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
		logger.Logger.Info(fmt.Sprintf("Updating swap with value %v", systemPayload.Payload.Value))
		broadcast.SysTransBus <- structs.SystemTransitionBroadcast{Swap: true, Type: "swap"}
		swapfile := config.BasePath + "/swapfile"
		if err := system.ConfigureSwap(swapfile, systemPayload.Payload.Value); err != nil {
			logger.Logger.Error(fmt.Sprintf("Unable to set swap: %v", err))
			broadcast.SysTransBus <- structs.SystemTransitionBroadcast{Swap: false, Type: "swap"}
			return fmt.Errorf("Unable to set swap: %v", err)
		}
		if err = config.UpdateConf(map[string]interface{}{
			"swapVal": systemPayload.Payload.Value,
		}); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't update swap value: %v", err))
		}
		go func() {
			time.Sleep(2 * time.Second)
			broadcast.SysTransBus <- structs.SystemTransitionBroadcast{Swap: false, Type: "swap"}
		}()
		logger.Logger.Info(fmt.Sprintf("Swap successfully set to %v", systemPayload.Payload.Value))
	case "update":
		if systemPayload.Payload.Update == "linux" {
			if err := system.RunUpgrade(); err != nil {
				logger.Logger.Error(fmt.Sprintf("Error updating host system: %v", err))
			}
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
		logger.Logger.Warn(fmt.Sprintf("%v", currentNetwork))
		if currentNetwork == "wireguard" {
			shipConf.Network = "none"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
		} else if currentNetwork == "none" && conf.WgRegistered == true {
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
		docker.UTransBus <- structs.UrbitTransition{Patp: patp, Type: "togglePower", Event: "loading"}
		update := make(map[string]structs.UrbitDocker)
		if shipConf.BootStatus == "noboot" {
			shipConf.BootStatus = "boot"
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				return fmt.Errorf("Couldn't update urbit config: %v", err)
			}
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
	case "delete-ship":
		conf := config.Conf()
		var res []string
		for _, pier := range conf.Piers {
			if pier != patp {
				res = append(res, pier)
			}
		}
		if err = config.UpdateConf(map[string]interface{}{
			"piers":  res,
		}); err != nil {
			return fmt.Errorf("Couldn't remove pier from config! %v",patp)
		}
		if err := docker.DeleteVolume(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't remove docker volume for %v",patp))
		}
		if conf.WgRegistered {
			if err := startram.SvcDelete(patp,"urbit"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't remove urbit anchor for %v",patp))
			}
			if err := startram.SvcDelete("s3."+patp,"s3"); err != nil {
				logger.Logger.Error(fmt.Sprintf("Couldn't remove s3 anchor for %v",patp))
			}
		}
		if err := config.RemoveUrbitConfig(patp); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't remove config for %v",patp))
		}
		return nil
	default:
		return fmt.Errorf("Unrecognized urbit action: %v", urbitPayload.Payload.Action)
	}
}

// validate password and add to auth session map
func LoginHandler(conn *structs.MuConn, msg []byte) error {
	loginMu.Lock()
	defer loginMu.Unlock()
	var loginPayload structs.WsLoginPayload
	err := json.Unmarshal(msg, &loginPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %v", err)
	}
	isAuthenticated := auth.AuthenticateLogin(loginPayload.Payload.Password)
	if isAuthenticated {
		failedLogins = 0
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
		failedLogins++
		logger.Logger.Warn(fmt.Sprintf("Failed auth: %v", loginPayload.Payload.Password))
		if failedLogins >= MaxFailedLogins && remainder == 0 {
			go enforceLockout()
		}
		return nil
	}
}

func enforceLockout() {
	remainder = 120
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for remainder > 0 {
		unauth, err := UnauthHandler()
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't broadcast lockout: %v", err))
		}
		broadcast.UnauthBroadcast(unauth)
		<-ticker.C
		remainder -= 1
	}
	loginMu.Lock()
	defer loginMu.Unlock()
	failedLogins = 0
	remainder = 0

	unauth, err := UnauthHandler()
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Couldn't broadcast lockout: %v", err))
	}
	broadcast.UnauthBroadcast(unauth)
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
			Remainder: remainder,
		},
	}
	resp, err := json.Marshal(blob)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling message: %v", err)
	}
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
