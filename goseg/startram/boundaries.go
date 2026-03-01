package startram

import (
	"encoding/json"
	"fmt"
	"groundseg/structs"

	"go.uber.org/zap"
)

// RetrieveStateSyncer isolates state/config side effects from transport retrieval calls.
type RetrieveStateSyncer interface {
	ApplyRetrieveState(retrieve structs.StartramRetrieve) error
}

type retrieveStateSyncer struct{}

var defaultRetrieveStateSyncer RetrieveStateSyncer = retrieveStateSyncer{}

// EventPublisher decouples event emission from domain flows.
type EventPublisher interface {
	Publish(event structs.Event)
}

type channelEventPublisher struct{}

var defaultEventPublisher EventPublisher = channelEventPublisher{}

func SetRetrieveStateSyncer(syncer RetrieveStateSyncer) {
	if syncer != nil {
		defaultRetrieveStateSyncer = syncer
	}
}

func SetEventPublisher(publisher EventPublisher) {
	if publisher != nil {
		defaultEventPublisher = publisher
	}
}

func ApplyRetrieveState(retrieve structs.StartramRetrieve) error {
	return defaultRetrieveStateSyncer.ApplyRetrieveState(retrieve)
}

func (channelEventPublisher) Publish(event structs.Event) {
	eventBus <- event
}

func PublishEvent(event structs.Event) {
	publishEvent(event)
}

func Events() <-chan structs.Event {
	return eventBus
}

func publishEvent(event structs.Event) {
	defaultEventPublisher.Publish(event)
}

func (retrieveStateSyncer) ApplyRetrieveState(retrieve structs.StartramRetrieve) error {
	defaultConfigService.SetStartramConfig(retrieve)
	zap.L().Info("StarTram info retrieved")
	if serialized, marshalErr := json.Marshal(retrieve); marshalErr == nil {
		zap.L().Debug(fmt.Sprintf("StarTram info: %s", string(serialized)))
	}

	if !defaultConfigService.IsWgRegistered() {
		zap.L().Info("Updating registration status")
		if updateErr := defaultConfigService.SetWgRegistered(true); updateErr != nil {
			zap.L().Error(fmt.Sprintf("%v", updateErr))
		}
	}

	publishEvent(structs.Event{Type: "retrieve", Data: nil})
	return nil
}

// RestoreWorker isolates restore orchestration from public API entrypoints.
type RestoreWorker interface {
	Restore(req RestoreBackupRequest) error
}

type restoreWorker struct{}

var defaultRestoreWorker RestoreWorker = restoreWorker{}

var (
	restoreBackupDevForWorker  = restoreBackupDev
	restoreBackupProdForWorker = restoreBackupProd
)

func SetRestoreWorker(worker RestoreWorker) {
	if worker != nil {
		defaultRestoreWorker = worker
	}
}

func (restoreWorker) Restore(req RestoreBackupRequest) error {
	if req.Ship == "" {
		return fmt.Errorf("ship is required")
	}
	switch req.Mode {
	case RestoreBackupModeDevelopment:
		return restoreBackupDevForWorker(req.Ship)
	case RestoreBackupModeProduction:
		return restoreBackupProdForWorker(req)
	default:
		return fmt.Errorf("unsupported restore mode: %s", req.Mode)
	}
}
