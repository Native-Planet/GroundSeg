package uploadsvc

import "groundseg/protocol/actions"

// Action is the upload transport action contract token used by this package.
// It intentionally aliases the protocol action token contract to keep uploadsvc aligned
// with protocol-defined action values and metadata.
type Action = actions.Action

// UploadPayload mirrors protocol upload payload requirements at the uploadsvc boundary.
type UploadPayload = actions.UploadPayload

// UploadActionContract mirrors upload action contract metadata with ownership at this seam.
type UploadActionContract = actions.UploadActionContract

var (
	// ActionUploadOpenEndpoint opens an upload session and provides upload endpoint metadata.
	ActionUploadOpenEndpoint = actions.ActionUploadOpenEndpoint
	// ActionUploadReset resets the current upload session state.
	ActionUploadReset = actions.ActionUploadReset
)

const (
	UploadPayloadOpenEndpoint = actions.UploadPayloadOpenEndpoint
	UploadPayloadReset        = actions.UploadPayloadReset
)

// ParseUploadAction validates an upload action token using protocol contract parsing.
func ParseUploadAction(raw string) (Action, error) {
	return actions.ParseAction(actions.NamespaceUpload, raw)
}

// SupportedUploadActions returns upload actions defined in protocol contracts.
func SupportedUploadActions() ([]Action, error) {
	return actions.SupportedActions(actions.NamespaceUpload)
}

// UploadActionContractByAction returns upload contracts keyed by action.
//
// This is the uploadsvc-owned contract catalog used by adapters and tests.
func UploadActionContractByAction() (map[Action]UploadActionContract, error) {
	return actions.UploadActionContractByAction()
}
