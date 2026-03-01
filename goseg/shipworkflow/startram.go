package shipworkflow

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"os"

	"go.uber.org/zap"
)

var (
	dispatchUrbitPayloadFn  = defaultDispatchUrbitPayload
	recoverWireguardFleetFn = func(piers []string, deleteMinioClient bool) error {
		return nil
	}
)

func SetDispatchUrbitPayload(handler func(structs.WsUrbitPayload) error) {
	dispatchUrbitPayloadFn = handler
}

func SetRecoverWireguardFleet(fn func(piers []string, deleteMinioClient bool) error) {
	recoverWireguardFleetFn = fn
}

func defaultDispatchUrbitPayload(payload structs.WsUrbitPayload) error {
	return fmt.Errorf("no urbit dispatcher configured")
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
	PublishTransitionWithPolicy(
		startram.PublishEvent,
		structs.Event{Type: eventType, Data: fmt.Sprintf("Error: %s", errmsg)},
		structs.Event{Type: eventType, Data: nil},
		3*time.Second,
	)
}

func publishStartramCompletion(eventType string, data interface{}) {
	PublishTransitionWithPolicy(
		startram.PublishEvent,
		structs.Event{Type: eventType, Data: data},
		structs.Event{Type: eventType, Data: nil},
		3*time.Second,
	)
}

func HandleStartramServices() error {
	return broadcastGetStartramServices()
}

func HandleStartramRegions() error {
	return broadcastLoadStartramRegions()
}

func HandleStartramRestart() error {
	zap.L().Info("Restarting StarTram")
	startram.PublishEvent(structs.Event{Type: "restart", Data: "startram"})
	settings := config.StartramSettingsSnapshot()
	// only restart if startram is on
	if !settings.WgOn {
		PublishTransitionWithPolicy(
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
	if err := recoverWireguardFleetFn(settings.Piers, true); err != nil {
		PublishTransitionWithPolicy(
			startram.PublishEvent,
			structs.Event{Type: "restart", Data: fmt.Sprintf("Error: %v", err)},
			structs.Event{Type: "restart", Data: ""},
			3*time.Second,
		)
		return fmt.Errorf("recover wireguard fleet: %w", err)
	}

	PublishTransitionWithPolicy(
		startram.PublishEvent,
		structs.Event{Type: "restart", Data: "done"},
		structs.Event{Type: "restart", Data: ""},
		3*time.Second,
	)
	return nil
}

func HandleStartramToggle() error {
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
				if err := dispatchUrbitPayloadFn(payload); err != nil {
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
		PublishTransitionWithPolicy(
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

func HandleStartramRegister(regCode, region string) error {
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

	publishStartramCompletion("register", "complete")
	return nil
}

func HandleStartramEndpoint(endpoint string) error {
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

func HandleStartramCancel(key string, reset bool) error {
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

func HandleStartramReminder(remind bool) error {
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

func HandleStartramSetBackupPassword(password string) error {
	err := config.UpdateConfTyped(config.WithRemoteBackupPassword(password))
	if err != nil {
		return fmt.Errorf("set backup password: %w", err)
	}
	return nil
}

func HandleStartramUploadBackup(patp string) error {
	startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: "upload"})
	filePath := "backup.key"
	keyBytes, err := os.ReadFile(filePath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to read private key file: %v", err))
		return fmt.Errorf("failed to read private key file: %w", err)
	}
	decodedKeyBytes, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		zap.L().Error(fmt.Sprintf("failed to decode private key file: %v", err))
		return fmt.Errorf("failed to decode private key file: %w", err)
	}
	pk := strings.TrimSpace(string(decodedKeyBytes))
	err = startram.UploadBackup(patp, pk, filePath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to upload backup: %v", err))
		startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: fmt.Sprintf("%v", err)})
		time.Sleep(3 * time.Second)
		startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: nil})
		return fmt.Errorf("upload backup failed: %w", err)
	}
	startram.PublishEvent(structs.Event{Type: "uploadBackup", Data: nil})
	return nil
}

func broadcastGetStartramServices() error {
	return broadcastGetStartramServicesFn()
}

func broadcastLoadStartramRegions() error {
	return broadcastLoadStartramRegionsFn()
}

var (
	broadcastGetStartramServicesFn = defaultBroadcastGetStartramServices
	broadcastLoadStartramRegionsFn = defaultBroadcastLoadStartramRegions
)

func defaultBroadcastGetStartramServices() error { return nil }
func defaultBroadcastLoadStartramRegions() error { return nil }

func init() {
	// These callbacks are bound at runtime by the handler package to avoid cycles.
	broadcastGetStartramServicesFn = broadcast.GetStartramServices
	broadcastLoadStartramRegionsFn = broadcast.LoadStartramRegions
}
