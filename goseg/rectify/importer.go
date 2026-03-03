package rectify

import (
	"context"
	"fmt"
	"groundseg/broadcast"
	"groundseg/docker/events"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

func ImportShipTransitionHandler() {
	if err := ImportShipTransitionHandlerWithContext(context.Background()); err != nil {
		zap.L().Warn(fmt.Sprintf("import ship transition handler stopped with error: %v", err))
	}
}

func ImportShipTransitionHandlerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-events.DefaultEventRuntime().ImportShipTransitions():
			if err := broadcast.ApplyBroadcastMutation(true, func(current *structs.AuthBroadcast) {
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
}
