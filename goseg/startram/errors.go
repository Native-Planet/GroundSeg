package startram

import (
	"groundseg/errpolicy"
	"groundseg/protocol/contracts"
)

type APIConnectionErrorContractDescriptor = contracts.ContractDescriptor

const APIConnectionErrorMessage = contracts.APIConnectionErrorMessage

var apiConnectionErrorContract = APIConnectionErrorContractDescriptor(
	contracts.MustContractDescriptor(contracts.APIConnectionError),
)

func APIConnectionErrorContract() APIConnectionErrorContractDescriptor {
	return apiConnectionErrorContract
}

type APIConnectionError struct {
	masked error
	cause  error
}

func (e APIConnectionError) Error() string {
	return e.masked.Error()
}

func (e APIConnectionError) Unwrap() error {
	return e.cause
}

func APIConnectionErrorContractForVersion(version string) (APIConnectionErrorContractDescriptor, bool) {
	if version == "" {
		return APIConnectionErrorContract(), false
	}
	contract := APIConnectionErrorContract()
	if !contract.IsActive(version) {
		return contract, false
	}
	return contract, true
}

func IsAPIConnectionErrorContractActive(version string) bool {
	_, ok := APIConnectionErrorContractForVersion(version)
	return ok
}

func IsAPIConnectionErrorContractDeprecated(version string) bool {
	contract := APIConnectionErrorContract()
	return contract.IsDeprecated(version)
}

func wrapAPIConnectionError(err error) error {
	masked := errpolicy.WrapMasked(contracts.APIConnectionErrorMessage, err)
	return APIConnectionError{
		masked: masked,
		cause:  err,
	}
}
