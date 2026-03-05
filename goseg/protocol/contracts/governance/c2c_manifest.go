package governance

import "groundseg/protocol/contracts/catalog/common"

const (
	ActionC2CConnect = "connect"

	C2CConnectContractID = ProtocolActionContractRoot + "." + NamespaceC2C + "." + ActionC2CConnect

	C2CConnectActionName        = "C2CConnectAction"
	C2CConnectActionDescription = "connect c2c client"
)

func c2cActionDeclarations() []ActionDeclaration {
	return []ActionDeclaration{
		{
			Namespace:         NamespaceC2C,
			Action:            ActionC2CConnect,
			ContractID:        C2CConnectContractID,
			Name:              C2CConnectActionName,
			Description:       C2CConnectActionDescription,
			Owner:             string(common.OwnerSystemWiFi),
			RequiredPayloads:  0,
			ForbiddenPayloads: 0,
		},
	}
}
