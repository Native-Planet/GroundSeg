package uploadsvc

import (
	"fmt"

	"groundseg/protocol/actions"
)

// Action represents the transport contract for upload websocket commands.
// The canonical action set is `actionBindings`; keep it as the source of truth for
// supported actions and dispatch behavior.
type OpenEndpointRequest struct {
	Endpoint      string
	TokenID       string
	TokenValue    string
	Remote        bool
	Fix           bool
	SelectedDrive string
}

type Action = actions.Action

const (
	ActionOpenEndpoint Action = actions.ActionUploadOpenEndpoint
	ActionReset        Action = actions.ActionUploadReset
)

type Service interface {
	OpenEndpoint(req OpenEndpointRequest) error
	Reset() error
}

type Command struct {
	Action              Action
	OpenEndpointRequest OpenEndpointRequest
}

type UnsupportedActionError = actions.UnsupportedActionError

type Executor struct {
	dispatcher actions.ActionDispatcher[Action, Service, Command]
	service    Service
}

var actionBindings = actions.NewActionDispatcher(actions.NamespaceUpload, []actions.ActionBinding[Action, Service, Command]{
	{
		Action: ActionOpenEndpoint,
		Execute: func(service Service, cmd Command) error {
			return service.OpenEndpoint(cmd.OpenEndpointRequest)
		},
		Operation: func(cmd Command) string {
			return fmt.Sprintf("open upload endpoint %s", cmd.OpenEndpointRequest.Endpoint)
		},
	},
	{
		Action: ActionReset,
		Execute: func(service Service, cmd Command) error {
			return service.Reset()
		},
		Operation: func(Command) string {
			return "reset upload session"
		},
	},
})

// ParseAction validates the action before commands are built.
func ParseAction(raw string) (Action, error) {
	return actions.ParseUploadAction(raw)
}

func SupportedActions() []Action {
	return actions.SupportedUploadActions()
}

func NewExecutor(service Service) (Executor, error) {
	if service == nil {
		return Executor{}, fmt.Errorf("upload service is required")
	}
	return Executor{
		dispatcher: actionBindings,
		service:    service,
	}, nil
}

func (e Executor) Execute(cmd Command) error {
	return e.dispatcher.Execute(cmd.Action, e.service, cmd)
}

func (e Executor) SupportedActions() []Action {
	return e.dispatcher.Supported()
}

func DescribeAction(cmd Command) (string, bool) {
	return actionBindings.Describe(cmd.Action, cmd)
}
