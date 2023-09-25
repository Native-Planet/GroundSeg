package rectify

import (
	"fmt"
	"goseg/broadcast"
	"goseg/docker"
	"goseg/logger"
)

func ImportShipTransitionHandler() {
	for {
		event := <-docker.ImportShipTransBus
		current := broadcast.GetState()
		uploadStruct := current.Upload
		switch event.Type {
		// uploading
		// creating
		// extracting
		// booting
		// remote (maybe not needed)
		// completed
		// aborted
		case "status":
			uploadStruct.Status = event.Event
		case "patp":
			uploadStruct.Patp = event.Event
		case "error":
			uploadStruct.Error = event.Event
		case "extracted":
			uploadStruct.Extracted = int64(event.Value)
		default:
			logger.Logger.Warn(fmt.Sprintf("Urecognized transition: %v", event.Type))
			continue
		}
		current.Upload = uploadStruct
		broadcast.UpdateBroadcast(current)
		broadcast.BroadcastToClients()
	}
}
