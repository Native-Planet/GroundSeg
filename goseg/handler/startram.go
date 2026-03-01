package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	startramServicesActionHandler = handleStartramServices
	startramRegionsActionHandler  = handleStartramRegions
	startramRegisterActionHandler = handleStartramRegister
	startramToggleActionHandler   = handleStartramToggle
	startramRestartActionHandler  = handleStartramRestart
	startramCancelActionHandler   = handleStartramCancel
	startramEndpointActionHandler = handleStartramEndpoint
	startramReminderActionHandler = handleStartramReminder
	startramSetBackupPWHandler    = handleStartramSetBackupPassword
)

func runStartramAsync(action string, fn func() error) {
	go func() {
		if err := fn(); err != nil {
			zap.L().Error(fmt.Sprintf("startram action %s failed: %v", action, err))
		}
	}()
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
	case "services":
		runStartramAsync("services", startramServicesActionHandler)
	case "regions":
		runStartramAsync("regions", startramRegionsActionHandler)
	case "register":
		regCode := startramPayload.Payload.Key
		region := startramPayload.Payload.Region
		runStartramAsync("register", func() error {
			return startramRegisterActionHandler(regCode, region)
		})
	case "toggle":
		runStartramAsync("toggle", startramToggleActionHandler)
	case "restart":
		runStartramAsync("restart", startramRestartActionHandler)
	case "cancel":
		key := startramPayload.Payload.Key
		reset := startramPayload.Payload.Reset
		runStartramAsync("cancel", func() error {
			return startramCancelActionHandler(key, reset)
		})
	case "endpoint":
		endpoint := startramPayload.Payload.Endpoint
		runStartramAsync("endpoint", func() error {
			return startramEndpointActionHandler(endpoint)
		})
	case "reminder":
		runStartramAsync("reminder", func() error {
			return startramReminderActionHandler(startramPayload.Payload.Remind)
		})
		/*
			case "restore-backup":
				go handleStartramRestoreBackup(startramPayload.Payload.Target, startramPayload.Payload.Patp, startramPayload.Payload.Backup, startramPayload.Payload.Key)
			case "upload-backup":
				go handleStartramUploadBackup(startramPayload.Payload.Patp)
		*/
	case "set-backup-password":
		runStartramAsync("set-backup-password", func() error {
			return startramSetBackupPWHandler(startramPayload.Payload.Password)
		})
	default:
		return fmt.Errorf("Unrecognized startram action: %v", startramPayload.Payload.Action)
	}
	return nil
}

func handleStartramServices() error {
	return broadcast.GetStartramServices()
}

func handleStartramRegions() error {
	return broadcast.LoadStartramRegions()
}

func appendOrchestrationError(stepErrors *[]error, context string, err error) {
	if err == nil {
		return
	}
	wrapped := fmt.Errorf("%s: %w", context, err)
	*stepErrors = append(*stepErrors, wrapped)
	zap.L().Error(wrapped.Error())
}

func publishStartramError(eventType, errmsg string, shouldLog bool) {
	if shouldLog {
		zap.L().Error(errmsg)
	}
	publishTransitionWithPolicy(
		startram.PublishEvent,
		structs.Event{Type: eventType, Data: fmt.Sprintf("Error: %s", errmsg)},
		structs.Event{Type: eventType, Data: nil},
		3*time.Second,
	)
}

func publishStartramCompletion(eventType string, data interface{}) {
	publishTransitionWithPolicy(
		startram.PublishEvent,
		structs.Event{Type: eventType, Data: data},
		structs.Event{Type: eventType, Data: nil},
		3*time.Second,
	)
}

func handleStartramRestart() error {
	zap.L().Info("Restarting StarTram")
	startram.PublishEvent(structs.Event{Type: "restart", Data: "startram"})
	settings := config.StartramSettingsSnapshot()
	// only restart if startram is on
	if !settings.WgOn {
		publishTransitionWithPolicy(
			startram.PublishEvent,
			structs.Event{Type: "restart", Data: "Error: startram is disabled"},
			structs.Event{Type: "restart", Data: ""},
			3*time.Second,
		)
		return fmt.Errorf("startram is disabled")
	}

	zap.L().Info("Recreating containers")
	startram.PublishEvent(structs.Event{Type: "restart", Data: "urbits"})
	startram.PublishEvent(structs.Event{Type: "restart", Data: "minios"})
	if err := RecoverWireguardFleet(settings.Piers, true); err != nil {
		publishTransitionWithPolicy(
			startram.PublishEvent,
			structs.Event{Type: "restart", Data: fmt.Sprintf("Error: %v", err)},
			structs.Event{Type: "restart", Data: ""},
			3*time.Second,
		)
		return fmt.Errorf("recover wireguard fleet: %w", err)
	}

	publishTransitionWithPolicy(
		startram.PublishEvent,
		structs.Event{Type: "restart", Data: "done"},
		structs.Event{Type: "restart", Data: ""},
		3*time.Second,
	)
	return nil
}

func handleStartramToggle() error {
	startram.PublishEvent(structs.Event{Type: "toggle", Data: "loading"})
	settings := config.StartramSettingsSnapshot()
	var stepErrors []error
	if settings.WgOn {
		if err := config.UpdateConfTyped(config.WithWgOn(false)); err != nil {
			appendOrchestrationError(&stepErrors, "update config to disable wireguard", err)
		}
		err := docker.StopContainerByName("wireguard")
		if err != nil {
			appendOrchestrationError(&stepErrors, "stop wireguard container", err)
		}
		// toggle ships back to local
		for _, patp := range settings.Piers {
			dockerConfig := config.UrbitConf(patp)
			if dockerConfig.Network == "wireguard" {
				payload := structs.WsUrbitPayload{
					Payload: structs.WsUrbitAction{
						Type:   "urbit",
						Action: "toggle-network",
						Patp:   patp,
					},
				}
				jsonData, err := json.Marshal(payload)
				if err != nil {
					appendOrchestrationError(&stepErrors, fmt.Sprintf("marshal toggle-network payload for %s", patp), err)
					continue
				}
				if err := UrbitHandler(jsonData); err != nil {
					appendOrchestrationError(&stepErrors, fmt.Sprintf("dispatch toggle-network for %s", patp), err)
				}
			}
		}
	} else {
		if err := config.UpdateConfTyped(config.WithWgOn(true)); err != nil {
			appendOrchestrationError(&stepErrors, "update config to enable wireguard", err)
		}
		_, err := docker.StartContainer("wireguard", "wireguard")
		if err != nil {
			appendOrchestrationError(&stepErrors, "start wireguard container", err)
		}
	}
	// delete mc
	if err := docker.DeleteContainer("mc"); err != nil {
		appendOrchestrationError(&stepErrors, "delete minio client container", err)
	}
	// load mc
	if err := docker.LoadMC(); err != nil {
		appendOrchestrationError(&stepErrors, "load minio client container", err)
	}
	for _, patp := range settings.Piers {
		// delete minio
		minio := fmt.Sprintf("minio_%s", patp)
		if err := docker.DeleteContainer(minio); err != nil {
			appendOrchestrationError(&stepErrors, fmt.Sprintf("delete %s container", minio), err)
		}
	}
	// load minio
	if err := docker.LoadMinIOs(); err != nil {
		appendOrchestrationError(&stepErrors, "load minio containers", err)
	}
	if joinedErr := errors.Join(stepErrors...); joinedErr != nil {
		publishTransitionWithPolicy(
			startram.PublishEvent,
			structs.Event{Type: "toggle", Data: fmt.Sprintf("Error: %v", joinedErr)},
			structs.Event{Type: "toggle", Data: nil},
			3*time.Second,
		)
		return joinedErr
	}
	startram.PublishEvent(structs.Event{Type: "toggle", Data: nil})
	return nil
}

func handleStartramRegister(regCode, region string) error {
	startram.PublishEvent(structs.Event{Type: "register", Data: "key"})
	// Reset Key Pair
	err := config.CycleWgKey()
	if err != nil {
		publishStartramError("register", fmt.Sprintf("%v", err), true)
		return fmt.Errorf("cycle wireguard key: %w", err)
	}
	// Register startram key
	if err := startram.Register(regCode, region); err != nil {
		publishStartramError("register", fmt.Sprintf("Failed registration: %v", err), true)
		return fmt.Errorf("startram register: %w", err)
	}
	// Register Services
	startram.PublishEvent(structs.Event{Type: "register", Data: "services"})
	if err := startram.RegisterExistingShips(); err != nil {
		publishStartramError("register", fmt.Sprintf("Unable to register ships: %v", err), true)
		return fmt.Errorf("register existing ships: %w", err)
	}
	// Start Wireguard
	startram.PublishEvent(structs.Event{Type: "register", Data: "starting"})
	if err := docker.LoadWireguard(); err != nil {
		publishStartramError("register", fmt.Sprintf("Unable to start Wireguard: %v", err), true)
		return fmt.Errorf("start wireguard: %w", err)
	}
	// Finish
	// debug
	//time.Sleep(2 * time.Second)
	//handleError("Self inflicted error for debug purposes")

	publishStartramCompletion("register", "complete")
	return nil
}

// endpoint action
func handleStartramEndpoint(endpoint string) error {
	// initialize
	startram.PublishEvent(structs.Event{Type: "endpoint", Data: "init"})
	settings := config.StartramSettingsSnapshot()
	// stop wireguard if running
	if settings.WgOn {
		startram.PublishEvent(structs.Event{Type: "endpoint", Data: "stopping"})
		if err := docker.StopContainerByName("wireguard"); err != nil {
			publishStartramError("endpoint", fmt.Sprintf("%v", err), false)
			return fmt.Errorf("stop wireguard: %w", err)
		}
	}
	// Wireguard registered
	if settings.WgRegistered {
		// unregister startram services if exists
		startram.PublishEvent(structs.Event{Type: "endpoint", Data: "unregistering"})
		for _, p := range settings.Piers {
			if err := startram.SvcDelete(p, "urbit"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v", p))
			}
			if err := startram.SvcDelete("s3."+p, "s3"); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v", p))
			}
		}
	}
	// reset pubkey
	startram.PublishEvent(structs.Event{Type: "endpoint", Data: "configuring"})
	err := config.CycleWgKey()
	if err != nil {
		publishStartramError("endpoint", fmt.Sprintf("%v", err), false)
		return fmt.Errorf("cycle wireguard key: %w", err)
	}
	// set endpoint to config and persist
	startram.PublishEvent(structs.Event{Type: "endpoint", Data: "finalizing"})
	err = config.UpdateConfTyped(
		config.WithEndpointURL(endpoint),
		config.WithWgRegistered(false),
	)
	if err != nil {
		publishStartramError("endpoint", fmt.Sprintf("%v", err), false)
		return fmt.Errorf("persist endpoint: %w", err)
	}

	publishStartramCompletion("endpoint", "complete")
	return nil
}

// cancel subscription with reg code
func handleStartramCancel(key string, reset bool) error {
	if reset {
		for _, svc := range config.GetStartramConfig().Subdomains {
			if err := startram.SvcDelete(svc.URL, svc.SvcType); err != nil {
				zap.L().Error(fmt.Sprintf("Couldn't delete service %v: %v", svc.URL, err))
			}
		}
	}
	if err := startram.CancelSub(key); err != nil {
		publishStartramError("cancelSub", fmt.Sprintf("Couldn't cancel subscription: %v", err), true)
		return fmt.Errorf("cancel subscription: %w", err)
	}
	if err := config.CycleWgKey(); err != nil {
		publishStartramError("cancelSub", fmt.Sprintf("%v", err), true)
		return fmt.Errorf("cycle wireguard key: %w", err)
	}
	publishStartramCompletion("cancelSub", nil)
	return nil
}

func handleStartramReminder(remind bool) error {
	ships := config.ShipSettingsSnapshot()
	var stepErrors []error
	for _, patp := range ships.Piers {
		if err := config.UpdateUrbit(patp, func(shipConf *structs.UrbitDocker) error {
			shipConf.StartramReminder = remind
			return nil
		}); err != nil {
			stepErrors = append(stepErrors, fmt.Errorf("update startram reminder for %s: %w", patp, err))
		}
	}
	if joined := errors.Join(stepErrors...); joined != nil {
		return joined
	}
	return nil
}

/*
// download the source backup, decrypt with key, and restore to target
func handleStartramRestoreBackup(target, source string, backup int, key string) {
	keyFile := "backup.key"
	startram.PublishEvent(structs.Event{Type: "restoreBackup", Data: "init"})
	if key == "" {
		keyBytes, err := os.ReadFile(keyFile)
		if err != nil || len(keyBytes) == 0 {
			zap.L().Error(fmt.Sprintf("No key provided and failed to read private key file: %v", err))
			return
		}
	}
	keyBytes, err := os.ReadFile(keyFile)
	if err != nil || len(keyBytes) == 0 {
		if os.IsNotExist(err) || len(keyBytes) == 0 {
			zap.L().Warn(fmt.Sprintf("Key file not found or empty, writing to %s...", keyFile))
			encodedKey := base64.StdEncoding.EncodeToString([]byte(key))
			err = os.WriteFile(keyFile, []byte(encodedKey), 0600)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Failed to write new key to file: %v", err))
				return
			}
			zap.L().Info("Backup key is saved.")
		} else {
			zap.L().Error(fmt.Sprintf("No key provided and failed to read private key file: %v", err))
			startram.PublishEvent(structs.Event{Type: "restoreBackup", Data: "error"})
			time.Sleep(3 * time.Second)
			startram.PublishEvent(structs.Event{Type: "restoreBackup", Data: nil})
			return
		}
	}
	startram.PublishEvent(structs.Event{Type: "restoreBackup", Data: "download"})
	backupFile, err := startram.GetBackup(source, fmt.Sprintf("%d", backup), key)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to get backup: %v", err))
		startram.PublishEvent(structs.Event{Type: "restoreBackup", Data: "error"})
		time.Sleep(3 * time.Second)
		startram.PublishEvent(structs.Event{Type: "restoreBackup", Data: nil})
		return
	}
	startram.PublishEvent(structs.Event{Type: "backup", Data: nil})
	handleNotImplement(fmt.Sprintf("restore %s from %s backup %v", target, source, backupFile))
}
*/

func handleStartramUploadBackup(patp string) {
	startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: "upload"})
	filePath := "backup.key"
	keyBytes, err := os.ReadFile(filePath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to read private key file: %v", err))
		return
	}
	decodedKeyBytes, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to decode private key file: %v", err))
		return
	}
	pk := strings.TrimSpace(string(decodedKeyBytes))
	err = startram.UploadBackup(patp, pk, filePath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to upload backup: %v", err))
		startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: fmt.Sprintf("%v", err)})
		time.Sleep(3 * time.Second)
		startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: nil})
		return
	}
	startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: nil})
	handleNotImplement("upload backup")
}

// temp
func handleNotImplement(action string) {
	zap.L().Error(fmt.Sprintf("temp error: %v not implemented", action))
}

func handleStartramSetBackupPassword(password string) error {
	err := config.UpdateConfTyped(config.WithRemoteBackupPassword(password))
	if err != nil {
		return fmt.Errorf("set backup password: %w", err)
	}
	return nil
}
