package shipcleanup

import (
	"errors"
	"fmt"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/shipcreator"
	"os"
)

type RollbackOptions struct {
	UploadArchivePath    string
	CustomPierPath       string
	RemoveContainer      bool
	RemoveContainerState bool
}

func RollbackProvisioning(patp string, opts RollbackOptions) error {
	var rollbackErrors []error

	if opts.UploadArchivePath != "" {
		if err := os.Remove(opts.UploadArchivePath); err != nil && !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove upload archive %s: %w", opts.UploadArchivePath, err))
		}
	}

	if opts.RemoveContainer {
		if err := orchestration.DeleteContainer(patp); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("delete container %s: %w", patp, err))
		}
	}

	if err := config.RemoveUrbitConfig(patp); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("remove urbit config %s: %w", patp, err))
	}

	if err := shipcreator.RemoveSysConfigPier(patp); err != nil {
		rollbackErrors = append(rollbackErrors, fmt.Errorf("remove ship %s from system config: %w", patp, err))
	}

	if opts.CustomPierPath != "" {
		if err := os.RemoveAll(opts.CustomPierPath); err != nil && !os.IsNotExist(err) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("remove custom pier path %s: %w", opts.CustomPierPath, err))
		}
	} else {
		if err := orchestration.DeleteVolume(patp); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("delete docker volume %s: %w", patp, err))
		}
	}

	if opts.RemoveContainerState {
		config.DeleteContainerState(patp)
	}

	return errors.Join(rollbackErrors...)
}
