package handler

import (
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/leakchannel"
	"groundseg/logger"
	"groundseg/structs"
	"groundseg/system"
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

func init() {
	go HandleLeakAction()
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
		// handle filesystem
		sel := shipPayload.Payload.SelectedDrive //string
		// if not using system-drive, that means custom location
		if sel != "system-drive" {
			// get list of devices -- lsblk -f
			blockDevices, err := system.ListHardDisks()
			if err != nil {
				return fmt.Errorf("Failed to retrieve block devices: %v", err)
			}
			// we're looking for the drive the user specified
			for _, dev := range blockDevices.BlockDevices {
				if dev.Name == sel {
					// lets see if its structured correctly
					for _, m := range dev.Mountpoints {
						matched, err := regexp.MatchString(`^/groundseg-\d+$`, m)
						if err != nil {
							return fmt.Errorf("Regex match error: %v", err)
						}
						// device provided in payload does not match groundseg's format
						if !matched {
							// we overwrite the fs
							if err := system.CreateGroundSegFilesystem(sel); err != nil {
								return err
							}
						}
					}
				}
			}
		}
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
	case "cancel":
		// send to UrbitHandler's delete-ship
		patp := shipPayload.Payload.Patp
		deletePayload := structs.WsUrbitPayload{
			Payload: structs.WsUrbitAction{
				Type:   "urbit",
				Action: "delete-ship",
				Patp:   patp,
			},
		}
		deleteMsg, err := json.Marshal(deletePayload)
		if err != nil {
			return fmt.Errorf("Failed to marshall payload for newly create %v for deletion: %v: %+v", err, deletePayload)
		}
		if err := UrbitHandler(deleteMsg); err != nil {
			return fmt.Errorf("Failed to delete newly created %v: %v", patp, err)
		}
		if err := resetNewShip(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown NewShip action: %v", shipPayload.Payload.Action)
	}
	return nil
}

// validate password and add to auth session map
func LoginHandler(conn *structs.MuConn, msg []byte) (map[string]string, error) {
	loginMu.Lock()
	defer loginMu.Unlock()
	var loginPayload structs.WsLoginPayload
	err := json.Unmarshal(msg, &loginPayload)
	if err != nil {
		return make(map[string]string), fmt.Errorf("Couldn't unmarshal login payload: %v", err)
	}
	isAuthenticated := auth.AuthenticateLogin(loginPayload.Payload.Password)
	if isAuthenticated {
		failedLogins = 0
		newToken, err := auth.AuthToken(loginPayload.Token.Token)
		if err != nil {
			return make(map[string]string), err
		}
		token := map[string]string{
			"id":    loginPayload.Token.ID,
			"token": newToken,
		}
		if err := auth.AddToAuthMap(conn.Conn, token, true); err != nil {
			return make(map[string]string), fmt.Errorf("Unable to process login: %v", err)
		}
		logger.Logger.Info(fmt.Sprintf("Session %s logged in", loginPayload.Token.ID))
		return token, nil
	} else {
		failedLogins++
		logger.Logger.Warn(fmt.Sprintf("Failed auth"))
		if failedLogins >= MaxFailedLogins && remainder == 0 {
			go enforceLockout()
		}
		loginError, _ := json.Marshal(map[string]string{
			"type":    "login-failed",
			"message": "Login failed. Please try again.",
		})
		conn.Write(loginError)
		return map[string]string{"id": loginPayload.Token.ID, "token": loginPayload.Token.Token}, nil
	}
}

func enforceLockout() {
	// todo: extend remainder
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

// return the c2c payload
func C2CHandler(msg []byte) ([]byte, error) {
	var c2cPayload structs.WsC2cPayload
	err := json.Unmarshal(msg, &c2cPayload)
	if err != nil {
		return nil, fmt.Errorf("Couldn't unmarshal c2c payload: %v", err)
	}
	var resp []byte
	switch c2cPayload.Payload.Type {
	case "c2c":
		system.C2CConnect(c2cPayload.Payload.SSID, c2cPayload.Payload.Password)
	default:
		return nil, fmt.Errorf("Invalid c2c request")
		/*
			blob := structs.C2CBroadcast{
				Type:  "c2c",
				SSIDS: system.C2CStoredSSIDs,
			}
			resp, err = json.Marshal(blob)
			if err != nil {
				return nil, fmt.Errorf("Error unmarshalling message: %v", err)
			}
		*/
	}
	return resp, nil
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
func PwHandler(msg []byte, urbitMode bool) error {
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
			if urbitMode {
				leakchannel.Logout <- struct{}{}
				return nil
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
