package rectify

import (
	"errors"
	"fmt"
	"strings"

	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
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
		settings := service.runtime.StartramSettingsSnapshotFn()
		current.Profile.Startram.Info.Endpoint = settings.EndpointURL
	case transition.StartramTransitionRegister:
		if eventData != transition.StartramTransitionComplete {
			return nil
		}
		settings := service.runtime.StartramSettingsSnapshotFn()
		current.Profile.Startram.Info.Running = settings.WgOn
		containerState, exists := service.runtime.GetContainerStateFn()[string(transition.ContainerTypeWireguard)]
		if exists {
			running := containerState.ActualStatus == string(transition.ContainerStatusRunning)
			current.Profile.Startram.Info.Running = running
			if err := service.runtime.UpdateConfig(config.WithWgOn(running)); err != nil {
				return fmt.Errorf("persisting wireguard running state: %w", err)
			}
		}
		current.Profile.Startram.Info.Registered = settings.WgRegistered
	}
	return nil
}

type StartramRetrieveReconciler struct {
	runtime         RectifyRuntime
	syncer          *urbitConfigSyncService
	createServiceFn func(subdomain, svcType string) error
}

func NewStartramRetrieveReconciler(runtime RectifyRuntime) *StartramRetrieveReconciler {
	return &StartramRetrieveReconciler{
		runtime:         runtime,
		syncer:          &urbitConfigSyncService{runtime: runtime},
		createServiceFn: startram.SvcCreate,
	}
}

func (reconciler *StartramRetrieveReconciler) Reconcile(current *structs.AuthBroadcast) error {
	runtime := reconciler.runtime
	startramSettings := runtime.StartramSettingsSnapshotFn()
	startramConfig := runtime.GetStartramConfigFn()
	var reconcileErr error
	for patp := range runtime.UrbitConfAllFn() {
		published := false
		publishRegistrationTransition := func(serviceCreated bool) {
			if published {
				return
			}
			publishUrbitServiceRegistrationTransitionWithCurrentState(current, patp, serviceCreated)
			published = true
		}

		local, err := reconciler.syncer.loadAndRefresh(patp)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Retrieve: unable to refresh urbit config for %s: %v", patp, err))
			reconcileErr = errors.Join(reconcileErr, fmt.Errorf("unable to refresh urbit config for %s: %w", patp, err))
			publishRegistrationTransition(false)
			continue
		}

		plan, err := reconciler.reconcilePatpState(patp, local, startramConfig.Subdomains, startramSettings)
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Retrieve: unable to reconcile %s startram state: %v", patp, err))
			reconcileErr = errors.Join(reconcileErr, err)
			publishRegistrationTransition(false)
			continue
		}

		if err := reconciler.persistPatpState(patp, plan); err != nil {
			reconcileErr = errors.Join(reconcileErr, err)
		}
		publishRegistrationTransition(plan.serviceCreated)
	}
	return reconcileErr
}

type startramPatpReconcilePlan struct {
	local          structs.UrbitDocker
	modified       bool
	serviceCreated bool
	persistWeb     bool
	persistNetwork bool
}

func (reconciler *StartramRetrieveReconciler) reconcilePatpState(patp string, local structs.UrbitDocker, subdomains []structs.Subdomain, startramSettings config.StartramSettings) (startramPatpReconcilePlan, error) {
	plan := startramPatpReconcilePlan{local: local, serviceCreated: true}

	endpointRoot, ok := startramEndpointRoot(startramSettings.EndpointURL)
	if !ok {
		return plan, fmt.Errorf("invalid startram endpoint URL %q for %s", startramSettings.EndpointURL, patp)
	}

	if !isStartramPatpRegistered(patp, endpointRoot, subdomains) {
		zap.L().Info(fmt.Sprintf("Registering missing StarTram service for %v", patp))
		createService := reconciler.createServiceFn
		if createService == nil {
			createService = startram.SvcCreate
		}
		var createErr error
		if err := createService(patp, "urbit"); err != nil {
			createErr = errors.Join(createErr, fmt.Errorf("create urbit service %s: %w", patp, err))
		}
		s3Subdomain := "s3." + patp
		if err := createService(s3Subdomain, "minio"); err != nil {
			createErr = errors.Join(createErr, fmt.Errorf("create minio service %s: %w", s3Subdomain, err))
		}
		if createErr != nil {
			plan.serviceCreated = false
			return plan, createErr
		}
	}

	for _, remote := range subdomains {
		if remote.Status == string(transition.StartramServiceStatusCreating) {
			plan.serviceCreated = false
		}
		parts := strings.Split(remote.URL, ".")
		if len(parts) < 2 {
			continue
		}
		if reconciler.reconcileUrbitWebService(patp, remote, &plan.local, &plan.modified, &plan.persistWeb, &plan.persistNetwork) {
			continue
		}
		if reconciler.reconcileUrbitNetworkServices(patp, remote, parts, &plan.local, &plan.modified, &plan.persistNetwork) {
			continue
		}
	}

	return plan, nil
}

func (reconciler *StartramRetrieveReconciler) persistPatpState(patp string, plan startramPatpReconcilePlan) error {
	if !plan.modified {
		return nil
	}
	var reconcileErr error

	if plan.persistWeb {
		if err := reconciler.syncer.updateWebConfig(patp, &plan.local); err != nil {
			reconcileErr = errors.Join(reconcileErr, fmt.Errorf("unable to persist %s web config updates: %w", patp, err))
		}
	}

	if plan.persistNetwork {
		if err := reconciler.syncer.updateNetworkConfig(patp, &plan.local); err != nil {
			reconcileErr = errors.Join(reconcileErr, fmt.Errorf("unable to persist %s network config updates: %w", patp, err))
		}
	}

	if reconcileErr != nil {
		zap.L().Warn(fmt.Sprintf("Retrieve: unable to persist reconciled state for %s: %v", patp, reconcileErr))
	}
	return reconcileErr
}

func isStartramPatpRegistered(patp, endpointRoot string, subdomains []structs.Subdomain) bool {
	expected := patp + "." + endpointRoot
	for _, remote := range subdomains {
		if remote.URL == expected {
			return true
		}
	}
	return false
}

func startramEndpointRoot(endpointURL string) (string, bool) {
	endpointParts := strings.Split(endpointURL, ".")
	if len(endpointParts) < 2 {
		return "", false
	}
	return strings.Join(endpointParts[1:], "."), true
}

type urbitConfigSyncService struct {
	runtime RectifyRuntime
}

func (service *urbitConfigSyncService) loadAndRefresh(patp string) (structs.UrbitDocker, error) {
	if err := service.runtime.LoadUrbitConfigFn(patp); err != nil {
		return structs.UrbitDocker{}, fmt.Errorf("loading urbit config for %s: %w", patp, err)
	}
	local := service.runtime.UrbitConfFn(patp)
	return local, nil
}

func (service *urbitConfigSyncService) updateWebConfig(patp string, local *structs.UrbitDocker) error {
	if err := service.runtime.UpdateUrbitSectionFn(
		patp,
		config.UrbitConfigSectionWeb,
		config.AdaptUrbitSectionMutation(func(webConfig *structs.UrbitWebConfig) error {
			webConfig.CustomUrbitWeb = local.CustomUrbitWeb
			return nil
		}),
	); err != nil {
		return fmt.Errorf("persisting web config for %s: %w", patp, err)
	}
	return nil
}

func (service *urbitConfigSyncService) updateNetworkConfig(patp string, local *structs.UrbitDocker) error {
	if err := service.runtime.UpdateUrbitSectionFn(
		patp,
		config.UrbitConfigSectionNetwork,
		config.AdaptUrbitSectionMutation(func(networkConfig *structs.UrbitNetworkConfig) error {
			networkConfig.WgHTTPPort = local.WgHTTPPort
			networkConfig.WgAmesPort = local.WgAmesPort
			networkConfig.WgS3Port = local.WgS3Port
			networkConfig.WgConsolePort = local.WgConsolePort
			networkConfig.WgURL = local.WgURL
			networkConfig.Network = local.Network
			return nil
		}),
	); err != nil {
		return fmt.Errorf("persisting network config for %s: %w", patp, err)
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
