package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/driveresolver"
	"groundseg/leakchannel"
	"groundseg/structs"
	"groundseg/system"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Native-Planet/perigee/libprg"
	"go.uber.org/zap"
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

var ErrInvalidLoginCredentials = errors.New("invalid login credentials")

func Initialize() {
	go HandleLeakAction()
}

func NewShipHandler(msg []byte) error {
	zap.L().Info("New ship")
	// Unmarshal JSON
	var shipPayload structs.WsNewShipPayload
	err := json.Unmarshal(msg, &shipPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal new ship payload: %w", err)
	}
	switch shipPayload.Payload.Action {
	case "boot":
		handleError := func(err error) {
			docker.PublishNewShipTransition(structs.NewShipTransition{Type: "freeError", Event: err.Error()})
			time.Sleep(5 * time.Second)
			docker.PublishNewShipTransition(structs.NewShipTransition{Type: "freeError", Event: ""})
		}
		keyType := shipPayload.Payload.KeyType
		if keyType == "master-ticket" {
			masterTicket := shipPayload.Payload.Key
			if !strings.HasPrefix(masterTicket, "~") {
				masterTicket = "~" + masterTicket
			}
			if len(masterTicket) != len("~sampel-sampel-sampel-sampel") {
				handleError(fmt.Errorf("Invalid master ticket length: %v", len(masterTicket)))
				return fmt.Errorf("Invalid master ticket length: %v", len(masterTicket))
			}
			kf, err := libprg.Keyfile(shipPayload.Payload.Patp, masterTicket, "", 0)
			if err != nil {
				handleError(fmt.Errorf("Couldn't get keyfile: %w", err))
				return err
			}
			shipPayload.Payload.Key = kf
		}
		driveResolution, err := driveresolver.Resolve(shipPayload.Payload.SelectedDrive)
		if err != nil {
			errMsg := fmt.Errorf("Failed to resolve selected drive: %w", err)
			handleError(errMsg)
			return errMsg
		}
		driveResolution, err = driveresolver.EnsureReady(driveResolution)
		if err != nil {
			errMsg := fmt.Errorf("Failed to prepare selected drive: %w", err)
			handleError(errMsg)
			return errMsg
		}
		// Check if patp is valid
		patp := sigRemove(shipPayload.Payload.Patp)
		isValid := checkPatp(patp)
		if !isValid {
			errMsg := fmt.Errorf("Invalid @p provided: %v", patp)
			// handleError(errMsg)
			return errMsg
		}
		go createUrbitShip(patp, shipPayload, driveResolution.Mountpoint)
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
			return fmt.Errorf("Failed to marshall payload for newly created %s for deletion: %w payload=%+v", patp, err, deletePayload)
		}
		if err := UrbitHandler(deleteMsg); err != nil {
			return fmt.Errorf("Failed to delete newly created %v: %w", patp, err)
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
		return make(map[string]string), fmt.Errorf("Couldn't unmarshal login payload: %w", err)
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
			return make(map[string]string), fmt.Errorf("Unable to process login: %w", err)
		}
		zap.L().Info(fmt.Sprintf("Session %s logged in", loginPayload.Token.ID))
		return token, nil
	} else {
		failedLogins++
		zap.L().Warn(fmt.Sprintf("Failed auth"))
		if failedLogins >= MaxFailedLogins && remainder == 0 {
			go enforceLockout()
		}
		loginError, _ := json.Marshal(map[string]string{
			"type":    "login-failed",
			"message": "Login failed. Please try again.",
		})
		conn.Write(loginError)
		return map[string]string{}, ErrInvalidLoginCredentials
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
			zap.L().Error(fmt.Sprintf("Couldn't broadcast lockout: %v", err))
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
		zap.L().Error(fmt.Sprintf("Couldn't broadcast lockout: %v", err))
	}
	broadcast.UnauthBroadcast(unauth)
}

// take a guess
func LogoutHandler(msg []byte) error {
	var logoutPayload structs.WsLogoutPayload
	err := json.Unmarshal(msg, &logoutPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal login payload: %w", err)
	}
	auth.RemoveFromAuthMap(logoutPayload.Token.ID, true)
	return nil
}

// return the c2c payload
func C2CHandler(msg []byte) ([]byte, error) {
	var c2cPayload structs.WsC2cPayload
	err := json.Unmarshal(msg, &c2cPayload)
	if err != nil {
		return nil, fmt.Errorf("Couldn't unmarshal c2c payload: %w", err)
	}
	var resp []byte
	switch c2cPayload.Payload.Type {
	case "c2c":
		if err := system.C2CConnect(c2cPayload.Payload.SSID, c2cPayload.Payload.Password); err != nil {
			return nil, fmt.Errorf("c2c connect failed: %w", err)
		}
	default:
		return nil, fmt.Errorf("Invalid c2c request")
		/*
			blob := structs.C2CBroadcast{
				Type:  "c2c",
				SSIDS: system.C2CStoredSSIDs,
			}
			resp, err = json.Marshal(blob)
			if err != nil {
				return nil, fmt.Errorf("Error unmarshalling message: %w", err)
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
		return nil, fmt.Errorf("Error unmarshalling message: %w", err)
	}
	return resp, nil
}

// password reset handler
func PwHandler(msg []byte, urbitMode bool) error {
	var pwPayload structs.WsPwPayload
	err := json.Unmarshal(msg, &pwPayload)
	if err != nil {
		return fmt.Errorf("Couldn't unmarshal password payload: %w", err)
	}
	switch pwPayload.Payload.Action {
	case "modify":
		zap.L().Info("Setting new password")
		conf := config.Conf()
		if auth.Hasher(pwPayload.Payload.Old) == conf.PwHash {
			if err := config.UpdateConfTyped(config.WithPwHash(auth.Hasher(pwPayload.Payload.Password))); err != nil {
				return fmt.Errorf("Unable to update password: %w", err)
			}
			if urbitMode {
				leakchannel.Logout <- struct{}{}
				return nil
			}
			if pwPayload.Token.ID == "" {
				return fmt.Errorf("Missing token id for logout after password update")
			}
			auth.RemoveFromAuthMap(pwPayload.Token.ID, true)
			return nil
		}
		return fmt.Errorf("Current password is incorrect")
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
