package handler

import (
	"encoding/json"
	"fmt"
	"goseg/auth"
	"goseg/broadcast"
	"goseg/config"
	"goseg/logger"
	"goseg/structs"
	"goseg/system"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	failedLogins int
	remainder    int
	loginMu      sync.Mutex
)

const (
	MaxFailedLogins = 5
	LockoutDuration = 2 * time.Minute
)

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
				logger.Logger.Debug(fmt.Sprintf("DebugMode detected, skipping shutdown. Exiting program."))
				os.Exit(0)
			} else {
				logger.Logger.Info(fmt.Sprintf("Turning off device.."))
				cmd := exec.Command("shutdown", "-h", "now")
				cmd.Run()
			}
		case "restart":
			logger.Logger.Info(fmt.Sprintf("Device restart requested"))
			if config.DebugMode {
				logger.Logger.Debug(fmt.Sprintf("DebugMode detected, skipping restart. Exiting program."))
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
		conf := config.Conf()
		file := conf.SwapFile
		if err := system.ConfigureSwap(file, systemPayload.Payload.Value); err != nil {
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
	case "wifi-toggle":
		if err := system.ToggleDevice(system.Device); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't toggle wifi device: %v", err))
		}
	case "wifi-connect":
		if err := system.ConnectToWifi(system.Device, systemPayload.Payload.SSID, systemPayload.Payload.Password); err != nil {
			logger.Logger.Error(fmt.Sprintf("Couldn't connect to wifi: %v", err))
		}
	default:
		return fmt.Errorf("Unrecognized system action: %v", systemPayload.Payload.Action)
	}
	return nil
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
		logger.Logger.Warn(fmt.Sprintf("Failed auth"))
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
