package startram

// RegistrationApplicationService orchestrates registration domain flows.
type RegistrationApplicationService interface {
	Register(regCode, region string) error
	CreateService(subdomain, svcType string) error
	DeleteService(subdomain, svcType string) error
	CreateAlias(subdomain, alias string) error
	DeleteAlias(subdomain, alias string) error
	RegisterExistingShips() error
	RegisterNewShip(ship string) error
}

type registrationApplicationService struct{}

var defaultRegistrationService RegistrationApplicationService = registrationApplicationService{}

func SetRegistrationService(service RegistrationApplicationService) {
	if service != nil {
		defaultRegistrationService = service
	}
}

func (registrationApplicationService) Register(regCode, region string) error {
	return register(regCode, region)
}

func (registrationApplicationService) CreateService(subdomain, svcType string) error {
	return svcCreate(subdomain, svcType)
}

func (registrationApplicationService) DeleteService(subdomain, svcType string) error {
	return svcDelete(subdomain, svcType)
}

func (registrationApplicationService) CreateAlias(subdomain, alias string) error {
	return aliasCreate(subdomain, alias)
}

func (registrationApplicationService) DeleteAlias(subdomain, alias string) error {
	return aliasDelete(subdomain, alias)
}

func (registrationApplicationService) RegisterExistingShips() error {
	return registerExistingShips()
}

func (registrationApplicationService) RegisterNewShip(ship string) error {
	return registerNewShip(ship)
}

func Register(regCode, region string) error {
	return defaultRegistrationService.Register(regCode, region)
}

func SvcCreate(subdomain, svcType string) error {
	return defaultRegistrationService.CreateService(subdomain, svcType)
}

func SvcDelete(subdomain, svcType string) error {
	return defaultRegistrationService.DeleteService(subdomain, svcType)
}

func AliasCreate(subdomain, alias string) error {
	return defaultRegistrationService.CreateAlias(subdomain, alias)
}

func AliasDelete(subdomain, alias string) error {
	return defaultRegistrationService.DeleteAlias(subdomain, alias)
}

func RegisterExistingShips() error {
	return defaultRegistrationService.RegisterExistingShips()
}

func RegisterNewShip(ship string) error {
	return defaultRegistrationService.RegisterNewShip(ship)
}
