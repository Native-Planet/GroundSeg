package shipworkflow

import (
	"fmt"

	"groundseg/click"
)

type deskAction string

type deskRunner func(patp, desk string) error

type deskLifecycleSpec struct {
	desk       string
	transition string
	install    deskRunner
	revive     deskRunner
	remove     deskRunner
}

const (
	deskActionInstall   deskAction = "install"
	deskActionUninstall deskAction = "uninstall"
)

func defaultDeskInstallRunner(patp, desk string) error {
	return click.InstallDesk(patp, "~nattyv", desk)
}

func installPenpaiCompanion(patp string) error {
	return runDeskLifecycle(patp, deskActionInstall, deskLifecycleSpec{
		desk:       "penpai",
		transition: "penpaiCompanion",
		install:    defaultDeskInstallRunner,
		revive:     click.ReviveDesk,
		remove:     click.UninstallDesk,
	})
}

func uninstallPenpaiCompanion(patp string) error {
	return runDeskLifecycle(patp, deskActionUninstall, deskLifecycleSpec{
		desk:       "penpai",
		transition: "penpaiCompanion",
		install:    defaultDeskInstallRunner,
		revive:     click.ReviveDesk,
		remove:     click.UninstallDesk,
	})
}

func installGallseg(patp string) error {
	return runDeskLifecycle(patp, deskActionInstall, deskLifecycleSpec{
		desk:       "groundseg",
		transition: "gallseg",
		install:    defaultDeskInstallRunner,
		revive:     click.ReviveDesk,
		remove:     click.UninstallDesk,
	})
}

func uninstallGallseg(patp string) error {
	return runDeskLifecycle(patp, deskActionUninstall, deskLifecycleSpec{
		desk:       "groundseg",
		transition: "gallseg",
		install:    defaultDeskInstallRunner,
		revive:     click.ReviveDesk,
		remove:     click.UninstallDesk,
	})
}

func runDeskLifecycle(patp string, action deskAction, spec deskLifecycleSpec) error {
	return runDeskTransition(patp, spec.transition, func() error {
		switch action {
		case deskActionInstall:
			return runDeskInstallTransition(patp, spec)
		case deskActionUninstall:
			return runDeskUninstallTransition(patp, spec)
		default:
			return fmt.Errorf("unsupported desk action %q for %s", action, spec.desk)
		}
	})
}

func runDeskInstallTransition(patp string, spec deskLifecycleSpec) error {
	status, err := click.GetDesk(patp, spec.desk, true)
	if err != nil {
		return fmt.Errorf("failed to get %s desk info: %w", spec.desk, err)
	}
	switch status {
	case "not-found":
		if spec.install == nil {
			return fmt.Errorf("install action is not configured for %s desk", spec.desk)
		}
		if err := spec.install(patp, spec.desk); err != nil {
			return fmt.Errorf("failed to install %s desk: %w", spec.desk, err)
		}
	case "suspended":
		if spec.revive == nil {
			return fmt.Errorf("revive action is not configured for %s desk", spec.desk)
		}
		if err := spec.revive(patp, spec.desk); err != nil {
			return fmt.Errorf("failed to revive %s desk: %w", spec.desk, err)
		}
	case "running":
		return nil
	}
	if err := waitForDeskState(patp, spec.desk, "running", true); err != nil {
		return fmt.Errorf("failed waiting for %s desk installation: %w", spec.desk, err)
	}
	return nil
}

func runDeskUninstallTransition(patp string, spec deskLifecycleSpec) error {
	if spec.remove == nil {
		return fmt.Errorf("remove action is not configured for %s desk", spec.desk)
	}
	if err := spec.remove(patp, spec.desk); err != nil {
		return fmt.Errorf("failed to uninstall %s desk: %w", spec.desk, err)
	}
	if err := waitForDeskState(patp, spec.desk, "running", false); err != nil {
		return fmt.Errorf("failed waiting for %s desk removal: %w", spec.desk, err)
	}
	return nil
}
