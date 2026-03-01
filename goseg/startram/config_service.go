package startram

import (
	"groundseg/config"
	"groundseg/structs"
)

// ConfigService isolates startram from package-level config globals.
type ConfigService interface {
	StartramSettingsSnapshot() config.StartramSettings
	IsWgRegistered() bool
	SetWgRegistered(registered bool) error
	SetStartramConfig(retrieve structs.StartramRetrieve)
	BasePath() string
}

type configService struct{}

var defaultConfigService ConfigService = configService{}

func SetConfigService(service ConfigService) {
	if service != nil {
		defaultConfigService = service
	}
}

func (configService) StartramSettingsSnapshot() config.StartramSettings {
	return config.StartramSettingsSnapshot()
}

func (configService) IsWgRegistered() bool {
	return config.Conf().WgRegistered
}

func (configService) SetWgRegistered(registered bool) error {
	return config.UpdateConfTyped(config.WithWgRegistered(registered))
}

func (configService) SetStartramConfig(retrieve structs.StartramRetrieve) {
	config.SetStartramConfig(retrieve)
}

func (configService) BasePath() string {
	return config.BasePath
}
