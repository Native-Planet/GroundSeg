package rectify

// this package is for watching event channels and rectifying mismatches
// between the desired and actual state, creating broadcast transitions,
// and anything else that needs to be done asyncronously

import (
	"fmt"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
	"strings"

	"go.uber.org/zap"
)

func UrbitTransitionHandler() {
	for {
		event := <-docker.UrbitTransitions()
		applyTransitionUpdate("urbit", func(current *structs.AuthBroadcast) {
			urbitStruct, exists := current.Urbits[event.Patp]
			if !exists {
				return
			}
			if !setUrbitTransition(&urbitStruct.Transition, event) {
				zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", event.Type))
				return
			}
			current.Urbits[event.Patp] = urbitStruct
		})
	}
}

func setUrbitTransition(transitionState *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
	switch transition.UrbitTransitionType(event.Type) {
	case transition.UrbitTransitionRollChop:
		transitionState.RollChop = event.Event
	case transition.UrbitTransitionChopOnUpgrade:
		transitionState.ChopOnUpgrade = event.Event
	case transition.UrbitTransitionChop:
		transitionState.Chop = event.Event
	case transition.UrbitTransitionPack:
		transitionState.Pack = event.Event
	case transition.UrbitTransitionPackMeld:
		transitionState.PackMeld = event.Event
	case transition.UrbitTransitionLoom:
		transitionState.Loom = event.Event
	case transition.UrbitTransitionSnapTime:
		transitionState.SnapTime = event.Event
	case transition.UrbitTransitionUrbitDomain:
		transitionState.UrbitDomain = event.Event
	case transition.UrbitTransitionMinIODomain:
		transitionState.MinIODomain = event.Event
	case transition.UrbitTransitionRebuildContainer:
		transitionState.RebuildContainer = event.Event
	case transition.UrbitTransitionToggleDevMode:
		transitionState.ToggleDevMode = event.Event
	case transition.UrbitTransitionTogglePower:
		transitionState.TogglePower = event.Event
	case transition.UrbitTransitionToggleNetwork:
		transitionState.ToggleNetwork = event.Event
	case transition.UrbitTransitionExportShip:
		transitionState.ExportShip = event.Event
	case transition.UrbitTransitionShipCompressed:
		transitionState.ShipCompressed = event.Value
	case transition.UrbitTransitionExportBucket:
		transitionState.ExportBucket = event.Event
	case transition.UrbitTransitionBucketCompressed:
		transitionState.BucketCompressed = event.Value
	case transition.UrbitTransitionDeleteShip:
		transitionState.DeleteShip = event.Event
	case transition.UrbitTransitionToggleMinIOLink:
		transitionState.ToggleMinIOLink = event.Event
	case transition.UrbitTransitionPenpaiCompanion:
		transitionState.PenpaiCompanion = event.Event
	case transition.UrbitTransitionGallseg:
		transitionState.Gallseg = event.Event
	case transition.UrbitTransitionDeleteService:
		transitionState.StartramServices = event.Event
	case transition.UrbitTransitionLocalTlonBackupsEnabled:
		transitionState.LocalTlonBackupsEnabled = event.Event
	case transition.UrbitTransitionRemoteTlonBackupsEnabled:
		transitionState.RemoteTlonBackupsEnabled = event.Event
	case transition.UrbitTransitionLocalTlonBackup:
		transitionState.LocalTlonBackup = event.Event
	case transition.UrbitTransitionLocalTlonBackupSchedule:
		transitionState.LocalTlonBackupSchedule = event.Event
	case transition.UrbitTransitionHandleRestoreTlonBackup:
		transitionState.HandleRestoreTlonBackup = event.Event
	case transition.UrbitTransitionServiceRegistrationStatus:
		transitionState.ServiceRegistrationStatus = event.Event
	default:
		return false
	}
	return true
}

func NewShipTransitionHandler() {
	for {
		event := <-docker.NewShipTransitions()
		applyTransitionUpdate("new ship", func(current *structs.AuthBroadcast) {
			if !setNewShipTransition(&current.NewShip, event) {
				zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", event.Type))
			}
		})
	}
}

func setNewShipTransition(target *structs.NewShip, event structs.NewShipTransition) bool {
	switch transition.NewShipTransitionType(event.Type) {
	case transition.NewShipTransitionError:
		target.Transition.Error = event.Event
	case transition.NewShipTransitionBootStage:
		// Events
		// starting: setting up docker and config
		// creating: actually create and start the container
		// booting: waiting until +code shows up
		// completed: ready to reset
		// aborted: something went wrong and we ran the cleanup routine
		// <empty>: free for new ship
		target.Transition.BootStage = event.Event
	case transition.NewShipTransitionPatp:
		target.Transition.Patp = event.Event
	case transition.NewShipTransitionFreeError:
		target.Transition.FreeError = event.Event
	default:
		return false
	}
	return true
}

func setSystemTransition(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
	switch transition.SystemTransitionType(event.Type) {
	case transition.SystemTransitionWifiConnect:
		target.WifiConnect = event.Event
	case transition.SystemTransitionSwap:
		target.Swap = event.BoolEvent
	case transition.SystemTransitionBugReport:
		target.BugReport = event.Event
	case transition.SystemTransitionBugReportError:
		target.BugReportError = event.Event
	default:
		return false
	}
	return true
}

func setStartramTransition(target *structs.StartramTransition, eventType string, eventData any) bool {
	switch transition.EventType(eventType) {
	case transition.StartramTransitionRestart:
		target.Restart = fmt.Sprintf("%v", eventData)
	case transition.StartramTransitionEndpoint:
		if eventData == nil {
			target.Endpoint = ""
		} else {
			target.Endpoint = fmt.Sprintf("%v", eventData)
		}
	case transition.StartramTransitionToggle:
		target.Toggle = eventData
	case transition.StartramTransitionRegister:
		target.Register = eventData
	default:
		return false
	}
	return true
}

func applyTransitionUpdate(context string, mutate func(*structs.AuthBroadcast)) {
	if err := broadcast.ApplyBroadcastUpdate(true, mutate); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to publish %s transition update: %v", context, err))
	}
}

func RectifyUrbit() {
	for {
		event := <-startram.Events()
		switch transition.EventType(event.Type) {
		case transition.StartramTransitionRestart:
			// startram - restarting wireguard container
			// urbits - recreating urbit containers
			// minios - recreating minio containers
			// done - completed
			applyTransitionUpdate("startram", func(current *structs.AuthBroadcast) {
				_ = setStartramTransition(&current.Profile.Startram.Transition, string(event.Type), event.Data)
			})
		case transition.StartramTransitionEndpoint:
			//init - started
			// unregistering - startram services unregistering
			// stopping - wireguard stopping
			// configuring - reset pubkey
			// finalizing - modifying endpoint
			// complete - successfully changed endpoint
			// Error: <text> - Error with info
			// nil/null - Empty, ready for action
			applyTransitionUpdate("startram", func(current *structs.AuthBroadcast) {
				_ = setStartramTransition(&current.Profile.Startram.Transition, string(event.Type), event.Data)
				if event.Data == transition.StartramTransitionComplete {
					conf := config.Conf()
					current.Profile.Startram.Info.Endpoint = conf.EndpointUrl
				}
			})
		case transition.StartramTransitionToggle:
			// loading - loading
			// nil/null - Empty
			applyTransitionUpdate("startram", func(current *structs.AuthBroadcast) {
				_ = setStartramTransition(&current.Profile.Startram.Transition, string(event.Type), event.Data)
			})
		case transition.StartramTransitionRegister:
			// key - registering startram key
			// services - registering startram services
			// starting - applying wg0.conf and starting container
			// complete - successfully finished registering
			// Error: <text> - Error with info
			// nil/null - Empty, ready for action
			applyTransitionUpdate("startram", func(current *structs.AuthBroadcast) {
				_ = setStartramTransition(&current.Profile.Startram.Transition, string(event.Type), event.Data)
				if event.Data == transition.StartramTransitionComplete {
					conf := config.Conf()
					current.Profile.Startram.Info.Running = conf.WgOn
					containerState, exists := config.GetContainerState()[string(transition.ContainerTypeWireguard)]
					if exists {
						running := containerState.ActualStatus == string(transition.ContainerStatusRunning)
						current.Profile.Startram.Info.Running = running
						if err := config.UpdateConfTyped(config.WithWgOn(running)); err != nil {
							zap.L().Error(fmt.Sprintf("%v", err))
						}
					}
					current.Profile.Startram.Info.Registered = conf.WgRegistered
				}
			})
		case transition.StartramTransitionRetrieve:
			conf := config.Conf()
			for patp := range config.UrbitConfAll() {
				modified := false
				serviceCreated := true
				startramConfig := config.GetStartramConfig()
				config.LoadUrbitConfig(patp)
				local := config.UrbitConf(patp)
				// check if existing ship was not created
				found := false
				for _, remote := range startramConfig.Subdomains {
					endpointUrl := strings.Split(conf.EndpointUrl, ".")
					if len(endpointUrl) < 2 {
						continue
					}
					rootUrl := strings.Join(endpointUrl[1:len(endpointUrl)], ".")
					if patp+"."+rootUrl == remote.URL {
						found = true
						break
					}
				}
				if !found {
					zap.L().Info(fmt.Sprintf("Registering missing StarTram service for %v", patp))
					startram.SvcCreate(patp, "urbit")
					startram.SvcCreate("s3."+patp, "minio")
				}
				for _, remote := range startramConfig.Subdomains {
					if remote.Status == string(transition.StartramServiceStatusCreating) {
						serviceCreated = false
					}
					parts := strings.Split(remote.URL, ".")
					if len(parts) < 2 {
						continue
					}
					// for urbit web
					subd := parts[0]
					if subd == patp && remote.SvcType == string(transition.StartramServiceTypeUrbitWeb) && remote.Status == string(transition.StartramServiceStatusOk) {
						// update alias
						if remote.Alias == "null" && local.CustomUrbitWeb != "" {
							zap.L().Debug(fmt.Sprintf("Retrieve: Resetting %v alias", patp))
							local.CustomUrbitWeb = ""
							modified = true
						} else if remote.Alias != local.CustomUrbitWeb {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v alias to %v", patp, remote.Alias))
							local.CustomUrbitWeb = remote.Alias
							modified = true
						}
						// update www port
						if remote.Port != local.WgHTTPPort {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v WG port to %v", patp, remote.Port))
							local.WgHTTPPort = remote.Port
							modified = true
						}
						// update remote url
						if remote.URL != local.WgURL {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v URL to %v", patp, remote.URL))
							local.WgURL = remote.URL
							modified = true
						}
						continue
					}
					// for urbit ames
					nestd := ""
					if len(parts) >= 2 {
						nestd = strings.Join(parts[:2], ".")
					}
					if nestd == "ames."+patp && remote.SvcType == string(transition.StartramServiceTypeUrbitAmes) && remote.Status == string(transition.StartramServiceStatusOk) {
						if remote.Port != local.WgAmesPort {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v ames port to %v", patp, remote.Port))
							local.WgAmesPort = remote.Port
							modified = true
						}
						continue
					}
					if nestd == "s3."+patp && remote.SvcType == string(transition.StartramServiceTypeMinio) && remote.Status == string(transition.StartramServiceStatusOk) {
						if remote.Port != local.WgS3Port {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v minio port to %v", patp, remote.Port))
							local.WgS3Port = remote.Port
							modified = true
						}
						continue
					}
					// for minio console
					consd := ""
					if len(parts) >= 3 {
						consd = strings.Join(parts[:3], ".")
					}
					if consd == "console.s3."+patp && remote.SvcType == string(transition.StartramServiceTypeMinioAdmin) && remote.Status == string(transition.StartramServiceStatusOk) {
						zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v console port to %v", patp, remote.Port))
						if remote.Port != local.WgConsolePort {
							local.WgConsolePort = remote.Port
							modified = true
						}
						continue
					}
				}
				if modified {
					if err := config.UpdateUrbit(patp, func(conf *structs.UrbitDocker) error {
						*conf = local
						return nil
					}); err != nil {
						zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s urbit config updates: %v", patp, err))
					}
				}
				publishUrbitServiceRegistrationTransition(patp, serviceCreated)
			}
		default:
		}
	}
}

func publishUrbitServiceRegistrationTransition(patp string, serviceCreated bool) {
	applyTransitionUpdate("urbit", func(current *structs.AuthBroadcast) {
		urbitStruct, ok := current.Urbits[patp]
		if !ok {
			return
		}
		if serviceCreated {
			urbitStruct.Transition.ServiceRegistrationStatus = string(transition.TransitionStatusEmpty)
		} else {
			urbitStruct.Transition.ServiceRegistrationStatus = string(transition.StartramServiceStatusCreating)
		}
		current.Urbits[patp] = urbitStruct
	})
}

func SystemTransitionHandler() {
	for {
		event := <-docker.SystemTransitions()
		applyTransitionUpdate("system", func(current *structs.AuthBroadcast) {
			if !setSystemTransition(&current.System.Transition, event) {
				zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", event.Type))
			}
		})
	}
}
