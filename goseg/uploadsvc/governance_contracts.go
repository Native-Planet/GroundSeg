package uploadsvc

import (
	"fmt"
	"sync"

	"groundseg/protocol/actions"
	"groundseg/protocol/contracts/governance"
)

type uploadActionGovernanceContract struct {
	Action            Action
	ContractID        string
	RequiredPayloads  actions.UploadPayload
	ForbiddenPayloads actions.UploadPayload
}

var (
	uploadGovernanceContractsByAction     map[Action]uploadActionGovernanceContract
	uploadGovernanceContractsByActionErr  error
	uploadGovernanceContractsByActionInit sync.Once
)

func buildUploadGovernanceContractsByAction() (map[Action]uploadActionGovernanceContract, error) {
	declarations := governance.UploadActionDeclarations()
	if len(declarations) == 0 {
		return nil, fmt.Errorf("governance upload action declarations are empty")
	}

	contracts := make(map[Action]uploadActionGovernanceContract, len(declarations))
	for _, declaration := range declarations {
		if declaration.Namespace != governance.NamespaceUpload {
			return nil, fmt.Errorf("unexpected governance namespace for upload action %s: %s", declaration.Action, declaration.Namespace)
		}
		action := Action(declaration.Action)
		if _, exists := contracts[action]; exists {
			return nil, fmt.Errorf("duplicate governance upload action declaration: %s", declaration.Action)
		}
		contracts[action] = uploadActionGovernanceContract{
			Action:            action,
			ContractID:        declaration.ContractID,
			RequiredPayloads:  actions.UploadPayload(declaration.RequiredPayloads),
			ForbiddenPayloads: actions.UploadPayload(declaration.ForbiddenPayloads),
		}
	}
	return contracts, nil
}

func uploadGovernanceContractForAction(action Action) (uploadActionGovernanceContract, error) {
	uploadGovernanceContractsByActionInit.Do(func() {
		uploadGovernanceContractsByAction, uploadGovernanceContractsByActionErr = buildUploadGovernanceContractsByAction()
	})
	if uploadGovernanceContractsByActionErr != nil {
		return uploadActionGovernanceContract{}, uploadGovernanceContractsByActionErr
	}
	contract, ok := uploadGovernanceContractsByAction[action]
	if !ok {
		return uploadActionGovernanceContract{}, actions.UnsupportedActionError{
			Namespace: actions.NamespaceUpload,
			Action:    actions.Action(action),
		}
	}
	return contract, nil
}
