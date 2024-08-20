package routines

import (
	"time"

	"go.uber.org/zap"
)

func TlonBackups() {
	for {
		// check backups
		zap.L().Debug("fake tlon backup check")
		time.Sleep(1 * time.Minute)
	}
}
