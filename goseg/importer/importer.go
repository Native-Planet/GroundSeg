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

type importerConfigRuntime struct {
	storagePathForFn             func(string) (string, error)
	swapSettingsFn               func() config.SwapSettings
	startramSettingsFn           func() config.StartramSettings
	validateUploadSessionTokenFn  func(structs.WsTokenStruct, structs.WsTokenStruct, *http.Request) auth.UploadTokenAuthorizationResult
}

type importerWorkflowRuntime struct {
	resolveImportedPierVolumePathFn     func(context.Context, string) (string, error)
	runImportedPierWorkflowFn           func(importedPierContext, importerRuntime) error
	runImportedPierPostImportWorkflowFn func(importedPierContext, importerRuntime) error
	cleanupMultipartFn                  func(string) error
	uploadCoordinatorFn                 shipworkflow.UploadImportCoordinator
}

type importerLifecycleRuntime struct {
	shipworkflowWaitForBootCodeFn    func(string, time.Duration)
	shipworkflowWaitForRemoteReadyFn func(string, time.Duration)
	shipworkflowSwitchToWireguardFn  func(string, bool) error
	acmeFixFn                        func(string) error
}

type importerShipConfigRuntime struct {
	shipcreatorCreateUrbitConfigFn   func(string, string) error
	shipcreatorAppendSysConfigPierFn func(string) error
}

type importerContainerRuntime struct {
	deleteContainerFn func(string) error
	startContainerFn  func(string, string) (structs.ContainerState, error)
	updateContainerFn func(string, structs.ContainerState)
}

type importerVolumeRuntime struct {
	dockerDeleteVolumeFn func(string) error
	dockerCreateVolumeFn func(string) error
}

type importerFilesystemRuntime struct {
	statFn          func(string) (os.FileInfo, error)
	mkdirAllFn      func(string, os.FileMode) error
	createDirFn     func(string, os.FileMode) error
	closeTempFileFn func(*os.File) error
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

type importerRuntime struct {
	dockerOrchestration.RuntimeTransitionOps
	configOps     *importerConfigRuntime
	workflowOps   *importerWorkflowRuntime
	lifecycleOps  *importerLifecycleRuntime
	shipConfigOps *importerShipConfigRuntime
	containerOps  *importerContainerRuntime
	volumeOps     *importerVolumeRuntime
	filesOps      *importerFilesystemRuntime
}

func defaultImporterRuntime() importerRuntime {
	orchestrationRuntime := dockerOrchestration.NewRuntime()
	return importerRuntime{
		RuntimeTransitionOps: orchestrationRuntime.RuntimeTransitionOps,
		configOps: &importerConfigRuntime{
			storagePathForFn:             config.GetStoragePath,
			swapSettingsFn:               config.SwapSettingsSnapshot,
			startramSettingsFn:           config.StartramSettingsSnapshot,
			validateUploadSessionTokenFn: auth.ValidateUploadSessionTokenRequest,
		},
		workflowOps: &importerWorkflowRuntime{
			resolveImportedPierVolumePathFn: func(_ context.Context, patp string) (string, error) {
				return resolveImportedPierVolumePath(patp)
			},
			runImportedPierWorkflowFn:           runImportedPierWorkflowDefault,
			runImportedPierPostImportWorkflowFn: runImportedPierPostImportWorkflowDefault,
			cleanupMultipartFn:                  system.RemoveMultipartFiles,
			uploadCoordinatorFn: func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
				return shipworkflow.DispatchUploadImportWithCoordinator(func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
					return configureUploadedPier(ctx, cmd)
				}, ctx, cmd)
			},
		},
		lifecycleOps: &importerLifecycleRuntime{
			shipworkflowWaitForBootCodeFn:    shipworkflow.WaitForBootCode,
			shipworkflowWaitForRemoteReadyFn: shipworkflow.WaitForRemoteReady,
			shipworkflowSwitchToWireguardFn:  shipworkflow.SwitchShipToWireguard,
			acmeFixFn:                        acme.Fix,
		},
		shipConfigOps: &importerShipConfigRuntime{
			shipcreatorCreateUrbitConfigFn:   shipcreator.CreateUrbitConfig,
			shipcreatorAppendSysConfigPierFn: shipcreator.AppendSysConfigPier,
		},
		containerOps: &importerContainerRuntime{
			deleteContainerFn: orchestrationRuntime.DeleteContainerFn,
			startContainerFn:  orchestrationRuntime.StartContainerFn,
			updateContainerFn: orchestrationRuntime.UpdateContainerStateFn,
		},
		volumeOps: &importerVolumeRuntime{
			dockerDeleteVolumeFn: dockerOrchestration.DeleteVolume,
			dockerCreateVolumeFn: dockerOrchestration.CreateVolume,
		},
		filesOps: &importerFilesystemRuntime{
			mkdirAllFn:      os.MkdirAll,
			createDirFn:     os.MkdirAll,
			closeTempFileFn: func(file *os.File) error { return file.Close() },
			statFn:          os.Stat,
		},
	}
}

func (runtime importerRuntime) ensureInitializedForInit() error {
	if runtime.configOps == nil {
		return errImporterStoragePathMissing
	}
	if err := runtime.configOps.validateForInit(); err != nil {
		return err
	}
	if runtime.filesOps == nil {
		return errImporterCreateDirMissing
	}
	return runtime.filesOps.validateForInit()
}

func (runtime importerRuntime) ensureForWorkflow() error {
	if runtime.configOps == nil {
		return errImporterTokenValidationCallbackMissing
	}
	if err := runtime.configOps.validate(); err != nil {
		return err
	}
	if runtime.workflowOps == nil {
		return errImporterWorkflowMissing
	}
	if err := runtime.workflowOps.validate(); err != nil {
		return err
	}
	if runtime.lifecycleOps == nil {
		return errImporterBootCodeWaitMissing
	}
	if err := runtime.lifecycleOps.validate(); err != nil {
		return err
	}
	if runtime.shipConfigOps == nil {
		return errImporterUrbitConfigMissing
	}
	if err := runtime.shipConfigOps.validate(); err != nil {
		return err
	}
	if runtime.containerOps == nil {
		return errImporterContainerDeleteMissing
	}
	if err := runtime.containerOps.validate(); err != nil {
		return err
	}
	if runtime.volumeOps == nil {
		return errImporterVolumeDeleteMissing
	}
	if err := runtime.volumeOps.validate(); err != nil {
		return err
	}
	if runtime.filesOps == nil {
		return errImporterFileStatusMissing
	}
	if err := runtime.filesOps.validate(); err != nil {
		return err
	}
	return nil
}

func (ops *importerConfigRuntime) storagePathFor(pathType string) (string, error) {
	if ops == nil || ops.storagePathForFn == nil {
		return "", errImporterStoragePathMissing
	}
	return ops.storagePathForFn(pathType)
}

func (ops *importerConfigRuntime) swapSettings() config.SwapSettings {
	if ops == nil || ops.swapSettingsFn == nil {
		return config.SwapSettings{}
	}
	return ops.swapSettingsFn()
}

func (ops *importerConfigRuntime) startramSettings() config.StartramSettings {
	if ops == nil || ops.startramSettingsFn == nil {
		return config.StartramSettings{}
	}
	return ops.startramSettingsFn()
}

func (ops *importerConfigRuntime) validateUploadSessionToken(expectedToken, providedToken structs.WsTokenStruct, req *http.Request) auth.UploadTokenAuthorizationResult {
	if ops == nil || ops.validateUploadSessionTokenFn == nil {
		return auth.UploadTokenAuthorizationResult{Status: tokens.UploadValidationStatusNotAuthorized, AuthorizationErr: errImporterTokenValidationCallbackMissing}
	}
	return ops.validateUploadSessionTokenFn(expectedToken, providedToken, req)
}

func (ops *importerConfigRuntime) validate() error {
	if ops == nil {
		return errImporterTokenValidationCallbackMissing
	}
	if ops.storagePathForFn == nil {
		return errImporterStoragePathMissing
	}
	if ops.swapSettingsFn == nil {
		return errImporterSwapSettingsMissing
	}
	if ops.startramSettingsFn == nil {
		return errImporterStartramSettingsMissing
	}
	if ops.validateUploadSessionTokenFn == nil {
		return errImporterTokenValidationCallbackMissing
	}
	return nil
}

func (ops *importerConfigRuntime) validateForInit() error {
	if ops == nil {
		return errImporterStoragePathMissing
	}
	if ops.storagePathForFn == nil {
		return errImporterStoragePathMissing
	}
	if ops.swapSettingsFn == nil {
		return errImporterSwapSettingsMissing
	}
	return nil
}

func (ops *importerWorkflowRuntime) resolveImportedPierVolumePath(ctx context.Context, patp string) (string, error) {
	if ops == nil || ops.resolveImportedPierVolumePathFn == nil {
		return "", errImporterVolumeResolutionMissing
	}
	return ops.resolveImportedPierVolumePathFn(ctx, patp)
}

func (ops *importerWorkflowRuntime) runImportedPierWorkflow(runtimeContext importedPierContext, runtime importerRuntime) error {
	if ops == nil || ops.runImportedPierWorkflowFn == nil {
		return errImporterWorkflowMissing
	}
	return ops.runImportedPierWorkflowFn(runtimeContext, runtime)
}

func (ops *importerWorkflowRuntime) runImportedPierPostImportWorkflow(runtimeContext importedPierContext, runtime importerRuntime) error {
	if ops == nil || ops.runImportedPierPostImportWorkflowFn == nil {
		return errImporterPostImportWorkflowMissing
	}
	return ops.runImportedPierPostImportWorkflowFn(runtimeContext, runtime)
}

func (ops *importerWorkflowRuntime) cleanupMultipart(path string) error {
	if ops == nil || ops.cleanupMultipartFn == nil {
		return errImporterCleanupMissing
	}
	return ops.cleanupMultipartFn(path)
}

func (ops *importerWorkflowRuntime) dispatchUploadImport(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
	if ops == nil || ops.uploadCoordinatorFn == nil {
		return errUploadImportCoordinatorUnconfigured
	}
	return ops.uploadCoordinatorFn(ctx, cmd)
}

func (ops *importerWorkflowRuntime) validate() error {
	if ops == nil {
		return errImporterWorkflowMissing
	}
	if ops.resolveImportedPierVolumePathFn == nil {
		return errImporterVolumeResolutionMissing
	}
	if ops.runImportedPierWorkflowFn == nil {
		return errImporterWorkflowMissing
	}
	if ops.runImportedPierPostImportWorkflowFn == nil {
		return errImporterPostImportWorkflowMissing
	}
	if ops.cleanupMultipartFn == nil {
		return errImporterCleanupMissing
	}
	if ops.uploadCoordinatorFn == nil {
		return errUploadImportCoordinatorUnconfigured
	}
	return nil
}

func (ops *importerLifecycleRuntime) waitForBootCode(patp string, timeout time.Duration) {
	if ops == nil || ops.shipworkflowWaitForBootCodeFn == nil {
		return
	}
	ops.shipworkflowWaitForBootCodeFn(patp, timeout)
}

func (ops *importerLifecycleRuntime) waitForRemoteReady(patp string, timeout time.Duration) {
	if ops == nil || ops.shipworkflowWaitForRemoteReadyFn == nil {
		return
	}
	ops.shipworkflowWaitForRemoteReadyFn(patp, timeout)
}

func (ops *importerLifecycleRuntime) switchToWireguard(patp string, value bool) error {
	if ops == nil || ops.shipworkflowSwitchToWireguardFn == nil {
		return errImporterWireguardSwitchMissing
	}
	return ops.shipworkflowSwitchToWireguardFn(patp, value)
}

func (ops *importerLifecycleRuntime) applyAcmeFix(patp string) error {
	if ops == nil || ops.acmeFixFn == nil {
		return errImporterAcmeFixMissing
	}
	return ops.acmeFixFn(patp)
}

func (ops *importerLifecycleRuntime) validate() error {
	if ops == nil {
		return errImporterBootCodeWaitMissing
	}
	if ops.shipworkflowWaitForBootCodeFn == nil {
		return errImporterBootCodeWaitMissing
	}
	if ops.shipworkflowWaitForRemoteReadyFn == nil {
		return errImporterRemoteReadyWaitMissing
	}
	if ops.shipworkflowSwitchToWireguardFn == nil {
		return errImporterWireguardSwitchMissing
	}
	if ops.acmeFixFn == nil {
		return errImporterAcmeFixMissing
	}
	return nil
}

func (ops *importerShipConfigRuntime) createUrbitConfig(patp, customDrive string) error {
	if ops == nil || ops.shipcreatorCreateUrbitConfigFn == nil {
		return errImporterUrbitConfigMissing
	}
	return ops.shipcreatorCreateUrbitConfigFn(patp, customDrive)
}

func (ops *importerShipConfigRuntime) appendUrbitSysConfig(patp string) error {
	if ops == nil || ops.shipcreatorAppendSysConfigPierFn == nil {
		return errImporterUrbitSysConfigMissing
	}
	return ops.shipcreatorAppendSysConfigPierFn(patp)
}

func (ops *importerShipConfigRuntime) validate() error {
	if ops == nil {
		return errImporterUrbitConfigMissing
	}
	if ops.shipcreatorCreateUrbitConfigFn == nil {
		return errImporterUrbitConfigMissing
	}
	if ops.shipcreatorAppendSysConfigPierFn == nil {
		return errImporterUrbitSysConfigMissing
	}
	return nil
}

func (ops *importerContainerRuntime) deleteContainer(patp string) error {
	if ops == nil || ops.deleteContainerFn == nil {
		return errImporterContainerDeleteMissing
	}
	return ops.deleteContainerFn(patp)
}

func (ops *importerContainerRuntime) startContainer(patp, image string) (structs.ContainerState, error) {
	if ops == nil || ops.startContainerFn == nil {
		return structs.ContainerState{}, errImporterStartContainerMissing
	}
	return ops.startContainerFn(patp, image)
}

func (ops *importerContainerRuntime) updateContainer(patp string, containerState structs.ContainerState) {
	if ops == nil || ops.updateContainerFn == nil {
		return
	}
	ops.updateContainerFn(patp, containerState)
}

func (ops *importerContainerRuntime) validate() error {
	if ops == nil {
		return errImporterContainerDeleteMissing
	}
	if ops.deleteContainerFn == nil {
		return errImporterContainerDeleteMissing
	}
	if ops.startContainerFn == nil {
		return errImporterStartContainerMissing
	}
	if ops.updateContainerFn == nil {
		return errImporterUpdateContainerMissing
	}
	return nil
}

func (ops *importerVolumeRuntime) deleteVolume(patp string) error {
	if ops == nil || ops.dockerDeleteVolumeFn == nil {
		return errImporterVolumeDeleteMissing
	}
	return ops.dockerDeleteVolumeFn(patp)
}

func (ops *importerVolumeRuntime) createVolume(patp string) error {
	if ops == nil || ops.dockerCreateVolumeFn == nil {
		return errImporterVolumeCreateMissing
	}
	return ops.dockerCreateVolumeFn(patp)
}

func (ops *importerVolumeRuntime) validate() error {
	if ops == nil {
		return errImporterVolumeDeleteMissing
	}
	if ops.dockerDeleteVolumeFn == nil {
		return errImporterVolumeDeleteMissing
	}
	if ops.dockerCreateVolumeFn == nil {
		return errImporterVolumeCreateMissing
	}
	return nil
}

func (ops *importerFilesystemRuntime) stat(path string) (os.FileInfo, error) {
	if ops == nil || ops.statFn == nil {
		return nil, errImporterFileStatusMissing
	}
	return ops.statFn(path)
}

func (ops *importerFilesystemRuntime) mkdir(path string, mode os.FileMode) error {
	if ops == nil || ops.mkdirAllFn == nil {
		return errImporterMkdirMissing
	}
	return ops.mkdirAllFn(path, mode)
}

func (ops *importerFilesystemRuntime) createDir(path string, mode os.FileMode) error {
	if ops == nil || ops.createDirFn == nil {
		return errImporterCreateDirMissing
	}
	return ops.createDirFn(path, mode)
}

func (ops *importerFilesystemRuntime) closeTempFile(file *os.File) error {
	if ops == nil || ops.closeTempFileFn == nil {
		return errImporterCloseTempFileMissing
	}
	return ops.closeTempFileFn(file)
}

func (ops *importerFilesystemRuntime) validate() error {
	if ops == nil {
		return errImporterFileStatusMissing
	}
	if ops.statFn == nil {
		return errImporterFileStatusMissing
	}
	if ops.closeTempFileFn == nil {
		return errImporterCloseTempFileMissing
	}
	if ops.mkdirAllFn == nil {
		return errImporterMkdirMissing
	}
	if ops.createDirFn == nil {
		return errImporterCreateDirMissing
	}
	return nil
}

func (ops *importerFilesystemRuntime) validateForInit() error {
	if ops == nil {
		return errImporterCreateDirMissing
	}
	if ops.createDirFn == nil {
		return errImporterCreateDirMissing
	}
	return nil
}

func withDefaultsImporterRuntime(runtime importerRuntime) importerRuntime {
	base := defaultImporterRuntime()
	if runtime.configOps != nil {
		base.configOps = runtime.configOps
	}
	if runtime.workflowOps != nil {
		base.workflowOps = runtime.workflowOps
	}
	if runtime.lifecycleOps != nil {
		base.lifecycleOps = runtime.lifecycleOps
	}
	if runtime.shipConfigOps != nil {
		base.shipConfigOps = runtime.shipConfigOps
	}
	if runtime.containerOps != nil {
		base.containerOps = runtime.containerOps
	}
	if runtime.volumeOps != nil {
		base.volumeOps = runtime.volumeOps
	}
	if runtime.filesOps != nil {
		base.filesOps = runtime.filesOps
	}
	return base
}

func (runtime importerRuntime) storagePathFor(pathType string) (string, error) {
	if runtime.configOps == nil {
		return "", errImporterStoragePathMissing
	}
	if runtime.configOps.storagePathForFn == nil {
		return "", errImporterStoragePathMissing
	}
	return runtime.configOps.storagePathForFn(pathType)
}

func (runtime importerRuntime) swapSettings() config.SwapSettings {
	if runtime.configOps == nil {
		return config.SwapSettings{}
	}
	if runtime.configOps.swapSettingsFn == nil {
		return config.SwapSettings{}
	}
	return runtime.configOps.swapSettingsFn()
}

func (runtime importerRuntime) startramSettings() config.StartramSettings {
	if runtime.configOps == nil {
		return config.StartramSettings{}
	}
	if runtime.configOps.startramSettingsFn == nil {
		return config.StartramSettings{}
	}
	return runtime.configOps.startramSettingsFn()
}

func (runtime importerRuntime) resolveImportedPierVolumePath(ctx context.Context, patp string) (string, error) {
	if runtime.workflowOps == nil {
		return "", errImporterVolumeResolutionMissing
	}
	if runtime.workflowOps.resolveImportedPierVolumePathFn == nil {
		return "", errImporterVolumeResolutionMissing
	}
	return runtime.workflowOps.resolveImportedPierVolumePathFn(ctx, patp)
}

func (runtime importerRuntime) runImportedPierWorkflow(ctx importedPierContext) error {
	if runtime.workflowOps == nil {
		return errImporterWorkflowMissing
	}
	if runtime.workflowOps.runImportedPierWorkflowFn == nil {
		return errImporterWorkflowMissing
	}
	return runtime.workflowOps.runImportedPierWorkflowFn(ctx, runtime)
}

func (runtime importerRuntime) runImportedPostImportWorkflow(ctx importedPierContext) error {
	if runtime.workflowOps == nil {
		return errImporterPostImportWorkflowMissing
	}
	if runtime.workflowOps.runImportedPierPostImportWorkflowFn == nil {
		return errImporterPostImportWorkflowMissing
	}
	return runtime.workflowOps.runImportedPierPostImportWorkflowFn(ctx, runtime)
}

func (runtime importerRuntime) cleanupMultipart(path string) error {
	if runtime.workflowOps == nil {
		return errImporterCleanupMissing
	}
	if runtime.workflowOps.cleanupMultipartFn == nil {
		return errImporterCleanupMissing
	}
	return runtime.workflowOps.cleanupMultipartFn(path)
}

func (runtime importerRuntime) dispatchUploadImport(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
	if runtime.workflowOps == nil {
		return errUploadImportCoordinatorUnconfigured
	}
	if runtime.workflowOps.uploadCoordinatorFn == nil {
		return errUploadImportCoordinatorUnconfigured
	}
	return runtime.workflowOps.uploadCoordinatorFn(ctx, cmd)
}

func (runtime importerRuntime) validateUploadSessionToken(expectedToken, providedToken structs.WsTokenStruct, req *http.Request) auth.UploadTokenAuthorizationResult {
	if runtime.configOps == nil {
		return auth.UploadTokenAuthorizationResult{Status: tokens.UploadValidationStatusNotAuthorized, AuthorizationErr: errImporterTokenValidationCallbackMissing}
	}
	if runtime.configOps.validateUploadSessionTokenFn == nil {
		return auth.UploadTokenAuthorizationResult{Status: tokens.UploadValidationStatusNotAuthorized, AuthorizationErr: errImporterTokenValidationCallbackMissing}
	}
	return runtime.configOps.validateUploadSessionTokenFn(expectedToken, providedToken, req)
}

func (runtime importerRuntime) stat(path string) (os.FileInfo, error) {
	if runtime.filesOps == nil {
		return nil, errImporterFileStatusMissing
	}
	if runtime.filesOps.statFn == nil {
		return nil, errImporterFileStatusMissing
	}
	return runtime.filesOps.statFn(path)
}

func (runtime importerRuntime) waitForBootCode(patp string, timeout time.Duration) {
	if runtime.lifecycleOps != nil {
		if runtime.lifecycleOps.shipworkflowWaitForBootCodeFn != nil {
			runtime.lifecycleOps.shipworkflowWaitForBootCodeFn(patp, timeout)
		}
	}
}

func (runtime importerRuntime) waitForRemoteReady(patp string, timeout time.Duration) {
	if runtime.lifecycleOps != nil {
		if runtime.lifecycleOps.shipworkflowWaitForRemoteReadyFn != nil {
			runtime.lifecycleOps.shipworkflowWaitForRemoteReadyFn(patp, timeout)
		}
	}
}

func (runtime importerRuntime) switchToWireguard(patp string, value bool) error {
	if runtime.lifecycleOps == nil {
		return errImporterWireguardSwitchMissing
	}
	if runtime.lifecycleOps.shipworkflowSwitchToWireguardFn == nil {
		return errImporterWireguardSwitchMissing
	}
	return runtime.lifecycleOps.shipworkflowSwitchToWireguardFn(patp, value)
}

func (runtime importerRuntime) applyAcmeFix(patp string) error {
	if runtime.lifecycleOps == nil {
		return errImporterAcmeFixMissing
	}
	if runtime.lifecycleOps.acmeFixFn == nil {
		return errImporterAcmeFixMissing
	}
	return runtime.lifecycleOps.acmeFixFn(patp)
}

func (runtime importerRuntime) createUrbitConfig(patp, customDrive string) error {
	if runtime.shipConfigOps == nil {
		return errImporterUrbitConfigMissing
	}
	if runtime.shipConfigOps.shipcreatorCreateUrbitConfigFn == nil {
		return errImporterUrbitConfigMissing
	}
	return runtime.shipConfigOps.shipcreatorCreateUrbitConfigFn(patp, customDrive)
}

func (runtime importerRuntime) appendUrbitSysConfig(patp string) error {
	if runtime.shipConfigOps == nil {
		return errImporterUrbitSysConfigMissing
	}
	if runtime.shipConfigOps.shipcreatorAppendSysConfigPierFn == nil {
		return errImporterUrbitSysConfigMissing
	}
	return runtime.shipConfigOps.shipcreatorAppendSysConfigPierFn(patp)
}

func (runtime importerRuntime) deleteContainer(patp string) error {
	if runtime.containerOps == nil {
		return errImporterContainerDeleteMissing
	}
	if runtime.containerOps.deleteContainerFn == nil {
		return errImporterContainerDeleteMissing
	}
	return runtime.containerOps.deleteContainerFn(patp)
}

func (runtime importerRuntime) startContainer(patp, image string) (structs.ContainerState, error) {
	if runtime.containerOps == nil {
		return structs.ContainerState{}, errImporterStartContainerMissing
	}
	if runtime.containerOps.startContainerFn == nil {
		return structs.ContainerState{}, errImporterStartContainerMissing
	}
	return runtime.containerOps.startContainerFn(patp, image)
}

func (runtime importerRuntime) updateContainer(patp string, containerState structs.ContainerState) {
	if runtime.containerOps != nil {
		if runtime.containerOps.updateContainerFn != nil {
			runtime.containerOps.updateContainerFn(patp, containerState)
		}
	}
}

func (runtime importerRuntime) deleteVolume(patp string) error {
	if runtime.volumeOps == nil {
		return errImporterVolumeDeleteMissing
	}
	if runtime.volumeOps.dockerDeleteVolumeFn == nil {
		return errImporterVolumeDeleteMissing
	}
	return runtime.volumeOps.dockerDeleteVolumeFn(patp)
}

func (runtime importerRuntime) createVolume(patp string) error {
	if runtime.volumeOps == nil {
		return errImporterVolumeCreateMissing
	}
	if runtime.volumeOps.dockerCreateVolumeFn == nil {
		return errImporterVolumeCreateMissing
	}
	return runtime.volumeOps.dockerCreateVolumeFn(patp)
}

func (runtime importerRuntime) mkdir(path string, mode os.FileMode) error {
	if runtime.filesOps == nil {
		return errImporterMkdirMissing
	}
	if runtime.filesOps.mkdirAllFn == nil {
		return errImporterMkdirMissing
	}
	return runtime.filesOps.mkdirAllFn(path, mode)
}

func (runtime importerRuntime) closeTempFile(file *os.File) error {
	if runtime.filesOps == nil {
		return errImporterCloseTempFileMissing
	}
	if runtime.filesOps.closeTempFileFn == nil {
		return errImporterCloseTempFileMissing
	}
	return runtime.filesOps.closeTempFileFn(file)
}

func (runtime importerRuntime) createDir(path string, mode os.FileMode) error {
	if runtime.filesOps == nil {
		return errImporterCreateDirMissing
	}
	if runtime.filesOps.createDirFn == nil {
		return errImporterCreateDirMissing
	}
	return runtime.filesOps.createDirFn(path, mode)
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

func Initialize(runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForInit(runtime...)
	if err != nil {
		return fmt.Errorf("initialize importer runtime: %w", err)
	}
	uploadDir, err = resolvedRuntime.storagePathFor("uploads")
	if err != nil {
		return fmt.Errorf("initialize upload directory: %w", err)
	}
	tempDir, err = resolvedRuntime.storagePathFor("temp")
	if err != nil {
		return fmt.Errorf("initialize temp directory: %w", err)
	}
	swapSettings := resolvedRuntime.swapSettings()
	if !strings.HasPrefix(swapSettings.SwapFile, "/opt") {
		var tempPath string
		lastSlashIndex := strings.LastIndex(swapSettings.SwapFile, "/")
		if lastSlashIndex != -1 {
			tempPath = swapSettings.SwapFile[:lastSlashIndex]
			tempDir = filepath.Join(tempPath, "temp")
			uploadDir = filepath.Join(tempPath, "uploads")
		}
	}
	if err := resolvedRuntime.createDir(uploadDir, 0755); err != nil {
		return fmt.Errorf("create upload directory %q: %w", uploadDir, err)
	}
	if err := resolvedRuntime.createDir(tempDir, 0755); err != nil {
		return fmt.Errorf("create temp directory %q: %w", tempDir, err)
	}
	return nil
}

func resolveImporterRuntime(overrides ...importerRuntime) importerRuntime {
	if len(overrides) > 0 {
		return withDefaultsImporterRuntime(overrides[0])
	}
	return defaultImporterRuntime()
}

func resolveImporterRuntimeForWorkflow(overrides ...importerRuntime) (importerRuntime, error) {
	resolvedRuntime := resolveImporterRuntime(overrides...)
	if err := resolvedRuntime.ensureForWorkflow(); err != nil {
		return resolvedRuntime, err
	}
	return resolvedRuntime, nil
}

func resolveImporterRuntimeForInit(overrides ...importerRuntime) (importerRuntime, error) {
	resolvedRuntime := resolveImporterRuntime(overrides...)
	if err := resolvedRuntime.ensureInitializedForInit(); err != nil {
		return resolvedRuntime, err
	}
	return resolvedRuntime, nil
}

func OpenUploadEndpoint(cmd OpenUploadEndpointCmd, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
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

	authz := resolvedRuntime.validateUploadSessionToken(expectedToken, token, nil)
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

func Reset() error {
	publishImportStatus("")
	publishImportTransition(structs.UploadTransition{Type: "patp", Event: ""})
	publishImportError("")
	publishImportTransition(structs.UploadTransition{Type: "extracted", Value: 0})
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

func publishImportStatus(event string) {
	publishImportTransition(structs.UploadTransition{Type: "status", Event: event})
}

func publishImportError(message string) {
	publishImportTransition(structs.UploadTransition{Type: "error", Event: message})
}

func publishImportTransition(uploadTransition structs.UploadTransition) {
	if err := publishImportTransitionWithPolicy(uploadTransition, transition.TransitionPublishBestEffort); err != nil {
		logger.Warnf("failed to publish import transition %+v: %v", uploadTransition, err)
	}
}

func publishImportTransitionWithPolicy(uploadTransition structs.UploadTransition, publishPolicy transition.TransitionPublishPolicy) error {
	err := events.DefaultEventRuntime().PublishImportShipTransition(context.Background(), uploadTransition)
	if err == nil {
		return nil
	}
	return transition.HandleTransitionPublishError(
		fmt.Sprintf("publish upload transition (%s)", uploadTransition.Type),
		err,
		publishPolicy,
	)
}

func failUploadRequest(w http.ResponseWriter, code int, message string) error {
	logger.Errorf("upload error: %v", message)
	publishImportError(message)
	publishImportStatus("aborted")
	return sendUploadResponse(w, code, "failure", message)
}

func validateUploadRequest(r *http.Request, uploadSession, patp string, runtime ...importerRuntime) (validatedUploadRequest, error) {
	runtimeConfig := resolveImporterRuntime(runtime...)
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
	authz := runtimeConfig.validateUploadSessionToken(sesh.Token, structs.WsTokenStruct{
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
	resolvedRuntime := resolveImporterRuntime(runtime...)
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
	if err := resolvedRuntime.closeTempFile(tempFile); err != nil {
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

func runImportPhases(filename, patp, customDrive string, phases workflowOrchestration.WorkflowPhases) error {
	workflowErr := workflowOrchestration.RunStructuredWorkflow(
		phases,
		workflowOrchestration.WorkflowCallbacks{
			Emit: func(phase lifecycle.Phase) {
				if phase == "" {
					return
				}
				publishImportStatus(string(phase))
			},
			OnError: func(err error) {
				publishImportError(err.Error())
			},
		},
	)
	if workflowErr == nil {
		return nil
	}
	if cleanupErr := errorCleanup(filename, patp, customDrive, workflowErr); cleanupErr != nil {
		return fmt.Errorf("import failed for %s: %w", patp, errors.Join(workflowErr, cleanupErr))
	}
	return workflowErr
}

func runImportPhasesWithSteps(filename, patp, customDrive string, steps ...lifecycle.Step) error {
	return runImportPhases(
		filename,
		patp,
		customDrive,
		workflowOrchestration.WorkflowPhases{
			Execute: steps,
		},
	)
}

func HTTPUploadHandler(w http.ResponseWriter, r *http.Request, uploadSession, patp string) {
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

	validated, err := validateUploadRequest(r, uploadSession, patp)
	if err != nil {
		if err := failUploadRequest(w, http.StatusUnauthorized, err.Error()); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}

	chunk, err := readUploadChunk(r)
	if err != nil {
		if err := failUploadRequest(w, http.StatusBadRequest, err.Error()); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}
	defer chunk.File.Close()

	pipelineResult := runImportUploadPipeline(validated, chunk)
	if pipelineResult.err != nil {
		if err := failUploadRequest(w, pipelineResult.statusCode, pipelineResult.err.Error()); err != nil {
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

func runImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload) uploadPipelineResult {
	pipeline := newImportUploadPipeline(validated, chunk)
	return pipeline.run()
}

func newImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload) *importUploadPipeline {
	return &importUploadPipeline{
		validated: validated,
		chunk:     chunk,
	}
}

func (pipeline *importUploadPipeline) run() uploadPipelineResult {
	logger.Debug(fmt.Sprintf("Upload session information for %v: %+v", pipeline.validated.SessionID, pipeline.session))
	publishUploadStatus(pipeline.validated.Patp)
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
					Run:   pipeline.setUploadFinalizationState,
				},
			},
		},
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

func (pipeline *importUploadPipeline) setUploadFinalizationState() error {
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

func publishUploadStatus(patp string) {
	publishImportTransition(structs.UploadTransition{Type: "patp", Event: patp})
	publishImportStatus("uploading")
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
		file.Close()
		return uploadChunkPayload{}, fmt.Errorf("parse upload chunk metadata for %s: %w", fileHeader.Filename, err)
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
	resolvedRuntime := resolveImporterRuntime(runtime...)
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
	if err := resolvedRuntime.dispatchUploadImport(validated.Context, dispatchCmd); err != nil {
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
	resolvedRuntime := resolveImporterRuntime(runtime...)
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
	resolvedRuntime := resolveImporterRuntime(runtime...)
	_, err := resolvedRuntime.stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path %s: %w", path, err)
	}
	return true, nil
}

func combineChunks(filename string, total int) error {
	destFilePath := filepath.Join(uploadDir, filename)
	destFile, err := os.Create(destFilePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

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
		_, err = io.CopyBuffer(destFile, partFile, buffer)
		partFile.Close() // Close immediately after copying

		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
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
	defer resolvedRuntime.cleanupMultipart(tempDir)
	pierCtx, err := newImportedPierContext(ctx, resolvedRuntime, cmd)
	if err != nil {
		return fmt.Errorf("build imported pier context: %w", err)
	}
	if err := resolvedRuntime.runImportedPierWorkflow(pierCtx); err != nil {
		return fmt.Errorf("run imported pier workflow for %s: %w", pierCtx.Patp, err)
	}
	if err := resolvedRuntime.runImportedPostImportWorkflow(pierCtx); err != nil {
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
	if _, err := runtime.storagePathFor("uploads"); err != nil {
		return importedPierContext{}, fmt.Errorf("resolve upload storage path for %s: %w", cmd.Patp, err)
	}
	if _, err := runtime.storagePathFor("temp"); err != nil {
		return importedPierContext{}, fmt.Errorf("resolve temp storage path for %s: %w", cmd.Patp, err)
	}
	volumePath, err := runtime.resolveImportedPierVolumePath(requestCtx, cmd.Patp)
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
	return fmt.Errorf("run imported pier workflow for %s: %w", ctx.Patp, resolvedRuntime.runImportedPierWorkflow(ctx))
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
						return extractUploadedPier(ctx)
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
	)
}

func runImportedPierPostImportWorkflow(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime, err := resolveImporterRuntimeForWorkflow(runtime...)
	if err != nil {
		return fmt.Errorf("prepare importer workflow for %s: %w", ctx.Patp, err)
	}
	if err := resolvedRuntime.runImportedPostImportWorkflow(ctx); err != nil {
		logger.Error(fmt.Sprintf("Imported pier post-processing failed for %s: %v", ctx.Patp, err))
		return fmt.Errorf("import pier post-processing for %s: %w", ctx.Patp, err)
	}
	return nil
}

func runImportedPierPostImportWorkflowDefault(ctx importedPierContext, runtime importerRuntime) error {
	return runImportedPierPostProcessWorkflow(ctx, runtime)
}

func runImportedPierPostProcessWorkflow(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
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
	)
}

func startImportedPierRegistration(patp string, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
	startramSettings := resolvedRuntime.startramSettings()
	if startramSettings.WgRegistered {
		return registerServices(patp)
	}
	return nil
}

func finalizeImportedPierReadiness(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
	logger.Info(fmt.Sprintf("Booting ship: %v", ctx.Patp))
	resolvedRuntime.waitForBootCode(ctx.Patp, 1*time.Second)
	if ctx.Fix {
		if err := resolvedRuntime.applyAcmeFix(ctx.Patp); err != nil {
			wrappedErr := fmt.Errorf("failed to apply ACME fix for imported ship %s: %w", ctx.Patp, err)
			logger.Error(wrappedErr.Error())
			return wrappedErr
		}
	}
	startramSettings := resolvedRuntime.startramSettings()
	if startramSettings.WgRegistered && startramSettings.WgOn && ctx.Remote {
		publishImportStatus("remote")
		resolvedRuntime.waitForRemoteReady(ctx.Patp, 1*time.Second)
		if err := resolvedRuntime.switchToWireguard(ctx.Patp, true); err != nil {
			wrappedErr := fmt.Errorf("failed to switch imported ship %s to Wireguard: %w", ctx.Patp, err)
			logger.Error(wrappedErr.Error())
			return wrappedErr
		}
	}
	publishImportStatus("completed")
	return nil
}

func prepareImportedPierEnvironment(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
	if err := resolvedRuntime.createUrbitConfig(ctx.Patp, ctx.CustomDrive); err != nil {
		return fmt.Errorf("failed to create urbit config: %w", err)
	}
	if err := resolvedRuntime.appendUrbitSysConfig(ctx.Patp); err != nil {
		return fmt.Errorf("failed to update system.json: %w", err)
	}
	logger.Info(fmt.Sprintf("Preparing environment for pier: %v", ctx.Patp))
	if err := resolvedRuntime.deleteContainer(ctx.Patp); err != nil {
		if !isIgnorableCleanupDeleteError(err) {
			return fmt.Errorf("failed to clean up pre-existing container %s: %w", ctx.Patp, err)
		}
		logger.Info(fmt.Sprintf("ignoring pre-existing container cleanup error for %s: %v", ctx.Patp, err))
	}
	if err := resolvedRuntime.deleteVolume(ctx.Patp); err != nil {
		if !isIgnorableCleanupDeleteError(err) {
			return fmt.Errorf("failed to clean up pre-existing volume %s: %w", ctx.Patp, err)
		}
		logger.Info(fmt.Sprintf("ignoring pre-existing volume cleanup error for %s: %v", ctx.Patp, err))
	}
	if ctx.CustomDrive == "" {
		if err := resolvedRuntime.createVolume(ctx.Patp); err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
		return nil
	}
	if err := resolvedRuntime.mkdir(ctx.CustomDrive, os.ModePerm); err != nil {
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

func extractUploadedPier(ctx importedPierContext) error {
	extractionDone := make(chan struct{})
	defer close(extractionDone)
	go monitorExtractionProgress(extractionDone)

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

func monitorExtractionProgress(done <-chan struct{}) {
	extractionTimeout := time.NewTimer(4 * time.Hour)
	select {
	case <-extractionTimeout.C:
		publishImportTransition(structs.UploadTransition{Type: "extracted", Value: 100})
		publishImportStatus("checking")
	case <-done:
		extractionTimeout.Stop()
	}
}

func bootImportedPier(ctx importedPierContext, runtime ...importerRuntime) error {
	resolvedRuntime := resolveImporterRuntime(runtime...)
	logger.Info(fmt.Sprintf("Starting extracted pier: %v", ctx.Patp))
	info, err := resolvedRuntime.startContainer(ctx.Patp, "vere")
	if err != nil {
		return fmt.Errorf("failed to start imported ship: %w", err)
	}
	resolvedRuntime.updateContainer(ctx.Patp, info)
	if err := os.Remove(ctx.ArchivePath); err != nil {
		logger.Warn(fmt.Sprintf("failed to remove uploaded archive %s: %v", ctx.Filename, err))
	}
	return nil
}

func errorCleanup(filename, patp, customDrive string, err error) error {
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
	publishImportError(err.Error())
	publishImportStatus("aborted")
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
