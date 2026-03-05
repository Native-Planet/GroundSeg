package shipworkflow

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

func HandleStartramSetBackupPassword(password string) error {
	return runStartramSetBackupPasswordWithRuntime(defaultStartramRuntime(), password)
}

func HandleStartramUploadBackup(patp string) error {
	return runStartramUploadBackupWithRuntime(defaultStartramRuntime(), patp)
}

func runStartramUploadBackupWithRuntime(runtime startramRuntime, patp string) error {
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionUploadBackup,
		startEvent:     startramEvent(transition.StartramTransitionUploadBackup, "upload"),
		clearEvent:     startramEvent(transition.StartramTransitionUploadBackup, nil),
		clearDelay:     3 * time.Second,
	},
		transitionStep[structs.Event]{
			Run: func() error {
				filePath := "backup.key"
				keyBytes, err := os.ReadFile(filePath)
				if err != nil {
					zap.L().Error(fmt.Sprintf("failed to read private key file: %v", err))
					return fmt.Errorf("failed to read private key file: %w", err)
				}
				decodedKeyBytes, err := base64.StdEncoding.DecodeString(string(keyBytes))
				if err != nil {
					zap.L().Error(fmt.Sprintf("failed to decode private key file: %v", err))
					return fmt.Errorf("failed to decode private key file: %w", err)
				}
				pk := strings.TrimSpace(string(decodedKeyBytes))
				if err := startram.UploadBackup(patp, pk, filePath); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to upload backup: %v", err))
					return fmt.Errorf("upload backup failed: %w", err)
				}
				return nil
			},
		},
	)
}
