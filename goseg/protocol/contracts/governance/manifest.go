package governance

const (
	ProtocolActionContractRoot = "protocol.actions"

	NamespaceUpload = "upload"
	NamespaceC2C    = "c2c"
)

type UploadPayloadRule uint8

const (
	UploadPayloadRuleOpenEndpoint UploadPayloadRule = 1 << iota
	UploadPayloadRuleReset
)

func (rule UploadPayloadRule) Has(flag UploadPayloadRule) bool {
	return rule&flag != 0
}

type ActionDeclaration struct {
	Namespace         string
	Action            string
	ContractID        string
	Name              string
	Description       string
	Owner             string
	RequiredPayloads  UploadPayloadRule
	ForbiddenPayloads UploadPayloadRule
}

type ContractDeclaration struct {
	ID          string
	Name        string
	Description string
	Message     string
	Owner       string
}

func ActionDeclarations() []ActionDeclaration {
	upload := uploadActionDeclarations()
	c2c := c2cActionDeclarations()
	out := make([]ActionDeclaration, 0, len(upload)+len(c2c))
	out = append(out, upload...)
	out = append(out, c2c...)
	return out
}

func UploadActionDeclarations() []ActionDeclaration {
	return uploadActionDeclarations()
}

func C2CActionDeclarations() []ActionDeclaration {
	return c2cActionDeclarations()
}

func StartramContractDeclarations() []ContractDeclaration {
	declarations := startramContractDeclarations()
	out := make([]ContractDeclaration, len(declarations))
	copy(out, declarations)
	return out
}
