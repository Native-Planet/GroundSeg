package shipworkflow

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Native-Planet/perigee/libprg"

	"groundseg/docker/events"
	"groundseg/driveresolver"
	"groundseg/structs"
)

var (
	decodeMasterTicketFn = libprg.Keyfile
	resolveDriveFn       = driveresolver.Resolve
	ensureDriveReadyFn   = driveresolver.EnsureReady
	normalizePatpFn      = normalizePatp
	validatePatpFn       = isValidPatp
)

func HandleNewShip(msg []byte, bootFn func(structs.WsNewShipPayload) error, cancelFn func(string) error, resetFn func() error) error {
	var shipPayload structs.WsNewShipPayload
	if err := json.Unmarshal(msg, &shipPayload); err != nil {
		return fmt.Errorf("Couldn't unmarshal new ship payload: %w", err)
	}
	switch shipPayload.Payload.Action {
	case "boot":
		if bootFn == nil {
			return fmt.Errorf("No new ship boot handler configured")
		}
		return bootFn(shipPayload)
	case "reset":
		if resetFn == nil {
			return fmt.Errorf("No new ship reset handler configured")
		}
		return resetFn()
	case "cancel":
		if cancelFn == nil {
			return fmt.Errorf("No new ship cancel handler configured")
		}
		if err := cancelFn(shipPayload.Payload.Patp); err != nil {
			return err
		}
		if resetFn == nil {
			return fmt.Errorf("No new ship reset handler configured")
		}
		return resetFn()
	default:
		return fmt.Errorf("Unknown NewShip action: %v", shipPayload.Payload.Action)
	}
}

func HandleNewShipBoot(shipPayload structs.WsNewShipPayload) error {
	handleError := func(err error) error {
		events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "freeError", Event: err.Error()})
		time.Sleep(5 * time.Second)
		events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "freeError", Event: ""})
		return err
	}

	keyType := shipPayload.Payload.KeyType
	if keyType == "master-ticket" {
		masterTicket := shipPayload.Payload.Key
		if !strings.HasPrefix(masterTicket, "~") {
			masterTicket = "~" + masterTicket
		}
		if len(masterTicket) != len("~sampel-sampel-sampel-sampel") {
			return handleError(fmt.Errorf("Invalid master ticket length: %v", len(masterTicket)))
		}
		kf, err := decodeMasterTicketFn(shipPayload.Payload.Patp, masterTicket, "", 0)
		if err != nil {
			return handleError(fmt.Errorf("Couldn't get keyfile: %w", err))
		}
		shipPayload.Payload.Key = kf
	}

	driveResolution, err := resolveDriveFn(shipPayload.Payload.SelectedDrive)
	if err != nil {
		return handleError(fmt.Errorf("Failed to resolve selected drive: %w", err))
	}
	driveResolution, err = ensureDriveReadyFn(driveResolution)
	if err != nil {
		return handleError(fmt.Errorf("Failed to prepare selected drive: %w", err))
	}

	patp := normalizePatpFn(shipPayload.Payload.Patp)
	if !validatePatpFn(patp) {
		return handleError(fmt.Errorf("Invalid @p provided: %v", patp))
	}

	go ProvisionShip(patp, shipPayload, driveResolution.Mountpoint)
	return nil
}

func ResetNewShip() error {
	events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "bootStage", Event: ""})
	events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "patp", Event: ""})
	events.DefaultEventRuntime().PublishNewShipTransition(context.Background(), structs.NewShipTransition{Type: "error", Event: ""})
	return nil
}

func CancelNewShip(patp string) error {
	return DeleteShip(patp)
}

func NormalizePatp(patp string) string {
	return normalizePatp(patp)
}

func IsValidPatp(patp string) bool {
	return isValidPatp(patp)
}

func normalizePatp(patp string) string {
	if strings.HasPrefix(patp, "~") {
		return patp[1:]
	}
	return patp
}

func isValidPatp(patp string) bool {
	if patp == "" {
		return false
	}
	pattern := regexp.MustCompile("^[a-z]{6}$|^[a-z]{3}$")
	pre := "dozmarbinwansamlitsighidfidlissogdirwacsabwissibrigsoldopmodfoglidhopdardorlorhodfolrintogsilmirholpaslacrovlivdalsatlibtabhanticpidtorbolfosdotlosdilforpilramtirwintadbicdifrocwidbisdasmidloprilnardapmolsanlocnovsitnidtipsicropwitnatpanminritpodmottamtolsavposnapnopsomfinfonbanmorworsipronnorbotwicsocwatdolmagpicdavbidbaltimtasmalligsivtagpadsaldivdactansidfabtarmonranniswolmispallasdismaprabtobrollatlonnodnavfignomnibpagsopralbilhaddocridmocpacravripfaltodtiltinhapmicfanpattaclabmogsimsonpinlomrictapfirhasbosbatpochactidhavsaplindibhosdabbitbarracparloddosbortochilmactomdigfilfasmithobharmighinradmashalraglagfadtopmophabnilnosmilfopfamdatnoldinhatnacrisfotribhocnimlarfitwalrapsarnalmoslandondanladdovrivbacpollaptalpitnambonrostonfodponsovnocsorlavmatmipfip"
	suf := "zodnecbudwessevpersutletfulpensytdurwepserwylsunrypsyxdyrnuphebpeglupdepdysputlughecryttyvsydnexlunmeplutseppesdelsulpedtemledtulmetwenbynhexfebpyldulhetmevruttylwydtepbesdexsefwycburderneppurrysrebdennutsubpetrulsynregtydsupsemwynrecmegnetsecmulnymtevwebsummutnyxrextebfushepbenmuswyxsymselrucdecwexsyrwetdylmynmesdetbetbeltuxtugmyrpelsyptermebsetdutdegtexsurfeltudnuxruxrenwytnubmedlytdusnebrumtynseglyxpunresredfunrevrefmectedrusbexlebduxrynnumpyxrygryxfeptyrtustyclegnemfermertenlusnussyltecmexpubrymtucfyllepdebbermughuttunbylsudpemdevlurdefbusbeprunmelpexdytbyttyplevmylwedducfurfexnulluclennerlexrupnedlecrydlydfenwelnydhusrelrudneshesfetdesretdunlernyrsebhulrylludremlysfynwerrycsugnysnyllyndyndemluxfedsedbecmunlyrtesmudnytbyrsenwegfyrmurtelreptegpecnelnevfes"

	for _, word := range strings.Split(patp, "-") {
		if !pattern.MatchString(word) {
			return false
		}
		if len(word) > 3 {
			if !strings.Contains(pre, word[0:3]) || !strings.Contains(suf, word[3:6]) {
				return false
			}
		} else if !strings.Contains(suf, word) {
			return false
		}
	}
	return true
}
