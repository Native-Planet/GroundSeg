package uploadsvc

import (
	"fmt"
)

type OpenEndpointRequest struct {
	Endpoint      string
	TokenID       string
	TokenValue    string
	Remote        bool
	Fix           bool
	SelectedDrive string
}

type Action string

const (
	ActionOpenEndpoint Action = "open-endpoint"
	ActionReset        Action = "reset"
)

type Service interface {
	OpenEndpoint(req OpenEndpointRequest) error
	Reset() error
}

type Command struct {
	Action              Action
	OpenEndpointRequest OpenEndpointRequest
}

type UnsupportedActionError struct {
	Action Action
}

func (e UnsupportedActionError) Error() string {
	return fmt.Sprintf("unsupported upload action: %s", e.Action)
}

type Executor struct {
	dispatch map[Action]func(Command) error
}

var supportedActions = []Action{
	ActionOpenEndpoint,
	ActionReset,
}

func SupportedActions() []Action {
	return append([]Action(nil), supportedActions...)
}

func NewExecutor(service Service) (Executor, error) {
	if service == nil {
		return Executor{}, fmt.Errorf("upload service is required")
	}
	dispatch := map[Action]func(Command) error{
		ActionOpenEndpoint: func(cmd Command) error {
			return service.OpenEndpoint(cmd.OpenEndpointRequest)
		},
		ActionReset: func(cmd Command) error {
			return service.Reset()
		},
	}
	return Executor{dispatch: dispatch}, nil
}

func (e Executor) Execute(cmd Command) error {
	handler, exists := e.dispatch[cmd.Action]
	if !exists {
		return UnsupportedActionError{Action: cmd.Action}
	}
	return handler(cmd)
}

func (e Executor) SupportedActions() []Action {
	actions := make([]Action, 0, len(supportedActions))
	for _, action := range supportedActions {
		if _, ok := e.dispatch[action]; ok {
			actions = append(actions, action)
		}
	}
	return actions
}
