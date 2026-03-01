package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/errpolicy"
	"groundseg/structs"
	"groundseg/uploadsvc"
	"groundseg/uploadsvc/adapters"

	"go.uber.org/zap"
)

type UploadMessageHandler struct {
	executor uploadsvc.Executor
}

func UploadSupportedActions() []string {
	actions := uploadsvc.SupportedActions()
	result := make([]string, 0, len(actions))
	for _, action := range actions {
		result = append(result, string(action))
	}
	return result
}

func NewUploadMessageHandler(service uploadsvc.Service) (UploadMessageHandler, error) {
	if service == nil {
		return UploadMessageHandler{}, fmt.Errorf("upload service is required")
	}
	executor, err := uploadsvc.NewExecutor(service)
	if err != nil {
		return UploadMessageHandler{}, err
	}
	return UploadMessageHandler{executor: executor}, nil
}

func (handler UploadMessageHandler) Handle(msg []byte) error {
	zap.L().Info("Upload")
	var uploadPayload structs.WsUploadPayload
	err := json.Unmarshal(msg, &uploadPayload)
	if err != nil {
		return errpolicy.WrapOperation("Couldn't unmarshal upload payload", err)
	}
	command, err := adapters.CommandFromWsPayload(uploadPayload)
	if err != nil {
		var unsupported uploadsvc.UnsupportedActionError
		if errors.As(err, &unsupported) {
			// Preserve external contract for unknown upload actions.
			return fmt.Errorf("Unrecognized upload action: %v", uploadPayload.Payload.Action)
		}
		return errpolicy.WrapOperation("Unsupported upload action", err)
	}
	if err := handler.executor.Execute(command); err != nil {
		var unsupported uploadsvc.UnsupportedActionError
		if errors.As(err, &unsupported) {
			// Preserve external contract for unknown upload actions.
			return fmt.Errorf("Unrecognized upload action: %v", uploadPayload.Payload.Action)
		}
		operation, _ := uploadsvc.DescribeAction(command)
		return errpolicy.WrapOperation(operation, err)
	}
	return nil
}
