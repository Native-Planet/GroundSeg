package transition

import "strings"

// Cross-module transition and boundary action contracts.

// EventType identifies the domain event producer/consumer channel kind.
type EventType string
type ContainerStatus string
type ContainerType string
type BroadcastMessageType string
type BroadcastAuthLevel string
type DockerAction string
type StartramTransitionStatus string
type StartramServiceType string
type StartramServiceStatus string

// UrbitTransitionType names all persisted urbit transition streams.
type UrbitTransitionType string

// NewShipTransitionType names new-ship workflow transition fields.
type NewShipTransitionType string

// SystemTransitionType names system-level transitions.
type SystemTransitionType string

// UploadTransitionType names upload-related transitions.
type UploadTransitionType string

const (
	DockerActionStop  DockerAction = "stop"
	DockerActionStart DockerAction = "start"
	DockerActionDie   DockerAction = "die"
)

const (
	ContainerTypeWireguard ContainerType = "wireguard"
	ContainerTypeVere      ContainerType = "vere"
)

const (
	ContainerStatusRunning  ContainerStatus = "running"
	ContainerStatusStopped  ContainerStatus = "stopped"
	ContainerStatusDied     ContainerStatus = "died"
	ContainerStatusUpPrefix                 = "Up"
)

const (
	UrbitTransitionRollChop                  UrbitTransitionType = "rollChop"
	UrbitTransitionChopOnUpgrade             UrbitTransitionType = "chopOnUpgrade"
	UrbitTransitionChop                      UrbitTransitionType = "chop"
	UrbitTransitionPack                      UrbitTransitionType = "pack"
	UrbitTransitionPackMeld                  UrbitTransitionType = "packMeld"
	UrbitTransitionLoom                      UrbitTransitionType = "loom"
	UrbitTransitionSnapTime                  UrbitTransitionType = "snapTime"
	UrbitTransitionUrbitDomain               UrbitTransitionType = "urbitDomain"
	UrbitTransitionMinIODomain               UrbitTransitionType = "minioDomain"
	UrbitTransitionRebuildContainer          UrbitTransitionType = "rebuildContainer"
	UrbitTransitionToggleDevMode             UrbitTransitionType = "toggleDevMode"
	UrbitTransitionTogglePower               UrbitTransitionType = "togglePower"
	UrbitTransitionToggleNetwork             UrbitTransitionType = "toggleNetwork"
	UrbitTransitionExportShip                UrbitTransitionType = "exportShip"
	UrbitTransitionShipCompressed            UrbitTransitionType = "shipCompressed"
	UrbitTransitionExportBucket              UrbitTransitionType = "exportBucket"
	UrbitTransitionBucketCompressed          UrbitTransitionType = "bucketCompressed"
	UrbitTransitionDeleteShip                UrbitTransitionType = "deleteShip"
	UrbitTransitionToggleMinIOLink           UrbitTransitionType = "toggleMinIOLink"
	UrbitTransitionPenpaiCompanion           UrbitTransitionType = "penpaiCompanion"
	UrbitTransitionGallseg                   UrbitTransitionType = "gallseg"
	UrbitTransitionDeleteService             UrbitTransitionType = "deleteService"
	UrbitTransitionLocalTlonBackupsEnabled   UrbitTransitionType = "localTlonBackupsEnabled"
	UrbitTransitionRemoteTlonBackupsEnabled  UrbitTransitionType = "remoteTlonBackupsEnabled"
	UrbitTransitionLocalTlonBackup           UrbitTransitionType = "localTlonBackup"
	UrbitTransitionLocalTlonBackupSchedule   UrbitTransitionType = "localTlonBackupSchedule"
	UrbitTransitionHandleRestoreTlonBackup   UrbitTransitionType = "handleRestoreTlonBackup"
	UrbitTransitionServiceRegistrationStatus UrbitTransitionType = "serviceRegistrationStatus"
)

const (
	NewShipTransitionBootStage NewShipTransitionType = "bootStage"
	NewShipTransitionError     NewShipTransitionType = "error"
	NewShipTransitionPatp      NewShipTransitionType = "patp"
	NewShipTransitionFreeError NewShipTransitionType = "freeError"
)

const (
	SystemTransitionBugReport      SystemTransitionType = "bugReport"
	SystemTransitionBugReportError SystemTransitionType = "bugReportError"
	SystemTransitionSwap           SystemTransitionType = "swap"
	SystemTransitionWifiConnect    SystemTransitionType = "wifiConnect"
)

const (
	StartramTransitionEndpoint      EventType = "endpoint"
	StartramTransitionRegister      EventType = "register"
	StartramTransitionRestart       EventType = "restart"
	StartramTransitionToggle        EventType = "toggle"
	StartramTransitionServices      EventType = "services"
	StartramTransitionRegions       EventType = "regions"
	StartramTransitionReminder      EventType = "reminder"
	StartramTransitionSetBackupPW   EventType = "set-backup-password"
	StartramTransitionRestoreBackup EventType = "restoreBackup"
	StartramTransitionBackup        EventType = "backup"
	StartramTransitionUploadBackup  EventType = "uploadBackup"
	StartramTransitionCancel        EventType = "cancel"
	StartramTransitionRetrieve      EventType = "retrieve"
)

const (
	UploadTransitionStatus    UploadTransitionType = "status"
	UploadTransitionPatp      UploadTransitionType = "patp"
	UploadTransitionError     UploadTransitionType = "error"
	UploadTransitionExtracted UploadTransitionType = "extracted"
)

const (
	StartramHandlerActionServices    EventType = "services"
	StartramHandlerActionRegions     EventType = "regions"
	StartramHandlerActionRegister    EventType = "register"
	StartramHandlerActionToggle      EventType = "toggle"
	StartramHandlerActionRestart     EventType = "restart"
	StartramHandlerActionCancel      EventType = "cancel"
	StartramHandlerActionEndpoint    EventType = "endpoint"
	StartramHandlerActionReminder    EventType = "reminder"
	StartramHandlerActionSetBackupPW EventType = "set-backup-password"
)

const (
	StartramTransitionLoading        StartramTransitionStatus = "loading"
	StartramTransitionStarting       StartramTransitionStatus = "starting"
	StartramTransitionStopping       StartramTransitionStatus = "stopping"
	StartramTransitionServicesAction                          = "services"
	StartramTransitionStartingAction                          = "starting"
	StartramTransitionStoppingAction                          = "stopping"
	StartramTransitionUnregistering                           = "unregistering"
	StartramTransitionConfiguring                             = "configuring"
	StartramTransitionFinalizing                              = "finalizing"
	StartramTransitionComplete       StartramTransitionStatus = "complete"
	StartramTransitionDone                                    = "done"
	StartramTransitionInit           StartramTransitionStatus = "init"
	StartramTransitionCreating       StartramTransitionStatus = "creating"
	StartramTransitionOk             StartramTransitionStatus = "ok"
)

const (
	StartramServiceTypeUrbitWeb   StartramServiceType = "urbit-web"
	StartramServiceTypeUrbitAmes  StartramServiceType = "urbit-ames"
	StartramServiceTypeMinio      StartramServiceType = "minio"
	StartramServiceTypeMinioAdmin StartramServiceType = "minio-console"

	StartramServiceStatusCreating StartramServiceStatus = "creating"
	StartramServiceStatusOk       StartramServiceStatus = "ok"
)

const (
	TransitionErrorPrefix                          = "Error: "
	TransitionStatusEmpty StartramTransitionStatus = ""
)

const (
	BroadcastMessageTypeStructure  BroadcastMessageType = "structure"
	BroadcastAuthLevelAuthorized   BroadcastAuthLevel   = "authorized"
	BroadcastAuthLevelSetup        BroadcastAuthLevel   = "setup"
	BroadcastAuthLevelUnauthorized BroadcastAuthLevel   = "unauthorized"
)

const (
	C2CActionConnect = "connect"
)

const (
	SystemActionRestart  = "restart"
	SystemActionPower    = "power"
	SystemActionShutdown = "shutdown"
	SystemActionUpdate   = "update"
)

func ParseDockerAction(raw string) (DockerAction, bool) {
	action := DockerAction(raw)
	switch action {
	case DockerActionStop, DockerActionStart, DockerActionDie:
		return action, true
	default:
		return "", false
	}
}

func ParseStartramHandlerAction(raw string) (EventType, bool) {
	action := EventType(raw)
	switch action {
	case StartramHandlerActionServices,
		StartramHandlerActionRegions,
		StartramHandlerActionRegister,
		StartramHandlerActionToggle,
		StartramHandlerActionRestart,
		StartramHandlerActionCancel,
		StartramHandlerActionEndpoint,
		StartramHandlerActionReminder,
		StartramHandlerActionSetBackupPW:
		return action, true
	default:
		return "", false
	}
}

func IsContainerUpStatus(status string) bool {
	return status == string(ContainerStatusUpPrefix) || strings.HasPrefix(status, string(ContainerStatusUpPrefix)+" ")
}
