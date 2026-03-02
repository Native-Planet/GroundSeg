package rectify

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
)

type UrbitTransitionApplier struct{}

func (applier UrbitTransitionApplier) Apply(current *structs.AuthBroadcast, event structs.UrbitTransition) error {
	urbitStruct, exists := current.Urbits[event.Patp]
	if !exists {
		return nil
	}
	if !setUrbitTransition(&urbitStruct.Transition, event) {
		zap.L().Warn(fmt.Sprintf("Unrecognized transition: %v", event.Type))
		return nil
	}
	current.Urbits[event.Patp] = urbitStruct
	return nil
}

type StartramTransitionService struct {
	runtime RectifyRuntime
}

func NewStartramTransitionService(runtime RectifyRuntime) StartramTransitionService {
	return StartramTransitionService{
		runtime: runtime,
	}
}

func (service StartramTransitionService) Apply(current *structs.AuthBroadcast, event structs.Event) error {
	if !setStartramTransition(&current.Profile.Startram.Transition, string(event.Type), event.Data) {
		return nil
	}
	return service.applyTransitionCompletion(current, transition.EventType(event.Type), event.Data)
}

func (service StartramTransitionService) applyTransitionCompletion(current *structs.AuthBroadcast, eventType transition.EventType, eventData any) error {
	switch eventType {
	case transition.StartramTransitionEndpoint:
		if eventData != transition.StartramTransitionComplete {
			return nil
		}
		settings := service.runtime.startramSettings()
		current.Profile.Startram.Info.Endpoint = settings.EndpointURL
	case transition.StartramTransitionRegister:
		if eventData != transition.StartramTransitionComplete {
			return nil
		}
		settings := service.runtime.startramSettings()
		current.Profile.Startram.Info.Running = settings.WgOn
		containerState, exists := service.runtime.GetContainerStateFn()[string(transition.ContainerTypeWireguard)]
		if exists {
			running := containerState.ActualStatus == string(transition.ContainerStatusRunning)
			current.Profile.Startram.Info.Running = running
			if err := service.runtime.UpdateConfTypedFn(config.WithWgOn(running)); err != nil {
				zap.L().Error(fmt.Sprintf("%v", err))
			}
		}
		current.Profile.Startram.Info.Registered = settings.WgRegistered
	}
	return nil
}

type StartramRetrieveReconciler struct {
	runtime RectifyRuntime
}

func NewStartramRetrieveReconciler(runtime RectifyRuntime) *StartramRetrieveReconciler {
	return &StartramRetrieveReconciler{runtime: runtime}
}

func (reconciler *StartramRetrieveReconciler) Reconcile(current *structs.AuthBroadcast) error {
	runtime := reconciler.runtime
	startramSettings := runtime.startramSettings()
	startramConfig := runtime.GetStartramConfigFn()
	for patp := range runtime.UrbitConfAllFn() {
		modified := false
		serviceCreated := true
		local := runtime.UrbitConfFn(patp)
		runtime.LoadUrbitConfigFn(patp)
		persistNetwork := false
		persistWeb := false

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
			if reconciler.reconcileUrbitWebService(patp, remote, &local, &modified, &persistWeb, &persistNetwork) {
				continue
			}
			if reconciler.reconcileUrbitNetworkServices(patp, remote, parts, &local, &modified, &persistNetwork) {
				continue
			}
		}

		if modified {
			if persistWeb {
				if runtime.UpdateUrbitWebConfigFn == nil {
					zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s web config updates: updateUrbitWebConfigFn dependency not configured", patp))
				} else if err := runtime.UpdateUrbitWebConfigFn(patp, func(config *structs.UrbitWebConfig) error {
					config.CustomUrbitWeb = local.CustomUrbitWeb
					return nil
				}); err != nil {
					zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s web config updates: %v", patp, err))
				}
			}

			if persistNetwork {
				if runtime.UpdateUrbitNetworkConfigFn == nil {
					zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s network config updates: updateUrbitNetworkConfigFn dependency not configured", patp))
				} else if err := runtime.UpdateUrbitNetworkConfigFn(patp, func(config *structs.UrbitNetworkConfig) error {
					config.WgHTTPPort = local.WgHTTPPort
					config.WgAmesPort = local.WgAmesPort
					config.WgS3Port = local.WgS3Port
					config.WgConsolePort = local.WgConsolePort
					config.WgURL = local.WgURL
					config.Network = local.Network
					return nil
				}); err != nil {
					zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist %s network config updates: %v", patp, err))
				}
			}
		}
		publishUrbitServiceRegistrationTransitionWithCurrentState(current, patp, serviceCreated)
	}
	return nil
}

func (reconciler *StartramRetrieveReconciler) reconcileUrbitWebService(patp string, remote structs.Subdomain, local *structs.UrbitDocker, modified *bool, persistWeb *bool, persistNetwork *bool) bool {
	parts := strings.Split(remote.URL, ".")
	if len(parts) < 2 {
		return false
	}
	subd := parts[0]
	if subd != patp || remote.SvcType != string(transition.StartramServiceTypeUrbitWeb) || remote.Status != string(transition.StartramServiceStatusOk) {
		return false
	}

	if remote.Alias == "null" && local.CustomUrbitWeb != "" {
		zap.L().Debug(fmt.Sprintf("Retrieve: Resetting %v alias", patp))
		local.CustomUrbitWeb = ""
		*persistWeb = true
		*modified = true
	} else if remote.Alias != local.CustomUrbitWeb {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v alias to %v", patp, remote.Alias))
		local.CustomUrbitWeb = remote.Alias
		*persistWeb = true
		*modified = true
	}
	if remote.Port != local.WgHTTPPort {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v WG port to %v", patp, remote.Port))
		local.WgHTTPPort = remote.Port
		*persistNetwork = true
		*modified = true
	}
	if remote.URL != local.WgURL {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v URL to %v", patp, remote.URL))
		local.WgURL = remote.URL
		*persistNetwork = true
		*modified = true
	}
	return true
}

func (reconciler *StartramRetrieveReconciler) reconcileUrbitNetworkServices(patp string, remote structs.Subdomain, urlParts []string, local *structs.UrbitDocker, modified *bool, persistNetwork *bool) bool {
	nestd := ""
	if len(urlParts) >= 2 {
		nestd = strings.Join(urlParts[:2], ".")
	}
	switch {
	case nestd == "ames."+patp && remote.SvcType == string(transition.StartramServiceTypeUrbitAmes) && remote.Status == string(transition.StartramServiceStatusOk):
		if remote.Port != local.WgAmesPort {
			zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v ames port to %v", patp, remote.Port))
			local.WgAmesPort = remote.Port
			*persistNetwork = true
			*modified = true
		}
		return true
	case nestd == "s3."+patp && remote.SvcType == string(transition.StartramServiceTypeMinio) && remote.Status == string(transition.StartramServiceStatusOk):
		if remote.Port != local.WgS3Port {
			zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v minio port to %v", patp, remote.Port))
			local.WgS3Port = remote.Port
			*persistNetwork = true
			*modified = true
		}
		return true
	}

	consd := ""
	if len(urlParts) >= 3 {
		consd = strings.Join(urlParts[:3], ".")
	}
	if consd == "console.s3."+patp && remote.SvcType == string(transition.StartramServiceTypeMinioAdmin) && remote.Status == string(transition.StartramServiceStatusOk) {
		zap.L().Debug(fmt.Sprintf("Retrieve: Setting %v console port to %v", patp, remote.Port))
		if remote.Port != local.WgConsolePort {
			local.WgConsolePort = remote.Port
			*persistNetwork = true
			*modified = true
		}
		return true
	}
	return false
}
