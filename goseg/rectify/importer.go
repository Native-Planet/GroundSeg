package rectify

import (
	"fmt"
	"groundseg/broadcast"
	"groundseg/docker"
	"groundseg/transition"
	"groundseg/structs"

	"go.uber.org/zap"
)

func ImportShipTransitionHandler() {
	for {
		event := <-docker.ImportShipTransitions()
		if err := broadcast.ApplyBroadcastUpdate(true, func(current *structs.AuthBroadcast) {
			uploadStruct := current.Upload
			switch transition.UploadTransitionType(event.Type) {
			// uploading
			// creating
			// extracting
			// booting
			// remote (maybe not needed)
			// completed
			// aborted
			case transition.UploadTransitionStatus:
				uploadStruct.Status = event.Event
			case transition.UploadTransitionPatp:
				uploadStruct.Patp = event.Event
			case transition.UploadTransitionError:
				uploadStruct.Error = event.Event
			case transition.UploadTransitionExtracted:
				uploadStruct.Extracted = int64(event.Value)
			default:
				zap.L().Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
				return
			}
			current.Upload = uploadStruct
		}); err != nil {
			zap.L().Warn(fmt.Sprintf("Unable to publish upload transition update: %v", err))
		}
	}
}
