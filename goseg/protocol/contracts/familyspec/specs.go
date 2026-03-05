package familyspec

import "fmt"

// ActionSpec describes a protocol action contract declaration contributed by a family package.
type ActionSpec struct {
	Namespace   string
	Action      string
	ContractID  string
	Name        string
	Description string
	Owner       string
}

// ContractSpec describes a non-action contract declaration contributed by a family package.
type ContractSpec struct {
	ID          string
	Name        string
	Description string
	Message     string
	Owner       string
}

// BuildActionContractID composes canonical action contract IDs.
func BuildActionContractID(root, namespace, action string) string {
	return fmt.Sprintf("%s.%s.%s", root, namespace, action)
}
