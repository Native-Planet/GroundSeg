package actions

import "fmt"

// ProtocolVersion identifies the action contract expectations used by websocket
// transport features.
const ProtocolVersion = "1.0"

// Namespace groups action enums by transport domain.
type Namespace string

const (
	NamespaceUpload Namespace = "upload"
	NamespaceC2C    Namespace = "c2c"
)

// Action is a transport action contract token.
type Action string

const (
	ActionUploadOpenEndpoint Action = "open-endpoint"
	ActionUploadReset        Action = "reset"
	ActionC2CConnect         Action = "connect"
)

var (
	uploadSupported = []Action{ActionUploadOpenEndpoint, ActionUploadReset}
	c2cSupported    = []Action{ActionC2CConnect}
)

// UnsupportedActionError is raised for unknown action values within a namespace.
type UnsupportedActionError struct {
	Namespace Namespace
	Action    Action
}

func (e UnsupportedActionError) Error() string {
	return fmt.Sprintf("unsupported %s action: %s", e.Namespace, e.Action)
}

// ParseAction validates an action for a given namespace and returns a typed Action.
func ParseAction(namespace Namespace, raw string) (Action, error) {
	action := Action(raw)
	allowed := map[Namespace][]Action{
		NamespaceUpload: uploadSupported,
		NamespaceC2C:    c2cSupported,
	}[namespace]

	for _, supported := range allowed {
		if action == supported {
			return action, nil
		}
	}
	return action, UnsupportedActionError{Namespace: namespace, Action: action}
}

// SupportedActions returns supported action tokens for a namespace.
func SupportedActions(namespace Namespace) []Action {
	var actions []Action
	switch namespace {
	case NamespaceUpload:
		actions = append(actions, uploadSupported...)
	case NamespaceC2C:
		actions = append(actions, c2cSupported...)
	default:
		return nil
	}
	out := make([]Action, len(actions))
	copy(out, actions)
	return out
}

// ParseUploadAction validates actions for the upload transport namespace.
func ParseUploadAction(raw string) (Action, error) {
	return ParseAction(NamespaceUpload, raw)
}

// ParseC2CAction validates actions for the c2c transport namespace.
func ParseC2CAction(raw string) (Action, error) {
	return ParseAction(NamespaceC2C, raw)
}

// SupportedUploadActions returns upload-supported actions.
func SupportedUploadActions() []Action {
	return SupportedActions(NamespaceUpload)
}

// SupportedC2CActions returns c2c-supported actions.
func SupportedC2CActions() []Action {
	return SupportedActions(NamespaceC2C)
}
