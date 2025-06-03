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
	"strings"

	"go.uber.org/zap"
)

func UrbitTransitionHandler() {
	for {
		event := <-docker.UTransBus
		current := broadcast.GetState()
		urbitStruct, exists := current.Urbits[event.Patp]
		if exists {
			switch event.Type {
			case "rollChop":
				urbitStruct.Transition.RollChop = event.Event
			case "chopOnUpgrade":
				urbitStruct.Transition.ChopOnUpgrade = event.Event
			case "chop":
				urbitStruct.Transition.Chop = event.Event
			case "pack":
				urbitStruct.Transition.Pack = event.Event
			case "packMeld":
				urbitStruct.Transition.PackMeld = event.Event
			case "loom":
				urbitStruct.Transition.Loom = event.Event
			case "snapTime":
				urbitStruct.Transition.SnapTime = event.Event
			case "urbitDomain":
				urbitStruct.Transition.UrbitDomain = event.Event
			case "minioDomain":
				urbitStruct.Transition.MinIODomain = event.Event
			case "rebuildContainer":
				urbitStruct.Transition.RebuildContainer = event.Event
			case "toggleDevMode":
				urbitStruct.Transition.ToggleDevMode = event.Event
			case "togglePower":
				urbitStruct.Transition.TogglePower = event.Event
			case "toggleNetwork":
				urbitStruct.Transition.ToggleNetwork = event.Event
			case "exportShip":
				urbitStruct.Transition.ExportShip = event.Event
			case "shipCompressed":
				urbitStruct.Transition.ShipCompressed = event.Value
			case "exportBucket":
				urbitStruct.Transition.ExportBucket = event.Event
			case "bucketCompressed":
				urbitStruct.Transition.BucketCompressed = event.Value
			case "deleteShip":
				urbitStruct.Transition.DeleteShip = event.Event
			case "toggleMinIOLink":
				urbitStruct.Transition.ToggleMinIOLink = event.Event
			case "penpaiCompanion":
				urbitStruct.Transition.PenpaiCompanion = event.Event
			case "gallseg":
				urbitStruct.Transition.Gallseg = event.Event
			case "deleteService":
				urbitStruct.Transition.StartramServices = event.Event
			case "localTlonBackupsEnabled":
				urbitStruct.Transition.LocalTlonBackupsEnabled = event.Event
			case "remoteTlonBackupsEnabled":
				urbitStruct.Transition.RemoteTlonBackupsEnabled = event.Event
			case "localTlonBackup":
				urbitStruct.Transition.LocalTlonBackup = event.Event
			case "localTlonBackupSchedule":
				urbitStruct.Transition.LocalTlonBackupSchedule = event.Event
			case "handleRestoreTlonBackup":
				urbitStruct.Transition.HandleRestoreTlonBackup = event.Event
			default:
				zap.L().Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
				continue
			}
			current.Urbits[event.Patp] = urbitStruct
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		}
	}
}

func NewShipTransitionHandler() {
	for {
		event := <-docker.NewShipTransBus
		switch event.Type {
		case "error":
			current := broadcast.GetState()
			current.NewShip.Transition.Error = event.Event
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "bootStage":
			// Events
			// starting: setting up docker and config
			// creating: actually create and start the container
			// booting: waiting until +code shows up
			// completed: ready to reset
			// aborted: something went wrong and we ran the cleanup routine
			// <empty>: free for new ship
			current := broadcast.GetState()
			current.NewShip.Transition.BootStage = event.Event
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "patp":
			current := broadcast.GetState()
			current.NewShip.Transition.Patp = event.Event
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "freeError":
			current := broadcast.GetState()
			current.NewShip.Transition.FreeError = event.Event
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		default:
			zap.L().Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
		}
	}
}

func RectifyUrbit() {
	for {
		event := <-startram.EventBus
		switch event.Type {
		case "restart":
			// startram - restarting wireguard container
			// urbits - recreating urbit containers
			// minios - recreating minio containers
			// done - completed
			current := broadcast.GetState()
			current.Profile.Startram.Transition.Restart = fmt.Sprintf("%v", event.Data)
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "endpoint":
			//init - started
			// unregistering - startram services unregistering
			// stopping - wireguard stopping
			// configuring - reset pubkey
			// finalizing - modifying endpoint
			// complete - successfully changed endpoint
			// Error: <text> - Error with info
			// nil/null - Empty, ready for action
			current := broadcast.GetState()
			if event.Data != nil {
				current.Profile.Startram.Transition.Endpoint = fmt.Sprintf("%v", event.Data)
			} else {
				current.Profile.Startram.Transition.Endpoint = ""
			}
			if event.Data == "complete" {
				conf := config.Conf()
				current.Profile.Startram.Info.Endpoint = conf.EndpointUrl
			}
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "toggle":
			// loading - loading
			// nil/null - Empty
			current := broadcast.GetState()
			current.Profile.Startram.Transition.Toggle = event.Data
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "register":
			// key - registering startram key
			// services - registering startram services
			// starting - applying wg0.conf and starting container
			// complete - successfully finished registering
			// Error: <text> - Error with info
			// nil/null - Empty, ready for action
			current := broadcast.GetState()
			current.Profile.Startram.Transition.Register = event.Data
			if event.Data == "complete" {
				conf := config.Conf()
				current.Profile.Startram.Info.Running = conf.WgOn
				containerState, exists := config.GetContainerState()["wireguard"]
				if exists {
					running := containerState.ActualStatus == "running"
					current.Profile.Startram.Info.Running = running
					if err := config.UpdateConf(map[string]interface{}{"wgOn": running}); err != nil {
						zap.L().Error(fmt.Sprintf("%v", err))
					}
				}
				current.Profile.Startram.Info.Registered = conf.WgRegistered
			}
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "retrieve":
			conf := config.Conf()
			for patp, _ := range config.UrbitConfAll() {
				modified := false
				serviceCreated := true
				startramConfig := config.StartramConfig // a structs.StartramRetrieve
				config.LoadUrbitConfig(patp)
				local := config.UrbitConf(patp) // a structs.UrbitDocker
				// check if existing ship was not created
				found := false
				for _, remote := range startramConfig.Subdomains {
					endpointUrl := strings.Split(conf.EndpointUrl, ".")
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
					if remote.Status == "creating" {
						serviceCreated = false
					}
					// for urbit web
					subd := strings.Split(remote.URL, ".")[0]
					if subd == patp && remote.SvcType == "urbit-web" && remote.Status == "ok" {
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
					nestd := strings.Join(strings.Split(remote.URL, ".")[:2], ".")
					if nestd == "ames."+patp && remote.SvcType == "urbit-ames" && remote.Status == "ok" {
						if remote.Port != local.WgAmesPort {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v ames port to %v", patp, remote.Port))
							local.WgAmesPort = remote.Port
							modified = true
						}
						continue
					}
					// for minio
					if nestd == "s3."+patp && remote.SvcType == "minio" && remote.Status == "ok" {
						if remote.Port != local.WgS3Port {
							zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v minio port to %v", patp, remote.Port))
							local.WgS3Port = remote.Port
							modified = true
						}
						continue
					}
					// for minio console
					consd := strings.Join(strings.Split(remote.URL, ".")[:3], ".")
					if consd == "console.s3."+patp && remote.SvcType == "minio-console" && remote.Status == "ok" {
						zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v console port to %v", patp, remote.Port))
						if remote.Port != local.WgConsolePort {
							local.WgConsolePort = remote.Port
							modified = true
						}
						continue
					}
				}
				if modified {
					config.UpdateUrbitConfig(map[string]structs.UrbitDocker{patp: local})
				}
				current := broadcast.GetState()
				urbitStruct, ok := current.Urbits[patp]
				if ok {
					if serviceCreated {
						urbitStruct.Transition.ServiceRegistrationStatus = ""
					} else {
						urbitStruct.Transition.ServiceRegistrationStatus = "creating"
					}

					// Put the modified struct back into the map
					current.Urbits[patp] = urbitStruct
				}
				broadcast.UpdateBroadcast(current)
				broadcast.BroadcastToClients()
			}
		default:
		}
	}
}

func SystemTransitionHandler() {
	for {
		event := <-docker.SysTransBus
		current := broadcast.GetState()
		switch event.Type {
		case "wifiConnect":
			current.System.Transition.WifiConnect = event.Event
		case "swap":
			current.System.Transition.Swap = event.BoolEvent
		case "bugReport":
			current.System.Transition.BugReport = event.Event
			broadcast.UpdateBroadcast(current)
			broadcast.BroadcastToClients()
		case "bugReportError":
			current.System.Transition.BugReportError = event.Event
		default:
			zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", event.Type))
		}
		broadcast.UpdateBroadcast(current)
		broadcast.BroadcastToClients()
	}
}
