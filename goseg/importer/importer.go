package importer

import (
	"context"
	"errors"
	"fmt"
	"github.com/docker/docker/errdefs"
	"groundseg/auth"
	"groundseg/auth/tokens"
	"groundseg/click/acme"
	"groundseg/config"
	"groundseg/docker/events"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/driveresolver"
	"groundseg/internal/resource"
	"groundseg/internal/seams"
	"groundseg/lifecycle"
	"groundseg/logger"
	workflowOrchestration "groundseg/orchestration"
	"groundseg/shipcleanup"
	"groundseg/shipcreator"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"groundseg/system"
	"groundseg/transition"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type uploadSession struct {
	Token           structs.WsTokenStruct
	Remote          bool
	Fix             bool
	SelectedDrive   string
	CustomDrive     string
	NeedsFormatting bool
	ExpiresAt       time.Time
}

type OpenUploadEndpointCmd struct {
	Endpoint      string
	Token         structs.WsTokenStruct
	Remote        bool
	Fix           bool
	SelectedDrive string
}

var (
	uploadSessions = make(map[string]uploadSession) // todo: add checkbox data to struct
	uploadDir      string
	tempDir        string
	uploadMu       sync.RWMutex
	uploadTTL      = 20 * time.Minute
	uploadKeyRegex = regexp.MustCompile(`^[a-f0-9]{32}$`)

	resolveStoragePathFor = config.GetStoragePath
)

var errImporterUpdateContainerStateMissing = seams.MissingRuntimeDependency("importer runtime container state callback", "")
var errImporterSwapSettingsMissing = seams.MissingRuntimeDependency("importer runtime swap settings callback", "")
var errImporterStartramSettingsMissing = seams.MissingRuntimeDependency("importer runtime startram settings callback", "")
var errImporterTokenValidationCallbackMissing = seams.MissingRuntimeDependency("importer runtime token validation callback", "")
var errImporterCreateDirMissing = seams.MissingRuntimeDependency("importer runtime directory creation callback", "")
var errImporterCloseTempFileMissing = seams.MissingRuntimeDependency("importer runtime close temp file callback", "")
var errImporterFileStatusMissing = seams.MissingRuntimeDependency("importer runtime file status callback", "")
var errImporterBootCodeWaitMissing = seams.MissingRuntimeDependency("importer runtime boot code wait callback", "")
var errImporterRemoteReadyWaitMissing = seams.MissingRuntimeDependency("importer runtime remote ready callback", "")
var errImporterWireguardSwitchMissing = seams.MissingRuntimeDependency("importer runtime wireguard switch callback", "")
var errImporterAcmeFixMissing = seams.MissingRuntimeDependency("importer runtime ACME fix callback", "")
var errImporterUrbitConfigMissing = seams.MissingRuntimeDependency("importer runtime create urbit config callback", "")
var errImporterUrbitSysConfigMissing = seams.MissingRuntimeDependency("importer runtime append urbit sysconfig callback", "")
var errImporterVolumeDeleteMissing = seams.MissingRuntimeDependency("importer runtime volume delete callback", "")
var errImporterContainerDeleteMissing = seams.MissingRuntimeDependency("importer runtime container delete callback", "")
var errImporterVolumeCreateMissing = seams.MissingRuntimeDependency("importer runtime volume create callback", "")
var errImporterMkdirMissing = seams.MissingRuntimeDependency("importer runtime mkdir callback", "")
var errImporterUpdateContainerMissing = seams.MissingRuntimeDependency("importer runtime container update callback", "")
var errImporterStartContainerMissing = seams.MissingRuntimeDependency("importer runtime container start callback", "")
var errImporterStoragePathMissing = seams.MissingRuntimeDependency("importer runtime storage path callback", "")
var errImporterVolumeResolutionMissing = seams.MissingRuntimeDependency("importer runtime volume resolution callback", "")
var errImporterCleanupMissing = seams.MissingRuntimeDependency("importer runtime cleanup callback", "")
var errImporterWorkflowMissing = seams.MissingRuntimeDependency("importer runtime workflow callback", "")
var errImporterPostImportWorkflowMissing = seams.MissingRuntimeDependency("importer runtime post-import workflow callback", "")

type importerStorageRuntime struct {
	StoragePathForFn   func(string) (string, error)    `runtime:"importer-core" runtime_name:"storage path"`
	CreateDirFn        func(string, os.FileMode) error `runtime:"importer-core" runtime_name:"create dir"`
	SwapSettingsFn     func() config.SwapSettings      `runtime:"importer-config" runtime_name:"swap settings"`
	StartramSettingsFn func() config.StartramSettings  `runtime:"importer-config" runtime_name:"startram settings"`
}

type importerUploadRuntime struct {
	ValidateUploadSessionTokenFn func(structs.WsTokenStruct, structs.WsTokenStruct, *http.Request) auth.UploadTokenAuthorizationResult `runtime:"importer-upload" runtime_name:"validate upload token"`
	StatFn                       func(string) (os.FileInfo, error)                                                                     `runtime:"importer-upload" runtime_name:"stat"`
	CloseTempFileFn              func(*os.File) error                                                                                  `runtime:"importer-upload" runtime_name:"close temp file"`
}

type importerProvisionRuntime struct {
	ResolveImportedPierVolumePathFn     func(context.Context, string) (string, error)    `runtime:"importer-provision" runtime_name:"volume resolution"`
	RunImportedPierWorkflowFn           func(importedPierContext, importerRuntime) error `runtime:"importer-provision" runtime_name:"import workflow"`
	RunImportedPierPostImportWorkflowFn func(importedPierContext, importerRuntime) error `runtime:"importer-provision" runtime_name:"post-import workflow"`
	CleanupMultipartFn                  func(string) error                               `runtime:"importer-provision" runtime_name:"cleanup"`
	UploadCoordinatorFn                 shipworkflow.UploadImportCoordinator             `runtime:"importer-provision" runtime_name:"upload import coordinator"`
	ShipworkflowWaitForBootCodeFn       func(string, time.Duration)                      `runtime:"importer-provision" runtime_name:"boot code wait"`
	ShipworkflowWaitForRemoteReadyFn    func(string, time.Duration)                      `runtime:"importer-provision" runtime_name:"remote ready"`
	ShipworkflowSwitchToWireguardFn     func(string, bool) error                         `runtime:"importer-provision" runtime_name:"wireguard switch"`
	AcmeFixFn                           func(string) error                               `runtime:"importer-provision" runtime_name:"acme fix"`
	ShipcreatorCreateUrbitConfigFn      func(string, string) error                       `runtime:"importer-provision" runtime_name:"create urbit config"`
	ShipcreatorAppendSysConfigPierFn    func(string) error                               `runtime:"importer-provision" runtime_name:"append urbit sysconfig"`
	dockerOrchestration.RuntimeContainerOps
	dockerOrchestration.RuntimeVolumeOps
	DeleteVolumeFn            func(string) error                                    `runtime:"importer-provision" runtime_name:"delete volume"`
	MkdirAllFn                func(string, os.FileMode) error                       `runtime:"importer-provision" runtime_name:"mkdir"`
	PublishImportTransitionFn func(context.Context, structs.UploadTransition) error `runtime:"importer-provision" runtime_name:"publish import transition callback"`
}

type importerRuntime struct {
	importerStorageRuntime
	importerUploadRuntime
	importerProvisionRuntime
}

func defaultImporterRuntime() importerRuntime {
	orchestrationRuntime := dockerOrchestration.NewRuntime()
	return importerRuntime{
		importerStorageRuntime: importerStorageRuntime{
			StoragePathForFn:   resolveStoragePathFor,
			SwapSettingsFn:     config.SwapSettingsSnapshot,
			StartramSettingsFn: config.StartramSettingsSnapshot,
			CreateDirFn:        os.MkdirAll,
		},
		importerUploadRuntime: importerUploadRuntime{
			ValidateUploadSessionTokenFn: auth.ValidateUploadSessionTokenRequest,
			StatFn:                       os.Stat,
			CloseTempFileFn: func(file *os.File) error {
				return file.Close()
			},
		},
		importerProvisionRuntime: importerProvisionRuntime{
			ResolveImportedPierVolumePathFn: func(_ context.Context, patp string) (string, error) {
				return resolveImportedPierVolumePath(patp)
			},
			RunImportedPierWorkflowFn:           runImportedPierWorkflowDefault,
			RunImportedPierPostImportWorkflowFn: runImportedPierPostImportWorkflowDefault,
			CleanupMultipartFn:                  system.RemoveMultipartFiles,
			UploadCoordinatorFn: func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
				return shipworkflow.DispatchUploadImportWithCoordinator(func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
					return configureUploadedPier(ctx, cmd)
				}, ctx, cmd)
			},
			ShipworkflowWaitForBootCodeFn:    shipworkflow.WaitForBootCode,
			ShipworkflowWaitForRemoteReadyFn: shipworkflow.WaitForRemoteReady,
			ShipworkflowSwitchToWireguardFn:  shipworkflow.SwitchShipToWireguard,
			AcmeFixFn:                        acme.Fix,
			ShipcreatorCreateUrbitConfigFn:   shipcreator.CreateUrbitConfig,
			ShipcreatorAppendSysConfigPierFn: shipcreator.AppendSysConfigPier,
			RuntimeContainerOps: dockerOrchestration.RuntimeContainerOps{
				DeleteContainerFn:      orchestrationRuntime.DeleteContainerFn,
				StartContainerFn:       orchestrationRuntime.StartContainerFn,
				UpdateContainerStateFn: orchestrationRuntime.UpdateContainerStateFn,
			},
			RuntimeVolumeOps: dockerOrchestration.RuntimeVolumeOps{
				CreateVolumeFn: dockerOrchestration.CreateVolume,
			},
			PublishImportTransitionFn: events.DefaultEventRuntime().PublishImportShipTransition,
			DeleteVolumeFn:            dockerOrchestration.DeleteVolume,
			MkdirAllFn:                os.MkdirAll,
		},
	}
}

func validateImporterConfigRuntime(ops importerRuntime, requireSwapSettings bool) error {
	groups := []string{"importer-core", "importer-upload"}
	if requireSwapSettings {
		groups = append(groups, "importer-config")
	}
	if err := seams.NewCallbackRequirementsWithGroups(groups...).ValidateCallbacks(ops, "importer runtime"); err != nil {
		return seams.MissingRuntimeDependency("importer runtime", err.Error())
	}
	return nil
}

func validateImporterUploadRuntime(ops importerRuntime) error {
	if err := seams.NewCallbackRequirementsWithGroups("importer-upload").ValidateCallbacks(ops, "importer runtime"); err != nil {
		return seams.MissingRuntimeDependency("importer runtime", err.Error())
	}
	return nil
}

func validateImporterProvisionRuntime(ops importerRuntime) error {
	if err := seams.NewCallbackRequirementsWithGroups("importer-provision").ValidateCallbacks(ops, "importer runtime"); err != nil {
		return seams.MissingRuntimeDependency("importer runtime", err.Error())
	}
	return nil
}

func resolveImportedPierVolumePath(patp string) (string, error) {
	if patp == "" {
		return "", fmt.Errorf("ship name is required for volume resolution")
	}
	return filepath.Join(dockerOrchestration.VolumeDir, patp, "_data"), nil
}

var (
	errUploadImportCoordinatorUnconfigured = seams.MissingRuntimeDependency("importer runtime upload import coordinator", "")
	errChunkCombineTimeout                 = errors.New("import chunk combining timed out")
	errImportPierConfig                    = errors.New("failed to configure imported pier")
)

type importerRuntimeRequirements struct {
	requireSwapSettings bool
	requireProvision    bool
}

var (
	importerInitRequirements     = importerRuntimeRequirements{requireSwapSettings: true}
	importerUploadRequirements   = importerRuntimeRequirements{}
	importerWorkflowRequirements = importerRuntimeRequirements{requireSwapSettings: true, requireProvision: true}
)

func Initialize(runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForInit(runtime...)
	if err != nil {
		return fmt.Errorf("initialize importer runtime: %w", err)
	}
	uploadDir, err = resolvedRuntime.StoragePathForFn("uploads")
	if err != nil {
		return fmt.Errorf("initialize upload directory: %w", err)
	}
	tempDir, err = resolvedRuntime.StoragePathForFn("temp")
	if err != nil {
		return fmt.Errorf("initialize temp directory: %w", err)
	}
	swapSettings := resolvedRuntime.SwapSettingsFn()
	if !strings.HasPrefix(swapSettings.SwapFile, "/opt") {
		var tempPath string
		lastSlashIndex := strings.LastIndex(swapSettings.SwapFile, "/")
		if lastSlashIndex != -1 {
			tempPath = swapSettings.SwapFile[:lastSlashIndex]
			tempDir = filepath.Join(tempPath, "temp")
			uploadDir = filepath.Join(tempPath, "uploads")
		}
	}
	if err := resolvedRuntime.CreateDirFn(uploadDir, 0755); err != nil {
		return fmt.Errorf("create upload directory %q: %w", uploadDir, err)
	}
	if err := resolvedRuntime.CreateDirFn(tempDir, 0755); err != nil {
		return fmt.Errorf("create temp directory %q: %w", tempDir, err)
	}
	return nil
}

func resolveImporterRuntime(overrides ...importerRuntime) importerRuntime {
	if len(overrides) > 0 {
		return importerRuntimeWithDefaults(overrides[0])
	}
	return defaultImporterRuntime()
}

func importerRuntimeWithDefaults(overrides importerRuntime) importerRuntime {
	runtime := defaultImporterRuntime()
	if overrides.StoragePathForFn != nil {
		runtime.StoragePathForFn = overrides.StoragePathForFn
	}
	if overrides.CreateDirFn != nil {
		runtime.CreateDirFn = overrides.CreateDirFn
	}
	if overrides.SwapSettingsFn != nil {
		runtime.SwapSettingsFn = overrides.SwapSettingsFn
	}
	if overrides.StartramSettingsFn != nil {
		runtime.StartramSettingsFn = overrides.StartramSettingsFn
	}
	if overrides.ValidateUploadSessionTokenFn != nil {
		runtime.ValidateUploadSessionTokenFn = overrides.ValidateUploadSessionTokenFn
	}
	if overrides.StatFn != nil {
		runtime.StatFn = overrides.StatFn
	}
	if overrides.CloseTempFileFn != nil {
		runtime.CloseTempFileFn = overrides.CloseTempFileFn
	}
	if overrides.ResolveImportedPierVolumePathFn != nil {
		runtime.ResolveImportedPierVolumePathFn = overrides.ResolveImportedPierVolumePathFn
	}
	if overrides.RunImportedPierWorkflowFn != nil {
		runtime.RunImportedPierWorkflowFn = overrides.RunImportedPierWorkflowFn
	}
	if overrides.RunImportedPierPostImportWorkflowFn != nil {
		runtime.RunImportedPierPostImportWorkflowFn = overrides.RunImportedPierPostImportWorkflowFn
	}
	if overrides.CleanupMultipartFn != nil {
		runtime.CleanupMultipartFn = overrides.CleanupMultipartFn
	}
	if overrides.UploadCoordinatorFn != nil {
		runtime.UploadCoordinatorFn = overrides.UploadCoordinatorFn
	}
	if overrides.ShipworkflowWaitForBootCodeFn != nil {
		runtime.ShipworkflowWaitForBootCodeFn = overrides.ShipworkflowWaitForBootCodeFn
	}
	if overrides.ShipworkflowWaitForRemoteReadyFn != nil {
		runtime.ShipworkflowWaitForRemoteReadyFn = overrides.ShipworkflowWaitForRemoteReadyFn
	}
	if overrides.ShipworkflowSwitchToWireguardFn != nil {
		runtime.ShipworkflowSwitchToWireguardFn = overrides.ShipworkflowSwitchToWireguardFn
	}
	if overrides.AcmeFixFn != nil {
		runtime.AcmeFixFn = overrides.AcmeFixFn
	}
	if overrides.ShipcreatorCreateUrbitConfigFn != nil {
		runtime.ShipcreatorCreateUrbitConfigFn = overrides.ShipcreatorCreateUrbitConfigFn
	}
	if overrides.ShipcreatorAppendSysConfigPierFn != nil {
		runtime.ShipcreatorAppendSysConfigPierFn = overrides.ShipcreatorAppendSysConfigPierFn
	}
	if overrides.DeleteContainerFn != nil {
		runtime.RuntimeContainerOps.DeleteContainerFn = overrides.DeleteContainerFn
	}
	if overrides.RuntimeContainerOps.StartContainerFn != nil {
		runtime.RuntimeContainerOps.StartContainerFn = overrides.RuntimeContainerOps.StartContainerFn
	}
	if overrides.RuntimeContainerOps.UpdateContainerStateFn != nil {
		runtime.RuntimeContainerOps.UpdateContainerStateFn = overrides.RuntimeContainerOps.UpdateContainerStateFn
	}
	if overrides.DeleteVolumeFn != nil {
		runtime.DeleteVolumeFn = overrides.DeleteVolumeFn
	}
	if overrides.RuntimeVolumeOps.CreateVolumeFn != nil {
		runtime.RuntimeVolumeOps.CreateVolumeFn = overrides.RuntimeVolumeOps.CreateVolumeFn
	}
	if overrides.MkdirAllFn != nil {
		runtime.MkdirAllFn = overrides.MkdirAllFn
	}
	if overrides.PublishImportTransitionFn != nil {
		runtime.PublishImportTransitionFn = overrides.PublishImportTransitionFn
	}
	return runtime
}

func resolveImporterRuntimeForRequirements(requirements importerRuntimeRequirements, overrides ...importerRuntime) (importerRuntime, error) {
	resolvedRuntime := resolveImporterRuntime(overrides...)
	if err := validateImporterConfigRuntime(resolvedRuntime, requirements.requireSwapSettings); err != nil {
		return resolvedRuntime, err
	}
	if err := validateImporterUploadRuntime(resolvedRuntime); err != nil {
		return resolvedRuntime, err
	}
	if requirements.requireProvision {
		if err := validateImporterProvisionRuntime(resolvedRuntime); err != nil {
			return resolvedRuntime, err
		}
	}
	return resolvedRuntime, nil
}

func resolveImporterRuntimeForWorkflow(overrides ...importerRuntime) (importerRuntime, error) {
	return resolveImporterRuntimeForRequirements(importerWorkflowRequirements, overrides...)
}

func resolveImporterRuntimeForUpload(overrides ...importerRuntime) (importerRuntime, error) {
	return resolveImporterRuntimeForRequirements(importerUploadRequirements, overrides...)
}

func resolveImporterRuntimeForInit(overrides ...importerRuntime) (importerRuntime, error) {
	return resolveImporterRuntimeForRequirements(importerInitRequirements, overrides...)
}

func OpenUploadEndpoint(cmd OpenUploadEndpointCmd, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForUpload(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer upload runtime: %w", err)
	}
	uploadMu.Lock()
	defer uploadMu.Unlock()
	// grab from payload
	endpoint := cmd.Endpoint
	token := cmd.Token
	remote := cmd.Remote
	fix := cmd.Fix
	if !uploadKeyRegex.MatchString(endpoint) {
		return errors.New("invalid upload session key format")
	}
	if token.ID == "" || token.Token == "" {
		return errors.New("missing upload auth token")
	}
	var expectedToken structs.WsTokenStruct
	if existingSession, exists := uploadSessions[endpoint]; exists {
		expectedToken = existingSession.Token
	}

	authz := resolvedRuntime.ValidateUploadSessionTokenFn(expectedToken, token, nil)
	if errors.Is(authz.AuthorizationErr, errImporterTokenValidationCallbackMissing) {
		return fmt.Errorf("upload token validation seam missing: %w", authz.AuthorizationErr)
	}
	if !authz.IsAuthorized() {
		if authz.Status == tokens.UploadValidationStatusTokenContract {
			return errors.New("token mismatch")
		}
		return errors.New("upload token id is not authorized")
	}

	driveResolution, err := driveresolver.Resolve(cmd.SelectedDrive)
	if err != nil {
		return fmt.Errorf("resolve selected drive: %w", err)
	}
	sel := driveResolution.SelectedDrive
	customDrive := driveResolution.Mountpoint

	// check if endpoint exists
	existingSession, exists := uploadSessions[endpoint]
	if !exists {
		//build new configuration
		sesh := uploadSession{
			Token:           token,
			Remote:          remote,
			Fix:             fix,
			SelectedDrive:   sel,
			CustomDrive:     customDrive,
			NeedsFormatting: driveResolution.NeedsFormatting,
			ExpiresAt:       time.Now().Add(uploadTTL),
		}
		uploadSessions[endpoint] = sesh
		return nil
	}
	// Modify checkboxes
	existingSession.Remote = remote
	existingSession.Fix = fix
	existingSession.SelectedDrive = sel
	existingSession.CustomDrive = customDrive
	existingSession.NeedsFormatting = driveResolution.NeedsFormatting
	existingSession.ExpiresAt = time.Now().Add(uploadTTL)

	uploadSessions[endpoint] = existingSession
	logger.Warnf("current upload configuration: %+v", cmd)
	return nil
}

func SetUploadSession(uploadPayload structs.WsUploadPayload, runtime ...importerRuntime) error {
	return OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint:      uploadPayload.Payload.Endpoint,
		Token:         uploadPayload.Token,
		Remote:        uploadPayload.Payload.Remote,
		Fix:           uploadPayload.Payload.Fix,
		SelectedDrive: uploadPayload.Payload.SelectedDrive,
	}, runtime...)
}

func ClearUploadSession(session string) {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	delete(uploadSessions, session)
}

func VerifySession(session string) bool {
	_, exists := loadUploadSession(session)
	return exists
}

func loadUploadSession(session string) (uploadSession, bool) {
	uploadMu.RLock()
	defer uploadMu.RUnlock()
	sesh, exists := uploadSessions[session]
	return sesh, exists
}

func storeUploadSession(session string, sesh uploadSession) {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	uploadSessions[session] = sesh
}

func Reset(runtime ...importerRuntime) error {
	var resetErr error
	if err := publishImportStatus("", runtime...); err != nil {
		resetErr = errors.Join(resetErr, err)
	}
	if err := publishImportTransition(structs.UploadTransition{Type: "patp", Event: ""}, runtime...); err != nil {
		resetErr = errors.Join(resetErr, err)
	}
	if err := publishImportError("", runtime...); err != nil {
		resetErr = errors.Join(resetErr, err)
	}
	if err := publishImportTransition(structs.UploadTransition{Type: "extracted", Value: 0}, runtime...); err != nil {
		resetErr = errors.Join(resetErr, err)
	}
	if resetErr != nil {
		return fmt.Errorf("publish baseline import transitions: %w", resetErr)
	}
	return nil
}

type validatedUploadRequest struct {
	SessionID string
	Patp      string
	Session   uploadSession
	Context   context.Context
}

type uploadChunkPayload struct {
	Filename string
	Index    int
	Total    int
	File     io.ReadCloser
}

type uploadChunkProgress struct {
	Filename  string
	Total     int
	AllChunks bool
}

func setUploadResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, X-Requested-With")
}

func sendUploadResponse(w http.ResponseWriter, code int, status, message string) error {
	w.WriteHeader(code)
	if _, err := w.Write([]byte(fmt.Sprintf(`{"status": "%s", "message": "%s"}`, status, message))); err != nil {
		return fmt.Errorf("failed to write upload response: %w", err)
	}
	return nil
}

func publishImportStatus(event string, runtime ...importerRuntime) error {
	return publishImportTransitionWithPolicy(
		structs.UploadTransition{
			Type:  "status",
			Event: event,
		},
		transition.TransitionPolicyForCriticality(transition.TransitionPublishCritical),
		runtime...,
	)
}

func publishImportStatusBestEffort(event string, runtime ...importerRuntime) error {
	return publishImportTransitionWithPolicy(
		structs.UploadTransition{
			Type:  "status",
			Event: event,
		},
		transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
		runtime...,
	)
}

func publishImportError(message string, runtime ...importerRuntime) error {
	return publishImportTransitionWithPolicy(
		structs.UploadTransition{Type: "error", Event: message},
		transition.TransitionPolicyForCriticality(transition.TransitionPublishCritical),
		runtime...,
	)
}

func publishImportTransition(uploadTransition structs.UploadTransition, runtime ...importerRuntime) error {
	return publishImportTransitionWithPolicy(
		uploadTransition,
		transition.TransitionPolicyForCriticality(transition.TransitionPublishCritical),
		runtime...,
	)
}

func publishImportTransitionBestEffort(uploadTransition structs.UploadTransition, runtime ...importerRuntime) error {
	return publishImportTransitionWithPolicy(
		uploadTransition,
		transition.TransitionPolicyForCriticality(transition.TransitionPublishNonCritical),
		runtime...,
	)
}

func publishImportTransitionWithPolicy(uploadTransition structs.UploadTransition, publishPolicy transition.TransitionPublishPolicy, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
	err := resolvedRuntime.PublishImportTransitionFn(context.Background(), uploadTransition)
	if err == nil {
		return nil
	}
	return transition.HandleTransitionPublishError(
		fmt.Sprintf("publish upload transition (%s)", uploadTransition.Type),
		err,
		publishPolicy,
	)
}

func failUploadRequest(w http.ResponseWriter, code int, message string, runtime ...importerRuntime) error {
	logger.Errorf("upload error: %v", message)
	if err := publishImportError(message, runtime...); err != nil {
		logger.Warnf("failed to publish import error transition for %q: %v", message, err)
	}
	if err := publishImportStatus("aborted", runtime...); err != nil {
		logger.Warnf("failed to publish import status transition for aborted upload of %q: %v", message, err)
	}
	return sendUploadResponse(w, code, "failure", message)
}

func validateUploadRequest(r *http.Request, uploadSession, patp string, runtime ...importerRuntime) (validatedUploadRequest, error) {
	runtimeConfig, err := resolveImporterRuntimeForUpload(runtime...)
	if err != nil {
		return validatedUploadRequest{}, fmt.Errorf("prepare importer upload runtime: %w", err)
	}
	var validated validatedUploadRequest
	session := uploadSession
	shipName := patp

	sesh, validSession := loadUploadSession(session)

	if !validSession {
		return validated, fmt.Errorf("invalid upload session")
	}
	if time.Now().After(sesh.ExpiresAt) {
		ClearUploadSession(session)
		return validated, fmt.Errorf("upload session expired")
	}

	tokenID := strings.TrimSpace(r.Header.Get("X-Upload-Token-Id"))
	tokenHash := strings.TrimSpace(r.Header.Get("X-Upload-Token"))
	if tokenID == "" || tokenHash == "" {
		return validated, fmt.Errorf("missing upload authorization headers")
	}
	authz := runtimeConfig.ValidateUploadSessionTokenFn(sesh.Token, structs.WsTokenStruct{
		ID:    tokenID,
		Token: tokenHash,
	}, r)
	if errors.Is(authz.AuthorizationErr, errImporterTokenValidationCallbackMissing) {
		return validated, fmt.Errorf("upload token validation seam missing: %w", authz.AuthorizationErr)
	}

	if !authz.IsAuthorized() {
		if authz.Status == tokens.UploadValidationStatusTokenContract {
			return validated, fmt.Errorf("upload token does not match upload session")
		}
		return validated, fmt.Errorf("upload token validation failed: %w", authz.AuthorizationErr)
	}

	if authz.IsRotated() {
		// keep session token in sync with any rotated, authorized token
		sesh.Token.Token = authz.AuthorizedToken
		storeUploadSession(session, sesh)
	}

	validated.SessionID = session
	validated.Patp = shipName
	validated.Session = sesh
	validated.Context = r.Context()
	return validated, nil
}

func ensureUploadDriveReady(sessionID string, sesh uploadSession) (uploadSession, error) {
	if sesh.SelectedDrive == "system-drive" || !sesh.NeedsFormatting {
		return sesh, nil
	}
	resolution, err := driveresolver.EnsureReady(driveresolver.Resolution{
		SelectedDrive:   sesh.SelectedDrive,
		Mountpoint:      sesh.CustomDrive,
		NeedsFormatting: sesh.NeedsFormatting,
	})
	if err != nil {
		return sesh, fmt.Errorf("ensure selected drive ready for upload session %s: %w", sessionID, err)
	}

	uploadMu.Lock()
	defer uploadMu.Unlock()
	sesh.NeedsFormatting = resolution.NeedsFormatting
	sesh.CustomDrive = resolution.Mountpoint
	uploadSessions[sessionID] = sesh
	return sesh, nil
}

func parseUploadChunkMetadata(r *http.Request, filename string) (int, int, error) {
	chunkIndex := r.FormValue("dzchunkindex")
	totalChunks := r.FormValue("dztotalchunkcount")
	logger.Debug(fmt.Sprintf("%v chunkIndex: %v, totalChunks: %v", filename, chunkIndex, totalChunks))

	index, err := strconv.Atoi(chunkIndex)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid chunk index")
	}
	total, err := strconv.Atoi(totalChunks)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid total chunk count")
	}
	return index, total, nil
}

func persistChunkToTemp(file io.Reader, filename string, index int, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForUpload(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer upload runtime: %w", err)
	}
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, index))
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	buffer := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(tempFile, file, buffer); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to save chunk: %w", err)
	}
	if err := resolvedRuntime.CloseTempFileFn(tempFile); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	return nil
}

func combineChunksWithTimeout(filename string, total int, timeout time.Duration) error {
	combineCtx, combineCancel := context.WithTimeout(context.Background(), timeout)
	defer combineCancel()
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- combineChunks(filename, total)
	}()
	select {
	case err := <-resultCh:
		if err != nil {
			return fmt.Errorf("combine chunks for %s: %w", filename, err)
		}
		return nil
	case <-combineCtx.Done():
		if errors.Is(combineCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("combine chunks for %s: %w", filename, errChunkCombineTimeout)
		}
		return fmt.Errorf("combine chunks for %s timed out: %w", filename, combineCtx.Err())
	}
}

func runImportPhases(filename, patp, customDrive string, phases workflowOrchestration.WorkflowPhases, runtime ...importerRuntime) error {
	var publishErr error
	workflowErr := workflowOrchestration.RunStructuredWorkflow(
		phases,
		workflowOrchestration.WorkflowCallbacks{
			Emit: func(phase lifecycle.Phase) {
				if phase == "" {
					return
				}
				if err := publishImportStatusBestEffort(string(phase), runtime...); err != nil {
					logger.Warnf("failed to publish import phase status %q: %v", phase, err)
					publishErr = errors.Join(publishErr, err)
				}
			},
			OnError: func(err error) {
				if transitionErr := publishImportError(err.Error(), runtime...); transitionErr != nil {
					logger.Warnf("failed to publish import phase error %q: %v", err, transitionErr)
					publishErr = errors.Join(publishErr, transitionErr)
				}
			},
		},
	)
	if workflowErr == nil && publishErr != nil {
		return publishErr
	}
	if workflowErr == nil {
		return nil
	}
	if cleanupErr := errorCleanup(filename, patp, customDrive, workflowErr, runtime...); cleanupErr != nil {
		return fmt.Errorf("import failed for %s: %w", patp, errors.Join(workflowErr, cleanupErr))
	}
	if publishErr != nil {
		return fmt.Errorf("import phase transition publish failed for %s: %w", patp, publishErr)
	}
	return workflowErr
}

func HTTPUploadHandler(w http.ResponseWriter, r *http.Request, uploadSession, patp string, runtime ...importerRuntime) {
	setUploadResponseHeaders(w)

	// Handle pre-flight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set timeouts on the request context
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)

	validated, err := validateUploadRequest(r, uploadSession, patp, runtime...)
	if err != nil {
		if err := failUploadRequest(w, http.StatusUnauthorized, err.Error(), runtime...); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}

	chunk, err := readUploadChunk(r)
	if err != nil {
		if err := failUploadRequest(w, http.StatusBadRequest, err.Error(), runtime...); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}
	defer func() {
		if err := resource.JoinCloseError(nil, chunk.File, "close uploaded chunk stream"); err != nil {
			logger.Errorf("failed to close uploaded chunk stream: %v", err)
		}
	}()

	pipelineResult := runImportUploadPipeline(validated, chunk, runtime...)
	if pipelineResult.err != nil {
		if err := failUploadRequest(w, pipelineResult.statusCode, pipelineResult.err.Error(), runtime...); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}
	if pipelineResult.completed {
		if err := sendUploadResponse(w, http.StatusOK, "success", "Upload successful"); err != nil {
			logger.Errorf("failed to write upload completion response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
			return
		}
		ClearUploadSession(validated.SessionID)
		return
	}

	// If we get here, just acknowledge the chunk reception
	if err := sendUploadResponse(w, http.StatusOK, "success", "Chunk received successfully"); err != nil {
		logger.Errorf("failed to write upload progress response: %v", err)
		http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		return
	}
}

type uploadPipelineResult struct {
	completed  bool
	statusCode int
	err        error
}

type importUploadPipeline struct {
	validated       validatedUploadRequest
	chunk           uploadChunkPayload
	session         uploadSession
	completionState importCompletionState
}

type importCompletionState struct {
	Error      error
	IsComplete bool
}

func runImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload, runtime ...importerRuntime) uploadPipelineResult {
	pipeline := newImportUploadPipeline(validated, chunk)
	return pipeline.run(runtime...)
}

func newImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload) *importUploadPipeline {
	return &importUploadPipeline{
		validated: validated,
		chunk:     chunk,
	}
}

func (pipeline *importUploadPipeline) run(runtime ...importerRuntime) uploadPipelineResult {
	logger.Debug(fmt.Sprintf("Upload session information for %v: %+v", pipeline.validated.SessionID, pipeline.session))
	if err := publishUploadStatus(pipeline.validated.Patp, runtime...); err != nil {
		logger.Warnf("failed to publish upload status for %s: %v", pipeline.validated.Patp, err)
	}
	if err := runImportPhases(
		pipeline.chunk.Filename,
		pipeline.validated.Patp,
		pipeline.validated.Session.CustomDrive,
		workflowOrchestration.WorkflowPhases{
			Prepare: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("preparing"),
					Run:   pipeline.prepareSession,
				},
			},
			Execute: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("persisting"),
					Run:   pipeline.persistChunk,
				},
			},
			Post: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("finalizing"),
					Run: func() error {
						return pipeline.setUploadFinalizationState(runtime...)
					},
				},
			},
		},
		runtime...,
	); err != nil {
		if pipeline.completionState.Error != nil {
			switch {
			case errors.Is(pipeline.completionState.Error, errChunkCombineTimeout):
				return uploadPipelineResult{
					completed:  pipeline.completionState.IsComplete,
					statusCode: http.StatusRequestTimeout,
					err:        pipeline.completionState.Error,
				}
			case errors.Is(pipeline.completionState.Error, errImportPierConfig):
				return uploadPipelineResult{
					completed:  pipeline.completionState.IsComplete,
					statusCode: http.StatusInternalServerError,
					err:        pipeline.completionState.Error,
				}
			}
			return uploadPipelineResult{
				completed:  pipeline.completionState.IsComplete,
				statusCode: http.StatusBadRequest,
				err:        pipeline.completionState.Error,
			}
		}
		return uploadPipelineResult{
			completed:  false,
			statusCode: http.StatusBadRequest,
			err:        err,
		}
	}
	return uploadPipelineResult{
		completed: pipeline.completionState.IsComplete,
	}
}

func (pipeline *importUploadPipeline) prepareSession() error {
	session, err := prepareUploadSessionForChunk(pipeline.validated.SessionID, pipeline.validated.Session)
	if err != nil {
		return fmt.Errorf("prepare upload session for %s: %w", pipeline.validated.Patp, err)
	}
	pipeline.session = session
	pipeline.validated.Session = session
	return nil
}

func (pipeline *importUploadPipeline) persistChunk() error {
	return persistUploadedChunk(pipeline.chunk.Filename, pipeline.chunk.Index, pipeline.chunk.File)
}

func (pipeline *importUploadPipeline) setUploadFinalizationState(runtime ...importerRuntime) error {
	allChunks, err := allChunksReceived(pipeline.chunk.Filename, pipeline.chunk.Total)
	if err != nil {
		return fmt.Errorf("read chunk manifest for %s: %w", pipeline.chunk.Filename, err)
	}
	completed, err := finalizeUploadOnCompletion(
		uploadChunkProgress{
			Filename:  pipeline.chunk.Filename,
			Total:     pipeline.chunk.Total,
			AllChunks: allChunks,
		},
		pipeline.validated,
		runtime...,
	)
	pipeline.completionState = importCompletionState{
		Error:      err,
		IsComplete: completed,
	}
	if err != nil {
		return fmt.Errorf("finalize upload on completion for %s: %w", pipeline.chunk.Filename, err)
	}
	return nil
}

func publishUploadStatus(patp string, runtime ...importerRuntime) error {
	if err := publishImportTransitionBestEffort(structs.UploadTransition{Type: "patp", Event: patp}, runtime...); err != nil {
		return err
	}
	if err := publishImportStatusBestEffort("uploading", runtime...); err != nil {
		return err
	}
	return nil
}

func prepareUploadSessionForChunk(sessionID string, session uploadSession) (uploadSession, error) {
	return ensureUploadDriveReady(sessionID, session)
}

func readUploadChunk(r *http.Request) (uploadChunkPayload, error) {
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return uploadChunkPayload{}, fmt.Errorf("failed to read uploaded file: %w", err)
	}

	index, total, err := parseUploadChunkMetadata(r, fileHeader.Filename)
	if err != nil {
		return uploadChunkPayload{}, resource.JoinCloseError(
			fmt.Errorf("parse upload chunk metadata for %s: %w", fileHeader.Filename, err),
			file,
			"close uploaded chunk stream",
		)
	}
	return uploadChunkPayload{
		Filename: fileHeader.Filename,
		Index:    index,
		Total:    total,
		File:     file,
	}, nil
}

func persistUploadedChunk(filename string, index int, data io.Reader) error {
	return persistChunkToTemp(data, filename, index)
}

func finalizeUploadOnCompletion(progress uploadChunkProgress, validated validatedUploadRequest, runtime ...importerRuntime) (bool, error) {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return false, fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	if !progress.AllChunks {
		return false, nil
	}
	if err := combineChunksWithTimeout(progress.Filename, progress.Total, 30*time.Minute); err != nil {
		if errors.Is(err, errChunkCombineTimeout) {
			return true, fmt.Errorf("timed out combining upload chunks for %s: %w", progress.Filename, errChunkCombineTimeout)
		}
		return true, fmt.Errorf("failed to combine chunks: %w", err)
	}
	dispatchCmd := toUploadImportCommand(validated, progress)
	if err := resolvedRuntime.UploadCoordinatorFn(validated.Context, dispatchCmd); err != nil {
		return true, fmt.Errorf("failed to finalize imported pier config: %w", errors.Join(errImportPierConfig, err))
	}
	return true, nil
}

func toUploadImportCommand(validated validatedUploadRequest, progress uploadChunkProgress) shipworkflow.UploadImportCommand {
	return shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, progress.Filename),
		Filename:    progress.Filename,
		Patp:        validated.Patp,
		Remote:      validated.Session.Remote,
		Fix:         validated.Session.Fix,
		CustomDrive: validated.Session.CustomDrive,
	}
}

func allChunksReceived(filename string, total int, runtime ...importerRuntime) (bool, error) {
	resolvedRuntime, err := resolveImporterRuntimeForUpload(runtime...)
	if err != nil {
		return false, fmt.Errorf("prepare importer upload runtime: %w", err)
	}
	for i := 0; i < total; i++ {
		partPath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))
		exists, err := fileExists(partPath, resolvedRuntime)
		if err != nil {
			return false, fmt.Errorf("check chunk part %d for %s: %w", i, filename, err)
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

func fileExists(path string, runtime ...importerRuntime) (bool, error) {
	resolvedRuntime, err := resolveImporterRuntimeForUpload(runtime...)
	if err != nil {
		return false, fmt.Errorf("prepare importer upload runtime: %w", err)
	}
	_, err = resolvedRuntime.StatFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path %s: %w", path, err)
	}
	return true, nil
}

func combineChunks(filename string, total int) (err error) {
	destFilePath := filepath.Join(uploadDir, filename)
	destFile, err := os.Create(destFilePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		err = resource.JoinCloseError(err, destFile, fmt.Sprintf("close destination file %s", destFilePath))
	}()

	// Use a buffer to improve performance
	buffer := make([]byte, 32*1024) // 32KB buffer

	for i := 0; i < total; i++ {
		partFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))

		// Open each chunk file
		partFile, err := os.Open(partFilePath)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %d: %w", i, err)
		}

		// Stream the chunk file to the destination using the buffer
		_, copyErr := io.CopyBuffer(destFile, partFile, buffer)
		if closeErr := partFile.Close(); closeErr != nil {
			if copyErr != nil {
				return errors.Join(
					fmt.Errorf("failed to copy chunk %d: %w", i, copyErr),
					fmt.Errorf("failed to close chunk file %q: %w", partFilePath, closeErr),
				)
			}
			return fmt.Errorf("failed to close chunk file %q: %w", partFilePath, closeErr)
		}

		if copyErr != nil {
			return fmt.Errorf("failed to copy chunk %d: %w", i, copyErr)
		}

		// Remove the chunk file after successful copy
		if err := os.Remove(partFilePath); err != nil {
			logger.Warn(fmt.Sprintf("failed to remove chunk file %s: %v", partFilePath, err))
			// Continue despite error removing the file
		}
	}

	return nil
}

func configureUploadedPier(ctx context.Context, cmd shipworkflow.UploadImportCommand, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	defer resolvedRuntime.CleanupMultipartFn(tempDir)
	pierCtx, err := newImportedPierContext(ctx, resolvedRuntime, cmd)
	if err != nil {
		return fmt.Errorf("build imported pier context: %w", err)
	}
	if err := resolvedRuntime.RunImportedPierWorkflowFn(pierCtx, resolvedRuntime); err != nil {
		return fmt.Errorf("run imported pier workflow for %s: %w", pierCtx.Patp, err)
	}
	if err := resolvedRuntime.RunImportedPierPostImportWorkflowFn(pierCtx, resolvedRuntime); err != nil {
		return fmt.Errorf("run imported pier post workflow for %s: %w", pierCtx.Patp, err)
	}
	return nil
}

type importedPierContext struct {
	Context     context.Context
	ArchivePath string
	Filename    string
	Patp        string
	Remote      bool
	Fix         bool
	CustomDrive string
	VolumePath  string
}

func newImportedPierContext(ctx context.Context, runtime importerRuntime, cmd shipworkflow.UploadImportCommand) (importedPierContext, error) {
	requestCtx := ctx
	if requestCtx == nil {
		requestCtx = context.Background()
	}
	if cmd.Patp == "" {
		return importedPierContext{}, fmt.Errorf("upload command missing ship name")
	}
	if _, err := runtime.StoragePathForFn("uploads"); err != nil {
		return importedPierContext{}, fmt.Errorf("resolve upload storage path for %s: %w", cmd.Patp, err)
	}
	if _, err := runtime.StoragePathForFn("temp"); err != nil {
		return importedPierContext{}, fmt.Errorf("resolve temp storage path for %s: %w", cmd.Patp, err)
	}
	volumePath, err := runtime.ResolveImportedPierVolumePathFn(requestCtx, cmd.Patp)
	if err != nil {
		return importedPierContext{}, fmt.Errorf("failed to resolve import volume path for %s: %w", cmd.Patp, err)
	}
	if cmd.CustomDrive != "" {
		volumePath = filepath.Join(cmd.CustomDrive, cmd.Patp)
	}
	archivePath := cmd.ArchivePath
	if archivePath == "" {
		if cmd.Filename == "" {
			return importedPierContext{}, fmt.Errorf("archive path and filename missing for %s", cmd.Patp)
		}
		archivePath = filepath.Join(uploadDir, cmd.Filename)
	}
	return importedPierContext{
		Context:     requestCtx,
		ArchivePath: archivePath,
		Filename:    cmd.Filename,
		Patp:        cmd.Patp,
		Remote:      cmd.Remote,
		Fix:         cmd.Fix,
		CustomDrive: cmd.CustomDrive,
		VolumePath:  volumePath,
	}, nil
}

func runImportedPierWorkflow(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare imported pier workflow for %s: %w", ctx.Patp, err)
	}
	return fmt.Errorf("run imported pier workflow for %s: %w", ctx.Patp, resolvedRuntime.RunImportedPierWorkflowFn(ctx, resolvedRuntime))
}

func runImportedPierWorkflowDefault(ctx importedPierContext, runtime importerRuntime) error {
	return runImportPhases(
		ctx.Filename,
		ctx.Patp,
		ctx.CustomDrive,
		workflowOrchestration.WorkflowPhases{
			Prepare: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("creating"),
					Run: func() error {
						return prepareImportedPierEnvironment(ctx, runtime)
					},
				},
			},
			Execute: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("extracting"),
					Run: func() error {
						return extractUploadedPier(ctx, runtime)
					},
				},
				{
					Phase: lifecycle.Phase("booting"),
					Run: func() error {
						return bootImportedPier(ctx, runtime)
					},
				},
			},
		},
		runtime,
	)
}

func runImportedPierPostImportWorkflow(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow for %s: %w", ctx.Patp, err)
	}
	if err := resolvedRuntime.RunImportedPierPostImportWorkflowFn(ctx, resolvedRuntime); err != nil {
		logger.Error(fmt.Sprintf("Imported pier post-processing failed for %s: %v", ctx.Patp, err))
		return fmt.Errorf("import pier post-processing for %s: %w", ctx.Patp, err)
	}
	return nil
}

func runImportedPierPostImportWorkflowDefault(ctx importedPierContext, runtime importerRuntime) error {
	return runImportedPierPostProcessWorkflow(ctx, runtime)
}

func runImportedPierPostProcessWorkflow(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	return runImportPhases(
		ctx.Filename,
		ctx.Patp,
		ctx.CustomDrive,
		workflowOrchestration.WorkflowPhases{
			Post: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("registering"),
					Run: func() error {
						return startImportedPierRegistration(ctx.Patp, resolvedRuntime)
					},
				},
				{
					Phase: lifecycle.Phase("finalizing"),
					Run: func() error {
						return finalizeImportedPierReadiness(ctx, resolvedRuntime)
					},
				},
			},
		},
		runtime...,
	)
}

func startImportedPierRegistration(patp string, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	startramSettings := resolvedRuntime.StartramSettingsFn()
	if startramSettings.WgRegistered {
		return registerServices(patp)
	}
	return nil
}

func finalizeImportedPierReadiness(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	logger.Info(fmt.Sprintf("Booting ship: %v", ctx.Patp))
	resolvedRuntime.ShipworkflowWaitForBootCodeFn(ctx.Patp, 1*time.Second)
	if ctx.Fix {
		if err := resolvedRuntime.AcmeFixFn(ctx.Patp); err != nil {
			wrappedErr := fmt.Errorf("failed to apply ACME fix for imported ship %s: %w", ctx.Patp, err)
			logger.Error(wrappedErr.Error())
			return wrappedErr
		}
	}
	startramSettings := resolvedRuntime.StartramSettingsFn()
	if startramSettings.WgRegistered && startramSettings.WgOn && ctx.Remote {
		if err := publishImportStatus("remote", runtime...); err != nil {
			return fmt.Errorf("failed to publish remote status for imported ship %s: %w", ctx.Patp, err)
		}
		resolvedRuntime.ShipworkflowWaitForRemoteReadyFn(ctx.Patp, 1*time.Second)
		if err := resolvedRuntime.ShipworkflowSwitchToWireguardFn(ctx.Patp, true); err != nil {
			wrappedErr := fmt.Errorf("failed to switch imported ship %s to Wireguard: %w", ctx.Patp, err)
			logger.Error(wrappedErr.Error())
			return wrappedErr
		}
	}
	if err := publishImportStatus("completed", runtime...); err != nil {
		return fmt.Errorf("failed to publish completed status for imported ship %s: %w", ctx.Patp, err)
	}
	return nil
}

func prepareImportedPierEnvironment(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	if err := resolvedRuntime.ShipcreatorCreateUrbitConfigFn(ctx.Patp, ctx.CustomDrive); err != nil {
		return fmt.Errorf("failed to create urbit config: %w", err)
	}
	if err := resolvedRuntime.ShipcreatorAppendSysConfigPierFn(ctx.Patp); err != nil {
		return fmt.Errorf("failed to update system.json: %w", err)
	}
	logger.Info(fmt.Sprintf("Preparing environment for pier: %v", ctx.Patp))
	if err := resolvedRuntime.DeleteContainerFn(ctx.Patp); err != nil {
		if !isIgnorableCleanupDeleteError(err) {
			return fmt.Errorf("failed to clean up pre-existing container %s: %w", ctx.Patp, err)
		}
		logger.Info(fmt.Sprintf("ignoring pre-existing container cleanup error for %s: %v", ctx.Patp, err))
	}
	if err := resolvedRuntime.DeleteVolumeFn(ctx.Patp); err != nil {
		if !isIgnorableCleanupDeleteError(err) {
			return fmt.Errorf("failed to clean up pre-existing volume %s: %w", ctx.Patp, err)
		}
		logger.Info(fmt.Sprintf("ignoring pre-existing volume cleanup error for %s: %v", ctx.Patp, err))
	}
	if ctx.CustomDrive == "" {
		if err := resolvedRuntime.CreateVolumeFn(ctx.Patp); err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
		return nil
	}
	if err := resolvedRuntime.MkdirAllFn(ctx.CustomDrive, os.ModePerm); err != nil {
		return fmt.Errorf("create custom pier directory error: %w", err)
	}
	return nil
}

func isIgnorableCleanupDeleteError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	if errdefs.IsNotFound(err) {
		return true
	}
	if strings.Contains(strings.ToLower(err.Error()), "no such volume") {
		return true
	}
	return false
}

func extractUploadedPier(ctx importedPierContext, runtime ...importerRuntime) error {
	extractionDone := make(chan struct{})
	defer close(extractionDone)
	go monitorExtractionProgress(extractionDone, runtime...)

	compressedPath := ctx.ArchivePath
	if err := extractUploadedArchive(compressedPath, ctx.VolumePath, ctx.Filename); err != nil {
		return fmt.Errorf("failed to extract %v: %w", ctx.Filename, err)
	}

	logger.Debug(fmt.Sprintf("%v extracted to %v", ctx.Filename, ctx.VolumePath))
	if err := restructureDirectory(ctx); err != nil {
		return fmt.Errorf("failed to restructure directory: %w", err)
	}
	return nil
}

func monitorExtractionProgress(done <-chan struct{}, runtime ...importerRuntime) {
	extractionTimeout := time.NewTimer(4 * time.Hour)
	select {
	case <-extractionTimeout.C:
		if err := publishImportTransitionBestEffort(structs.UploadTransition{Type: "extracted", Value: 100}, runtime...); err != nil {
			logger.Warnf("failed to publish extraction timeout transition: %v", err)
		}
		if err := publishImportStatusBestEffort("checking", runtime...); err != nil {
			logger.Warnf("failed to publish checking status after extraction timeout: %v", err)
		}
	case <-done:
		extractionTimeout.Stop()
	}
}

func bootImportedPier(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow runtime: %w", err)
	}
	logger.Info(fmt.Sprintf("Starting extracted pier: %v", ctx.Patp))
	info, err := resolvedRuntime.StartContainerFn(ctx.Patp, "vere")
	if err != nil {
		return fmt.Errorf("failed to start imported ship: %w", err)
	}
	resolvedRuntime.UpdateContainerStateFn(ctx.Patp, info)
	if err := os.Remove(ctx.ArchivePath); err != nil {
		logger.Warn(fmt.Sprintf("failed to remove uploaded archive %s: %v", ctx.Filename, err))
	}
	return nil
}

func errorCleanup(filename, patp, customDrive string, err error, runtime ...importerRuntime) error {
	// notify that we are cleaning up
	logger.Info(fmt.Sprintf("Pier import process failed: %s: %v", patp, err))
	logger.Info(fmt.Sprintf("Running cleanup routine"))

	customPierPath := ""
	if customDrive != "" {
		customPierPath = filepath.Join(customDrive, patp)
	}
	if err := shipcleanup.RollbackProvisioning(patp, shipcleanup.RollbackOptions{
		UploadArchivePath:    filepath.Join(uploadDir, filename),
		CustomPierPath:       customPierPath,
		RemoveContainer:      true,
		RemoveContainerState: true,
	}); err != nil {
		logger.Error(fmt.Sprintf("Import rollback encountered errors: %v", err))
		return err
	}
	if publishErr := publishImportError(err.Error(), runtime...); publishErr != nil {
		logger.Warnf("failed to publish import cleanup error for %s: %v", patp, publishErr)
	}
	if publishErr := publishImportStatus("aborted", runtime...); publishErr != nil {
		logger.Warnf("failed to publish aborted status for %s: %v", patp, publishErr)
	}
	return nil
}

func restructureDirectory(ctx importedPierContext) error {
	patp := ctx.Patp
	volDir := ctx.VolumePath
	if volDir == "" {
		return fmt.Errorf("no docker volume for %s", patp)
	}

	logger.Info("Checking pier directory")
	logger.Info(fmt.Sprintf("%v pier path: %v", patp, volDir))
	// find .urb
	var urbLoc []string
	if err := filepath.Walk(volDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk volume directory %s: %w", volDir, err)
		}
		if info.IsDir() && filepath.Base(path) == ".urb" && !strings.Contains(path, "__MACOSX") {
			urbLoc = append(urbLoc, filepath.Dir(path))
		}
		return nil
	}); err != nil {
		return fmt.Errorf("find ship directory in volume %s: %w", volDir, err)
	}
	// there can only be one
	if len(urbLoc) > 1 {
		return fmt.Errorf("%v ships detected in pier directory", len(urbLoc))
	}
	if len(urbLoc) < 1 {
		return fmt.Errorf("no ship found in pier directory")
	}
	logger.Debug(fmt.Sprintf(".urb subdirectory in %v", urbLoc[0]))
	pierDir := filepath.Join(volDir, patp)
	tempDir := filepath.Join(volDir, "temp_dir")
	unusedDir := filepath.Join(volDir, "unused")
	// move it into the right place
	if filepath.Join(pierDir, ".urb") != filepath.Join(urbLoc[0], ".urb") {
		logger.Info(".urb location incorrect! Restructuring directory structure")
		logger.Debug(fmt.Sprintf(".urb found in %v", urbLoc[0]))
		logger.Debug(fmt.Sprintf("Moving to %v", tempDir))
		if volDir == urbLoc[0] { // .urb in root
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				return fmt.Errorf("create temp import directory %s: %w", tempDir, err)
			}
			items, err := ioutil.ReadDir(urbLoc[0])
			if err != nil {
				return fmt.Errorf("read pier directory %s: %w", urbLoc[0], err)
			}
			for _, item := range items {
				if item.Name() != patp {
					if err := os.Rename(filepath.Join(urbLoc[0], item.Name()), filepath.Join(tempDir, item.Name())); err != nil {
						return fmt.Errorf("move %s to temp directory: %w", item.Name(), err)
					}
				}
			}
		} else {
			if err := os.Rename(urbLoc[0], tempDir); err != nil {
				return fmt.Errorf("move %s to temp directory: %w", urbLoc[0], err)
			}
		}
		unused := []string{}
		dirs, err := ioutil.ReadDir(volDir)
		if err != nil {
			return fmt.Errorf("read import directory %s: %w", volDir, err)
		}
		for _, dir := range dirs {
			dirName := dir.Name()
			if dirName != "temp_dir" && dirName != "unused" {
				unused = append(unused, dirName)
			}
		}
		if len(unused) > 0 {
			if err := os.MkdirAll(unusedDir, 0755); err != nil {
				return fmt.Errorf("create unused directory %s: %w", unusedDir, err)
			}
			for _, u := range unused {
				if err := os.Rename(filepath.Join(volDir, u), filepath.Join(unusedDir, u)); err != nil {
					return fmt.Errorf("archive directory %s: %w", u, err)
				}
			}
		}
		if err := os.Rename(tempDir, pierDir); err != nil {
			return fmt.Errorf("finalize pier directory %s: %w", pierDir, err)
		}
		logger.Info(fmt.Sprintf("%v restructuring done", patp))
	} else {
		logger.Debug("No restructuring needed")
	}
	return nil
}

func registerServices(patp string) error {
	if err := shipworkflow.RegisterShipServices(patp); err != nil {
		logger.Error(fmt.Sprintf("unable to register startram service for %s: %v", patp, err))
		return fmt.Errorf("register ship services for %s: %w", patp, err)
	}
	return nil
}
