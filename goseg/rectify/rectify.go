package rectify

// this package is for watching event channels and rectifying mismatches
// between the desired and actual state, creating broadcast transitions,
// and anything else that needs to be done asyncronously

import (
	"context"
	"fmt"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

var (
	getStartramSettings  = config.StartramSettingsSnapshot
	getStartramConfigFn  = config.GetStartramConfig
	updateConfTypedFn    = config.UpdateConfTyped
	getContainerStatesFn = config.GetContainerState
	getUrbitConfigsFn    = config.UrbitConfAll
	getUrbitConfigFn     = config.UrbitConf
	loadUrbitConfigFn    = config.LoadUrbitConfig
	updateUrbitConfigFn  = config.UpdateUrbit
)

func UrbitTransitionHandler() {
	for {
		event := <-events.UrbitTransitions()
		applyTransitionUpdate("urbit", urbitTransitionCommand{event: event})
	}
}

func UrbitTransitionHandlerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events.UrbitTransitions():
			applyTransitionUpdate("urbit", urbitTransitionCommand{event: event})
		}
	}
}

type urbitTransitionCommand struct {
	event structs.UrbitTransition
}

func (command urbitTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	urbitStruct, exists := current.Urbits[command.event.Patp]
	if !exists {
		return nil
	}
	if !setUrbitTransition(&urbitStruct.Transition, command.event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", command.event.Type))
		return nil
	}
	current.Urbits[command.event.Patp] = urbitStruct
	return nil
}

type transitionReducer[K comparable, T any, E any] func(*T, E) bool

func setByTransition[K comparable, T any, E any](
	reducers map[K]transitionReducer[K, T, E],
	target *T,
	key K,
	event E,
) bool {
	if reducer, ok := reducers[key]; ok {
		return reducer(target, event)
	}
	return false
}

var urbitTransitionReducers = map[transition.UrbitTransitionType]transitionReducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition]{
	transition.UrbitTransitionRollChop: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.RollChop = event.Event
		return true
	},
	transition.UrbitTransitionChopOnUpgrade: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ChopOnUpgrade = event.Event
		return true
	},
	transition.UrbitTransitionChop: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.Chop = event.Event
		return true
	},
	transition.UrbitTransitionPack: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.Pack = event.Event
		return true
	},
	transition.UrbitTransitionPackMeld: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.PackMeld = strconv.FormatInt(int64(event.Value), 10)
		return true
	},
	transition.UrbitTransitionLoom: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.Loom = event.Event
		return true
	},
	transition.UrbitTransitionSnapTime: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.SnapTime = event.Event
		return true
	},
	transition.UrbitTransitionUrbitDomain: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.UrbitDomain = event.Event
		return true
	},
	transition.UrbitTransitionMinIODomain: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.MinIODomain = event.Event
		return true
	},
	transition.UrbitTransitionRebuildContainer: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.RebuildContainer = event.Event
		return true
	},
	transition.UrbitTransitionToggleDevMode: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ToggleDevMode = event.Event
		return true
	},
	transition.UrbitTransitionTogglePower: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.TogglePower = event.Event
		return true
	},
	transition.UrbitTransitionToggleNetwork: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ToggleNetwork = event.Event
		return true
	},
	transition.UrbitTransitionExportShip: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ExportShip = event.Event
		return true
	},
	transition.UrbitTransitionShipCompressed: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ShipCompressed = event.Value
		return true
	},
	transition.UrbitTransitionExportBucket: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ExportBucket = event.Event
		return true
	},
	transition.UrbitTransitionBucketCompressed: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.BucketCompressed = event.Value
		return true
	},
	transition.UrbitTransitionDeleteShip: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.DeleteShip = event.Event
		return true
	},
	transition.UrbitTransitionToggleMinIOLink: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ToggleMinIOLink = event.Event
		return true
	},
	transition.UrbitTransitionPenpaiCompanion: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.PenpaiCompanion = event.Event
		return true
	},
	transition.UrbitTransitionGallseg: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.Gallseg = event.Event
		return true
	},
	transition.UrbitTransitionDeleteService: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.StartramServices = event.Event
		return true
	},
	transition.UrbitTransitionLocalTlonBackupsEnabled: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.LocalTlonBackupsEnabled = event.Event
		return true
	},
	transition.UrbitTransitionRemoteTlonBackupsEnabled: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.RemoteTlonBackupsEnabled = event.Event
		return true
	},
	transition.UrbitTransitionLocalTlonBackup: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.LocalTlonBackup = event.Event
		return true
	},
	transition.UrbitTransitionLocalTlonBackupSchedule: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.LocalTlonBackupSchedule = event.Event
		return true
	},
	transition.UrbitTransitionHandleRestoreTlonBackup: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.HandleRestoreTlonBackup = event.Event
		return true
	},
	transition.UrbitTransitionServiceRegistrationStatus: func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		state.ServiceRegistrationStatus = event.Event
		return true
	},
}

func setUrbitTransition(transitionState *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
	return setByTransition(urbitTransitionReducers, transitionState, transition.UrbitTransitionType(event.Type), event)
}

func NewShipTransitionHandler() {
	for {
		event := <-events.NewShipTransitions()
		applyTransitionUpdate("new ship", newShipTransitionCommand{event: event})
	}
}

func NewShipTransitionHandlerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events.NewShipTransitions():
			applyTransitionUpdate("new ship", newShipTransitionCommand{event: event})
		}
	}
}

type newShipTransitionCommand struct {
	event structs.NewShipTransition
}

func (command newShipTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	if !setNewShipTransition(&current.NewShip, command.event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", command.event.Type))
	}
	return nil
}

var newShipTransitionReducers = map[transition.NewShipTransitionType]transitionReducer[transition.NewShipTransitionType, structs.NewShip, structs.NewShipTransition]{
	transition.NewShipTransitionError: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		target.Transition.Error = event.Event
		return true
	},
	transition.NewShipTransitionBootStage: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		// Events
		// starting: setting up docker and config
		// creating: actually create and start the container
		// booting: waiting until +code shows up
		// completed: ready to reset
		// aborted: something went wrong and we ran the cleanup routine
		// <empty>: free for new ship
		target.Transition.BootStage = event.Event
		return true
	},
	transition.NewShipTransitionPatp: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		target.Transition.Patp = event.Event
		return true
	},
	transition.NewShipTransitionFreeError: func(target *structs.NewShip, event structs.NewShipTransition) bool {
		target.Transition.FreeError = event.Event
		return true
	},
}

func setNewShipTransition(target *structs.NewShip, event structs.NewShipTransition) bool {
	return setByTransition(newShipTransitionReducers, target, transition.NewShipTransitionType(event.Type), event)
}

var systemTransitionReducers = map[transition.SystemTransitionType]transitionReducer[transition.SystemTransitionType, structs.SystemTransitionBroadcast, structs.SystemTransition]{
	transition.SystemTransitionWifiConnect: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.WifiConnect = event.Event
		return true
	},
	transition.SystemTransitionSwap: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.Swap = event.BoolEvent
		return true
	},
	transition.SystemTransitionBugReport: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.BugReport = event.Event
		return true
	},
	transition.SystemTransitionBugReportError: func(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
		target.BugReportError = event.Event
		return true
	},
}

func setSystemTransition(target *structs.SystemTransitionBroadcast, event structs.SystemTransition) bool {
	return setByTransition(systemTransitionReducers, target, transition.SystemTransitionType(event.Type), event)
}

var startramTransitionReducers = map[transition.EventType]transitionReducer[transition.EventType, structs.StartramTransition, any]{
	transition.StartramTransitionRestart: func(target *structs.StartramTransition, eventData any) bool {
		target.Restart = fmt.Sprintf("%v", eventData)
		return true
	},
	transition.StartramTransitionEndpoint: func(target *structs.StartramTransition, eventData any) bool {
		if eventData == nil {
			target.Endpoint = ""
			return true
		}
		target.Endpoint = fmt.Sprintf("%v", eventData)
		return true
	},
	transition.StartramTransitionToggle: func(target *structs.StartramTransition, eventData any) bool {
		target.Toggle = eventData
		return true
	},
	transition.StartramTransitionRegister: func(target *structs.StartramTransition, eventData any) bool {
		target.Register = eventData
		return true
	},
}

func setStartramTransition(target *structs.StartramTransition, eventType string, eventData any) bool {
	return setByTransition(startramTransitionReducers, target, transition.EventType(eventType), eventData)
}

func applyTransitionUpdate(context string, transition broadcast.BroadcastTransition) {
	if err := broadcast.ApplyBroadcastTransition(true, transition); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to publish %s transition update: %v", context, err))
	}
}

func RectifyUrbit() {
	for {
		event := <-startram.Events()
		switch transition.EventType(event.Type) {
		case transition.StartramTransitionRestart:
			applyTransitionUpdate("startram", startramTransitionCommand{event: event})
		case transition.StartramTransitionEndpoint:
			applyTransitionUpdate("startram", startramTransitionCommand{event: event})
		case transition.StartramTransitionToggle:
			applyTransitionUpdate("startram", startramTransitionCommand{event: event})
		case transition.StartramTransitionRegister:
			applyTransitionUpdate("startram", startramTransitionCommand{event: event})
		case transition.StartramTransitionRetrieve:
			applyTransitionUpdate("startram", startramRetrieveTransition{})
		default:
		}
	}
}

func RectifyUrbitWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-startram.Events():
			switch transition.EventType(event.Type) {
			case transition.StartramTransitionRestart:
				applyTransitionUpdate("startram", startramTransitionCommand{event: event})
			case transition.StartramTransitionEndpoint:
				applyTransitionUpdate("startram", startramTransitionCommand{event: event})
			case transition.StartramTransitionToggle:
				applyTransitionUpdate("startram", startramTransitionCommand{event: event})
			case transition.StartramTransitionRegister:
				applyTransitionUpdate("startram", startramTransitionCommand{event: event})
			case transition.StartramTransitionRetrieve:
				applyTransitionUpdate("startram", startramRetrieveTransition{})
			default:
			}
		}
	}
}

type startramTransitionCommand struct {
	event structs.Event
}

func (command startramTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	if !setStartramTransition(&current.Profile.Startram.Transition, string(command.event.Type), command.event.Data) {
		return nil
	}
	switch transition.EventType(command.event.Type) {
	case transition.StartramTransitionEndpoint:
		if command.event.Data == transition.StartramTransitionComplete {
			settings := getStartramSettings()
			current.Profile.Startram.Info.Endpoint = settings.EndpointURL
		}
	case transition.StartramTransitionRegister:
		if command.event.Data != transition.StartramTransitionComplete {
			return nil
		}
		settings := getStartramSettings()
		current.Profile.Startram.Info.Running = settings.WgOn
		containerState, exists := getContainerStatesFn()[string(transition.ContainerTypeWireguard)]
		if exists {
			running := containerState.ActualStatus == string(transition.ContainerStatusRunning)
			current.Profile.Startram.Info.Running = running
			if err := updateConfTypedFn(config.WithWgOn(running)); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}
		current.Profile.Startram.Info.Registered = settings.WgRegistered
	}
	return nil
}

type startramRetrieveTransition struct{}

func (startramRetrieveTransition) Apply(current *structs.AuthBroadcast) error {
	startramSettings := getStartramSettings()
	startramConfig := getStartramConfigFn()
	for patp := range getUrbitConfigsFn() {
		modified := false
		serviceCreated := true
		local := getUrbitConfigFn(patp)
		loadUrbitConfigFn(patp)

		found := false
		for _, remote := range startramConfig.Subdomains {
			endpointUrl := strings.Split(startramSettings.EndpointURL, ".")
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
			subd := parts[0]
			if subd == patp && remote.SvcType == string(transition.StartramServiceTypeUrbitWeb) && remote.Status == string(transition.StartramServiceStatusOk) {
				if remote.Alias == "null" && local.CustomUrbitWeb != "" {
					zap.L().Debug(fmt.Sprintf("Retrieve: Resetting %v alias", patp))
					local.CustomUrbitWeb = ""
					modified = true
				} else if remote.Alias != local.CustomUrbitWeb {
					zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v alias to %v", patp, remote.Alias))
					local.CustomUrbitWeb = remote.Alias
					modified = true
				}
				if remote.Port != local.WgHTTPPort {
					zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v WG port to %v", patp, remote.Port))
					local.WgHTTPPort = remote.Port
					modified = true
				}
				if remote.URL != local.WgURL {
					zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v URL to %v", patp, remote.URL))
					local.WgURL = remote.URL
					modified = true
				}
				continue
			}

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
			if err := updateUrbitConfigFn(patp, func(conf *structs.UrbitDocker) error {
				*conf = local
				return nil
			}); err != nil {
				zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s urbit config updates: %v", patp, err))
			}
		}
		publishUrbitServiceRegistrationTransitionWithCurrentState(current, patp, serviceCreated)
	}
	return nil
}

func publishUrbitServiceRegistrationTransition(patp string, serviceCreated bool) {
	applyTransitionUpdate("urbit", urbitServiceRegistrationTransitionCommand{
		patp:          patp,
		serviceStatus: serviceCreated,
	})
}

type urbitServiceRegistrationTransitionCommand struct {
	patp          string
	serviceStatus bool
}

func (command urbitServiceRegistrationTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	publishUrbitServiceRegistrationTransitionWithCurrentState(current, command.patp, command.serviceStatus)
	return nil
}

func publishUrbitServiceRegistrationTransitionWithCurrentState(current *structs.AuthBroadcast, patp string, serviceCreated bool) {
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
}

func SystemTransitionHandler() {
	for {
		event := <-events.SystemTransitions()
		applyTransitionUpdate("system", systemTransitionCommand{event: event})
	}
}

func SystemTransitionHandlerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events.SystemTransitions():
			applyTransitionUpdate("system", systemTransitionCommand{event: event})
		}
	}
}

type systemTransitionCommand struct {
	event structs.SystemTransition
}

func (command systemTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	if !setSystemTransition(&current.System.Transition, command.event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", command.event.Type))
	}
	return nil
}
