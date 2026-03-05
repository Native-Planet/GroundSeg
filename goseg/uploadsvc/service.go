package uploadsvc

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"groundseg/protocol/actions"
)

type OpenEndpointRequest struct {
	Endpoint      string
	TokenID       string
	TokenValue    string
	Remote        bool
	Fix           bool
	SelectedDrive string
}

type ResetRequest struct{}

type Service interface {
	OpenEndpoint(req OpenEndpointRequest) error
	Reset() error
}

type Command struct {
	Action              Action
	OpenEndpointRequest *OpenEndpointRequest
	ResetRequest        *ResetRequest
}

type CommandValidationError struct {
	Action  Action
	Problem string
	Cause   error
}

func (e CommandValidationError) Error() string {
	if e.Problem == "" && e.Cause != nil {
		return e.Cause.Error()
	}
	return e.Problem
}

func (e CommandValidationError) Unwrap() error {
	return e.Cause
}

func (e CommandValidationError) Is(target error) bool {
	return target == ErrCommandValidation
}

var (
	ErrCommandValidation             = errors.New("upload command validation failed")
	ErrUploadContractUnavailable     = errors.New("upload action contract metadata unavailable")
	ErrOpenEndpointRequestMissing    = errors.New("open-endpoint request is required")
	ErrResetRequestMissing           = errors.New("reset request is required")
	ErrOpenEndpointEndpointMissing   = errors.New("open-endpoint endpoint is required")
	ErrOpenEndpointTokenIDMissing    = errors.New("open-endpoint token id is required")
	ErrOpenEndpointTokenValueMissing = errors.New("open-endpoint token value is required")
	ErrOpenEndpointPayloadMix        = errors.New("open-endpoint command must not include reset payload")
	ErrResetPayloadMix               = errors.New("reset command must not include open-endpoint payload")
	ErrUploadDispatch                = errors.New("upload command dispatch failed")
)

type DispatchError struct {
	Action Action
	Err    error
}

func (e DispatchError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", ErrUploadDispatch, e.Action)
	}
	return fmt.Sprintf("%s (%s): %v", ErrUploadDispatch, e.Action, e.Err)
}

func (e DispatchError) Unwrap() error {
	return e.Err
}

func (e DispatchError) Is(target error) bool {
	return target == ErrUploadDispatch
}

type Executor struct {
	dispatcher actions.ActionDispatcher[actions.Action, Service, Command]
	service    Service
}

type uploadActionDescriptor struct {
	action    Action
	execute   func(Service, Command) error
	operation func(Command) string
}

var uploadActionDescriptors = []uploadActionDescriptor{
	{
		action: ActionUploadOpenEndpoint,
		execute: func(service Service, cmd Command) error {
			return service.OpenEndpoint(*cmd.OpenEndpointRequest)
		},
		operation: func(cmd Command) string {
			if cmd.OpenEndpointRequest == nil {
				return "open upload endpoint"
			}
			return fmt.Sprintf("open upload endpoint %s", cmd.OpenEndpointRequest.Endpoint)
		},
	},
	{
		action: ActionUploadReset,
		execute: func(service Service, cmd Command) error {
			return service.Reset()
		},
	},
}

func CommandFromPayload(action Action, openReq *OpenEndpointRequest, resetReq *ResetRequest) (Command, error) {
	cmd := Command{
		Action:              action,
		OpenEndpointRequest: openReq,
		ResetRequest:        resetReq,
	}
	if err := ValidateCommand(cmd); err != nil {
		return Command{}, err
	}
	return cmd, nil
}

// CommandFromUploadInputs applies upload-specific normalization for ws payload shape and
// routes validation through ValidateCommand.
func CommandFromUploadInputs(action Action, openReq OpenEndpointRequest, resetReq *ResetRequest) (Command, error) {
	contract, err := actionContractForAction(action)
	if err != nil {
		return Command{}, err
	}
	openEndpointRequest := openReqPointerForPayload(openReq, contract)
	return CommandFromPayload(action, openEndpointRequest, resetReq)
}

func hasOpenEndpointFields(req OpenEndpointRequest) bool {
	return req.Endpoint != "" || req.Remote || req.Fix || req.SelectedDrive != ""
}

func openReqPointerForPayload(openReq OpenEndpointRequest, contract uploadActionGovernanceContract) *OpenEndpointRequest {
	if !contract.RequiredPayloads.Has(actions.UploadPayloadOpenEndpoint) && !hasOpenEndpointFields(openReq) {
		return nil
	}
	return &openReq
}

func actionContractForAction(action Action) (uploadActionGovernanceContract, error) {
	contract, err := uploadGovernanceContractForAction(action)
	if err != nil {
		return uploadActionGovernanceContract{}, err
	}
	return contract, nil
}

func ValidateCommand(cmd Command) error {
	contract, err := actionContractForAction(cmd.Action)
	if err != nil {
		var unsupported actions.UnsupportedActionError
		if errors.As(err, &unsupported) {
			return err
		}
		return CommandValidationError{
			Action:  cmd.Action,
			Problem: err.Error(),
			Cause:   errors.Join(ErrCommandValidation, ErrUploadContractUnavailable),
		}
	}

	openPayloadPresent := cmd.OpenEndpointRequest != nil
	resetPayloadPresent := cmd.ResetRequest != nil
	required := contract.RequiredPayloads
	forbidden := contract.ForbiddenPayloads

	if required.Has(actions.UploadPayloadOpenEndpoint) && !openPayloadPresent {
		return CommandValidationError{Action: cmd.Action, Problem: openEndpointMissingError(contract.Action), Cause: errors.Join(ErrCommandValidation, ErrOpenEndpointRequestMissing)}
	}
	if required.Has(actions.UploadPayloadOpenEndpoint) {
		if err := validateRequiredOpenEndpointPayload(cmd.Action, cmd.OpenEndpointRequest); err != nil {
			return err
		}
	}
	if required.Has(actions.UploadPayloadReset) && !resetPayloadPresent {
		return CommandValidationError{Action: cmd.Action, Problem: resetRequestMissingError(contract.Action), Cause: errors.Join(ErrCommandValidation, ErrResetRequestMissing)}
	}

	if forbidden.Has(actions.UploadPayloadOpenEndpoint) && openPayloadPresent {
		return CommandValidationError{
			Action:  cmd.Action,
			Problem: actionUploadPayloadViolationMessage(contract.Action, actions.UploadPayloadOpenEndpoint),
			Cause:   errors.Join(ErrCommandValidation, ErrResetPayloadMix),
		}
	}
	if forbidden.Has(actions.UploadPayloadReset) && resetPayloadPresent {
		return CommandValidationError{
			Action:  cmd.Action,
			Problem: actionUploadPayloadViolationMessage(contract.Action, actions.UploadPayloadReset),
			Cause:   errors.Join(ErrCommandValidation, ErrOpenEndpointPayloadMix),
		}
	}

	if required.Has(actions.UploadPayloadOpenEndpoint) && required.Has(actions.UploadPayloadReset) {
		return CommandValidationError{Action: cmd.Action, Problem: "unsupported action", Cause: ErrCommandValidation}
	}
	return nil
}

func validateRequiredOpenEndpointPayload(action Action, request *OpenEndpointRequest) error {
	if request == nil {
		return nil
	}
	if strings.TrimSpace(request.Endpoint) == "" {
		return CommandValidationError{
			Action:  action,
			Problem: ErrOpenEndpointEndpointMissing.Error(),
			Cause:   errors.Join(ErrCommandValidation, ErrOpenEndpointEndpointMissing),
		}
	}
	if strings.TrimSpace(request.TokenID) == "" {
		return CommandValidationError{
			Action:  action,
			Problem: ErrOpenEndpointTokenIDMissing.Error(),
			Cause:   errors.Join(ErrCommandValidation, ErrOpenEndpointTokenIDMissing),
		}
	}
	if strings.TrimSpace(request.TokenValue) == "" {
		return CommandValidationError{
			Action:  action,
			Problem: ErrOpenEndpointTokenValueMissing.Error(),
			Cause:   errors.Join(ErrCommandValidation, ErrOpenEndpointTokenValueMissing),
		}
	}
	return nil
}

func openEndpointMissingError(action Action) string {
	switch action {
	case ActionUploadOpenEndpoint:
		return ErrOpenEndpointRequestMissing.Error()
	default:
		return fmt.Sprintf("%s action requires an open-endpoint payload", action)
	}
}

func resetRequestMissingError(action Action) string {
	switch action {
	case ActionUploadReset:
		return ErrResetRequestMissing.Error()
	default:
		return fmt.Sprintf("%s action requires a reset payload", action)
	}
}

func actionUploadPayloadViolationMessage(action Action, forbiddenPayload actions.UploadPayload) string {
	switch action {
	case ActionUploadOpenEndpoint:
		if forbiddenPayload == actions.UploadPayloadReset {
			return ErrOpenEndpointPayloadMix.Error()
		}
	case ActionUploadReset:
		if forbiddenPayload == actions.UploadPayloadOpenEndpoint {
			return ErrResetPayloadMix.Error()
		}
	}
	return fmt.Sprintf("%s action payload mix is invalid", action)
}

func newUploadDispatcher() (actions.ActionDispatcher[actions.Action, Service, Command], error) {
	contracts, err := actions.UploadActionContractByAction()
	if err != nil {
		return actions.ActionDispatcher[actions.Action, Service, Command]{}, err
	}
	actionsKeys := make([]actions.Action, 0, len(contracts))
	for action := range contracts {
		actionsKeys = append(actionsKeys, action)
	}
	sort.Slice(actionsKeys, func(i, j int) bool { return string(actionsKeys[i]) < string(actionsKeys[j]) })
	descriptorByAction := make(map[actions.Action]uploadActionDescriptor, len(uploadActionDescriptors))
	for _, descriptor := range uploadActionDescriptors {
		if _, exists := descriptorByAction[descriptor.action]; exists {
			return actions.ActionDispatcher[actions.Action, Service, Command]{}, fmt.Errorf("upload action %q is defined multiple times", descriptor.action)
		}
		descriptorByAction[descriptor.action] = descriptor
	}

	dispatcherBindings := make([]actions.ActionBinding[actions.Action, Service, Command], 0, len(contracts))
	seen := make(map[actions.Action]struct{}, len(contracts))
	for _, action := range actionsKeys {
		contract := contracts[action]
		descriptor, ok := descriptorByAction[contract.Action]
		if !ok {
			return actions.ActionDispatcher[actions.Action, Service, Command]{}, fmt.Errorf("upload action definition missing for %q", contract.Action)
		}
		if _, exists := seen[contract.Action]; exists {
			return actions.ActionDispatcher[actions.Action, Service, Command]{}, fmt.Errorf("upload action %q is defined multiple times", contract.Action)
		}
		seen[contract.Action] = struct{}{}

		operation := descriptor.operation
		if operation == nil {
			description := contract.Description
			operation = func(_ Command) string { return description }
		}
		dispatcherBindings = append(dispatcherBindings, actions.ActionBinding[actions.Action, Service, Command]{
			Action:    contract.Action,
			Execute:   descriptor.execute,
			Operation: operation,
		})
	}
	if len(dispatcherBindings) == 0 {
		return actions.ActionDispatcher[actions.Action, Service, Command]{}, fmt.Errorf("no upload actions configured")
	}
	if len(dispatcherBindings) != len(contracts) {
		return actions.ActionDispatcher[actions.Action, Service, Command]{}, fmt.Errorf("upload action definition corrupted")
	}

	for _, descriptor := range descriptorByAction {
		if _, ok := seen[descriptor.action]; !ok {
			return actions.ActionDispatcher[actions.Action, Service, Command]{}, fmt.Errorf("upload action configured but unsupported: %q", descriptor.action)
		}
	}

	return actions.NewActionDispatcher(actions.NamespaceUpload, dispatcherBindings), nil
}

func NewExecutor(service Service) (Executor, error) {
	if service == nil {
		return Executor{}, fmt.Errorf("upload service is required")
	}
	dispatcher, err := newUploadDispatcher()
	if err != nil {
		return Executor{}, err
	}
	if len(dispatcher.Supported()) == 0 {
		return Executor{}, fmt.Errorf("no upload actions configured")
	}
	return Executor{
		dispatcher: dispatcher,
		service:    service,
	}, nil
}

func (e Executor) Execute(cmd Command) error {
	if err := ValidateCommand(cmd); err != nil {
		return err
	}
	err := e.dispatcher.Execute(cmd.Action, e.service, cmd)
	if err == nil {
		return nil
	}
	var unsupported actions.UnsupportedActionError
	if errors.As(err, &unsupported) {
		return unsupported
	}
	return DispatchError{Action: cmd.Action, Err: err}
}

func (e Executor) SupportedActions() []Action {
	return e.dispatcher.Supported()
}

func DescribeAction(cmd Command) (string, bool) {
	dispatcher, err := newUploadDispatcher()
	if err != nil {
		return "", false
	}
	return dispatcher.Describe(cmd.Action, cmd)
}
