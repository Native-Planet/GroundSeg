package startram

type FetchRegionsCommand struct {
	EndpointURL string
}

type RetrieveCommand struct {
	EndpointURL string
	Pubkey      string
}

type RegisterCommand struct {
	EndpointURL string
	Pubkey      string
	RegCode     string
	Region      string
}

type ServiceCommand struct {
	EndpointURL string
	Pubkey      string
	Subdomain   string
	SvcType     string
}

type AliasCommand struct {
	EndpointURL string
	Pubkey      string
	Subdomain   string
	Alias       string
}

type CancelSubscriptionCommand struct {
	EndpointURL string
	RegKey      string
}

func buildFetchRegionsCommand() FetchRegionsCommand {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return FetchRegionsCommand{
		EndpointURL: settings.EndpointURL,
	}
}

func buildRetrieveCommand() RetrieveCommand {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return RetrieveCommand{
		EndpointURL: settings.EndpointURL,
		Pubkey:      settings.Pubkey,
	}
}

func buildRegisterCommand(regCode, region string) RegisterCommand {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return RegisterCommand{
		EndpointURL: settings.EndpointURL,
		Pubkey:      settings.Pubkey,
		RegCode:     regCode,
		Region:      region,
	}
}

func buildServiceCommand(subdomain, svcType string) ServiceCommand {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return ServiceCommand{
		EndpointURL: settings.EndpointURL,
		Pubkey:      settings.Pubkey,
		Subdomain:   subdomain,
		SvcType:     svcType,
	}
}

func buildAliasCommand(subdomain, alias string) AliasCommand {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return AliasCommand{
		EndpointURL: settings.EndpointURL,
		Pubkey:      settings.Pubkey,
		Subdomain:   subdomain,
		Alias:       alias,
	}
}

func buildCancelSubscriptionCommand(regKey string) CancelSubscriptionCommand {
	settings := defaultConfigService.StartramSettingsSnapshot()
	return CancelSubscriptionCommand{
		EndpointURL: settings.EndpointURL,
		RegKey:      regKey,
	}
}
