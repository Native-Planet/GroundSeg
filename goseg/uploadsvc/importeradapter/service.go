package importeradapter

import (
	"groundseg/importer"
	"groundseg/structs"
	"groundseg/uploadsvc"
)

type service struct{}

func New() uploadsvc.Service {
	return service{}
}

func (service) OpenEndpoint(req uploadsvc.OpenEndpointRequest) error {
	cmd := importer.OpenUploadEndpointCmd{
		Endpoint: req.Endpoint,
		Token: structs.WsTokenStruct{
			ID:    req.TokenID,
			Token: req.TokenValue,
		},
		Remote:        req.Remote,
		Fix:           req.Fix,
		SelectedDrive: req.SelectedDrive,
	}
	return importer.OpenUploadEndpoint(cmd)
}

func (service) Reset() error {
	return importer.Reset()
}
