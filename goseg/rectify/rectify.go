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
	"groundseg/internal/transitionlifecycle"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type RectifyRuntime struct {
	StartramSettingsSnapshotFn func() config.StartramSettings
	GetStartramConfigFn        func() structs.StartramRetrieve
	UpdateConfTypedFn          func(...config.ConfUpdateOption) error
	GetContainerStateFn        func() map[string]structs.ContainerState
	UrbitConfAllFn             func() map[string]structs.UrbitDocker
	UrbitConfFn                func(string) structs.UrbitDocker
	LoadUrbitConfigFn          func(string) error
	UpdateUrbitFn              func(string, func(*structs.UrbitDocker) error) error
}

type RectifyRuntimeDependencies struct {
	StartramSettingsSnapshotFn func() config.StartramSettings
	GetStartramConfigFn        func() structs.StartramRetrieve
	UpdateConfTypedFn          func(...config.ConfUpdateOption) error
	GetContainerStateFn        func() map[string]structs.ContainerState
	UrbitConfAllFn             func() map[string]structs.UrbitDocker
	UrbitConfFn                func(string) structs.UrbitDocker
	LoadUrbitConfigFn          func(string) error
	UpdateUrbitFn              func(string, func(*structs.UrbitDocker) error) error
}

func NewRectifyRuntime() RectifyRuntime {
	return NewRectifyRuntimeWithDependencies(defaultRectifyRuntimeDependencies())
}

func NewRectifyRuntimeWithDependencies(dependencies RectifyRuntimeDependencies) RectifyRuntime {
	return RectifyRuntime{
		StartramSettingsSnapshotFn: dependencies.StartramSettingsSnapshotFn,
		GetStartramConfigFn:        dependencies.GetStartramConfigFn,
		UpdateConfTypedFn:          dependencies.UpdateConfTypedFn,
		GetContainerStateFn:        dependencies.GetContainerStateFn,
		UrbitConfAllFn:             dependencies.UrbitConfAllFn,
		UrbitConfFn:                dependencies.UrbitConfFn,
		LoadUrbitConfigFn:          dependencies.LoadUrbitConfigFn,
		UpdateUrbitFn:              dependencies.UpdateUrbitFn,
	}
}

func defaultRectifyRuntimeDependencies() RectifyRuntimeDependencies {
	return RectifyRuntimeDependencies{
		StartramSettingsSnapshotFn: config.StartramSettingsSnapshot,
		GetStartramConfigFn:        config.GetStartramConfig,
		UpdateConfTypedFn:          config.UpdateConfTyped,
		GetContainerStateFn:        config.GetContainerState,
		UrbitConfAllFn:             config.UrbitConfAll,
		UrbitConfFn:                config.UrbitConf,
		LoadUrbitConfigFn:          config.LoadUrbitConfig,
		UpdateUrbitFn:              config.UpdateUrbit,
	}
}

func (runtime RectifyRuntime) startramSettings() config.StartramSettings {
	if runtime.StartramSettingsSnapshotFn == nil {
		return config.StartramSettingsSnapshot()
	}
	return runtime.StartramSettingsSnapshotFn()
}

func UrbitTransitionHandler() {
	if err := runTransitionEventLoopWithoutContext("urbit", events.UrbitTransitions(), func(event structs.UrbitTransition) broadcast.BroadcastTransition {
		return urbitTransitionCommand{event: event}
	}); err != nil {
		zap.L().Warn(fmt.Sprintf("urbit transition handler stopped with error: %v", err))
	}
}

func UrbitTransitionHandlerWithContext(ctx context.Context) error {
	return runTransitionEventLoop(ctx, "urbit", events.UrbitTransitions(), func(event structs.UrbitTransition) broadcast.BroadcastTransition {
		return urbitTransitionCommand{event: event}
	})
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

var urbitTransitionReducers = map[transition.UrbitTransitionType]transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition]{
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
	return transitionlifecycle.ApplyReducer(
		urbitTransitionReducers,
		transitionState,
		transition.UrbitTransitionType(event.Type),
		event,
	)
}

func NewShipTransitionHandler() {
	if err := runTransitionEventLoopWithoutContext("new ship", events.NewShipTransitions(), func(event structs.NewShipTransition) broadcast.BroadcastTransition {
		return newShipTransitionCommand{event: event}
	}); err != nil {
		zap.L().Warn(fmt.Sprintf("new ship transition handler stopped with error: %v", err))
	}
}

func NewShipTransitionHandlerWithContext(ctx context.Context) error {
	return runTransitionEventLoop(ctx, "new ship", events.NewShipTransitions(), func(event structs.NewShipTransition) broadcast.BroadcastTransition {
		return newShipTransitionCommand{event: event}
	})
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

var newShipTransitionReducers = map[transition.NewShipTransitionType]transitionlifecycle.Reducer[transition.NewShipTransitionType, structs.NewShip, structs.NewShipTransition]{
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
	return transitionlifecycle.ApplyReducer(
		newShipTransitionReducers,
		target,
		transition.NewShipTransitionType(event.Type),
		event,
	)
}

var systemTransitionReducers = map[transition.SystemTransitionType]transitionlifecycle.Reducer[transition.SystemTransitionType, structs.SystemTransitionBroadcast, structs.SystemTransition]{
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
	return transitionlifecycle.ApplyReducer(
		systemTransitionReducers,
		target,
		transition.SystemTransitionType(event.Type),
		event,
	)
}

var startramTransitionReducers = map[transition.EventType]transitionlifecycle.Reducer[transition.EventType, structs.StartramTransition, any]{
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
	return transitionlifecycle.ApplyReducer(
		startramTransitionReducers,
		target,
		transition.EventType(eventType),
		eventData,
	)
}

func applyTransitionUpdate(context string, transition broadcast.BroadcastTransition) {
	if err := broadcast.ApplyBroadcastTransition(true, transition); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to publish %s transition update: %v", context, err))
	}
}

func RectifyUrbit() {
	runtime := NewRectifyRuntime()
	if err := runTransitionEventLoopWithoutContext("startram", startram.Events(), func(event structs.Event) broadcast.BroadcastTransition {
		transitionType := transition.EventType(event.Type)
		switcher, ok := rectifyStartramTransitionRouters[transitionType]
		if !ok {
			return nil
		}
		return switcher(runtime, event)
	}); err != nil {
		zap.L().Warn(fmt.Sprintf("startram rectify handler stopped with error: %v", err))
	}
}

func RectifyUrbitWithContext(ctx context.Context) error {
	runtime := NewRectifyRuntime()
	return runTransitionEventLoop(ctx, "startram", startram.Events(), func(event structs.Event) broadcast.BroadcastTransition {
		transitionType := transition.EventType(event.Type)
		switcher, ok := rectifyStartramTransitionRouters[transitionType]
		if !ok {
			return nil
		}
		return switcher(runtime, event)
	})
}

type rectifyStartramTransitionRouter func(RectifyRuntime, structs.Event) broadcast.BroadcastTransition

var rectifyStartramTransitionRouters = map[transition.EventType]rectifyStartramTransitionRouter{
	transition.StartramTransitionRestart: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{event: event, runtime: runtime}
	},
	transition.StartramTransitionEndpoint: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{event: event, runtime: runtime}
	},
	transition.StartramTransitionToggle: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{event: event, runtime: runtime}
	},
	transition.StartramTransitionRegister: func(runtime RectifyRuntime, event structs.Event) broadcast.BroadcastTransition {
		return startramTransitionCommand{event: event, runtime: runtime}
	},
	transition.StartramTransitionRetrieve: func(runtime RectifyRuntime, _ structs.Event) broadcast.BroadcastTransition {
		return startramRetrieveTransition{runtime: runtime}
	},
}

type startramTransitionCommand struct {
	event   structs.Event
	runtime RectifyRuntime
}

func (command startramTransitionCommand) Apply(current *structs.AuthBroadcast) error {
	if !setStartramTransition(&current.Profile.Startram.Transition, string(command.event.Type), command.event.Data) {
		return nil
	}
	switch transition.EventType(command.event.Type) {
	case transition.StartramTransitionEndpoint:
		if command.event.Data == transition.StartramTransitionComplete {
			settings := command.runtime.startramSettings()
			current.Profile.Startram.Info.Endpoint = settings.EndpointURL
		}
	case transition.StartramTransitionRegister:
		if command.event.Data != transition.StartramTransitionComplete {
			return nil
		}
		settings := command.runtime.startramSettings()
		current.Profile.Startram.Info.Running = settings.WgOn
		containerState, exists := command.runtime.GetContainerStateFn()[string(transition.ContainerTypeWireguard)]
		if exists {
			running := containerState.ActualStatus == string(transition.ContainerStatusRunning)
			current.Profile.Startram.Info.Running = running
			if err := command.runtime.UpdateConfTypedFn(config.WithWgOn(running)); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}
		current.Profile.Startram.Info.Registered = settings.WgRegistered
	}
	return nil
}

type startramRetrieveTransition struct {
	runtime RectifyRuntime
}

func (transitionCommand startramRetrieveTransition) Apply(current *structs.AuthBroadcast) error {
	runtime := transitionCommand.runtime
	startramSettings := runtime.startramSettings()
	startramConfig := runtime.GetStartramConfigFn()
	for patp := range runtime.UrbitConfAllFn() {
		modified := false
		serviceCreated := true
		local := runtime.UrbitConfFn(patp)
		runtime.LoadUrbitConfigFn(patp)

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
			if err := runtime.UpdateUrbitFn(patp, func(conf *structs.UrbitDocker) error {
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

func runTransitionEventLoop[T any](ctx context.Context, label string, ch <-chan T, mapEvent func(T) broadcast.BroadcastTransition) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-ch:
			command := mapEvent(event)
			if command == nil {
				continue
			}
			applyTransitionUpdate(label, command)
		}
	}
}

func runTransitionEventLoopWithoutContext[T any](label string, ch <-chan T, mapEvent func(T) broadcast.BroadcastTransition) error {
	return runTransitionEventLoop(context.Background(), label, ch, mapEvent)
}
