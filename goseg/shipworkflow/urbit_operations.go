package shipworkflow

import (
	"fmt"
	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/exporter"
	"groundseg/shipcleanup"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	getUrbitConfigFn            = config.UrbitConf
	loadUrbitConfigFn           = config.LoadUrbitConfig
	getContainerStatesFn        = config.GetContainerState
	updateContainerStateFn      = config.UpdateContainerState
	getStartramSettingsSnapshot = config.StartramSettingsSnapshot
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

func startramReminder(patp string, remind bool) error {
	if err := PersistUrbitConfig(patp, func(conf *structs.UrbitDocker) error {
		conf.StartramReminder = remind
		return nil
	}, persistUrbitConfigFn); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func urbitDeleteStartramService(patp string, service string) error {
	settings := getStartramSettingsSnapshot()
	// check svc type, reconstruct subdomain

	// Accessing parts of the URL
	parts := strings.Split(settings.EndpointURL, ".")
	if len(parts) < 2 {
		return fmt.Errorf("Failed to recreate subdomain for manual service deletion")
	} else {
		baseURL := parts[len(parts)-2] + "." + parts[len(parts)-1]
		var subdomain string
		switch service {
		case "urbit-web":
			subdomain = fmt.Sprintf("%s.%s", patp, baseURL)
		case "urbit-ames":
			subdomain = fmt.Sprintf("%s.%s.%s", "ames", patp, baseURL)
		case "minio":
			subdomain = fmt.Sprintf("%s.%s.%s", "s3", patp, baseURL)
		case "minio-console":
			subdomain = fmt.Sprintf("%s.%s.%s", "console.s3", patp, baseURL)
		case "minio-bucket":
			subdomain = fmt.Sprintf("%s.%s.%s", "bucket.s3", patp, baseURL)
		default:
			return fmt.Errorf("Invalid service type: unable to manually delete service")
		}
		if err := startram.SvcDelete(subdomain, service); err != nil {
			return fmt.Errorf("Failed to delete startram service: %w", err)
		} else {
			_, err := startram.SyncRetrieve()
			if err != nil {
				return fmt.Errorf("Failed to retrieve after manual service deletion: %w", err)
			}
		}
		return nil
	}
}

func packPier(patp string) error {
	return runPackLifecycle(patp)
}

func RunPack(patp string) error {
	return runPackLifecycle(patp)
}

func RunScheduledPack(patp string, delay time.Duration) error {
	if delay > 0 {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s in %v", patp, delay))
		time.Sleep(delay)
	} else {
		zap.L().Info(fmt.Sprintf("Starting scheduled pack for %s", patp))
	}
	return runPackLifecycle(patp)
}

func runPackLifecycle(patp string) error {
	packError := func(err error) error {
		return fmt.Errorf("pack operation: %w", err)
	}
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionPack),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "packing",
			SuccessEvent: "success",
			EmitSuccess:  true,
			ErrorEvent:   func(err error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   3 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return packError(wrapLifecycleError(patp, "Failed to get ship status", err))
				}
				status, exists := statuses[patp]
				if !exists {
					return packError(fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp))
				}
				// running
				if strings.Contains(status, "Up") {
					// send |pack
					if err := click.SendPack(patp); err != nil {
						return packError(wrapLifecycleError(patp, "Failed to send pack command", err))
					}
					// not running
				} else {
					// set DesiredStatus to prevent auto-restart when pack container exits
					if containerState, exists := getContainerStatesFn()[patp]; exists {
						containerState.DesiredStatus = "stopped"
						updateContainerStateFn(patp, containerState)
					}
					// switch boot status to pack
					err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
						conf.BootStatus = "pack"
						return nil
					})
					if err != nil {
						return packError(wrapLifecycleError(patp, "Failed to update urbit config to pack", err))
					}
				}
				// set last meld
				now := time.Now().Unix()
				err = persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.MeldLast = strconv.FormatInt(now, 10)
					return nil
				})
				if err != nil {
					return packError(wrapLifecycleError(patp, "Failed to update urbit config with last meld time", err))
				}
				return nil
			},
		},
	)
}

func packMeldPier(patp string) error {
	var isRunning bool
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionPackMeld),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "packing",
			SuccessEvent: "success",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   3 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return wrapLifecycleError(patp, "Failed to get ship status", err)
				}
				status, exists := statuses[patp]
				if !exists {
					return fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp)
				}
				isRunning = strings.Contains(status, "Up")
				// set DesiredStatus to prevent auto-restart from die/stop event handlers during maintenance
				if containerState, exists := getContainerStatesFn()[patp]; exists {
					containerState.DesiredStatus = "stopped"
					updateContainerStateFn(patp, containerState)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "stopping",
			Run: func() error {
				if !isRunning {
					return nil
				}
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for pack & meld %s: %v", patp, err))
					if err = docker.StopContainerByName(patp); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to stop ship for pack & meld %s: %v", patp, err))
					}
				}
				if err := WaitComplete(patp); err != nil {
					return wrapLifecycleError(patp, "Failed waiting for stop completion on %s before pack & meld", err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Run: func() error {
				// start ship as pack
				zap.L().Info(fmt.Sprintf("Attempting to urth pack %s", patp))
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "pack"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to pack", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start pack container", err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for urth pack to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for pack completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "melding",
			Run: func() error {
				// start ship as meld
				zap.L().Info(fmt.Sprintf("Attempting to urth meld %s", patp))
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "meld"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to meld", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start meld container", err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for urth meld to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for meld completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "starting",
			Run: func() error {
				if !isRunning {
					return nil
				}
				// restore DesiredStatus so normal auto-restart behavior resumes
				if containerState, exists := getContainerStatesFn()[patp]; exists {
					containerState.DesiredStatus = "running"
					updateContainerStateFn(patp, containerState)
				}
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "boot"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to meld", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start meld container", err)
				}
				return nil
			},
		},
	)
}

func wrapLifecycleError(patp string, detail string, err error) error {
	return fmt.Errorf("%s for %s: %w", detail, patp, err)
}

func toggleAlias(patp string) error {
	currentConf := getUrbitConfigFn(patp)
	nextShowUrbitWeb := "custom"
	if currentConf.ShowUrbitWeb == "custom" {
		nextShowUrbitWeb = "default"
	}
	if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
		conf.ShowUrbitWeb = nextShowUrbitWeb
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func setUrbitDomain(patp string, urbitPayload structs.WsUrbitPayload) error {
	currentConf := getUrbitConfigFn(patp)
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionUrbitDomain),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "loading",
			SuccessEvent: "done",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   time.Second,
		},
		transitionStep[string]{
			Event: "success",
			Run: func() error {
				alias := urbitPayload.Payload.Domain
				oldDomain := currentConf.WgURL
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("Failed to check Urbit domain alias for %s: %v", patp, err)
				}
				if !areAliases {
					return fmt.Errorf("Invalid Urbit domain alias for %s", patp)
				}
				if err := startram.AliasCreate(patp, alias); err != nil {
					return fmt.Errorf("set urbit domain alias for %s: %w", patp, err)
				}
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.CustomUrbitWeb = alias
					conf.ShowUrbitWeb = "custom"
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
	)
}

func setMinIODomain(patp string, urbitPayload structs.WsUrbitPayload) error {
	currentConf := getUrbitConfigFn(patp)
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionMinIODomain),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "loading",
			SuccessEvent: "done",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   time.Second,
		},
		transitionStep[string]{
			Event: "success",
			Run: func() error {
				alias := urbitPayload.Payload.Domain
				oldDomain := fmt.Sprintf("s3.%s", currentConf.WgURL)
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("Failed to check MinIO domain alias for %s: %v", patp, err)
				}
				if !areAliases {
					return fmt.Errorf("Invalid MinIO domain alias for %s", patp)
				}
				if err := startram.AliasCreate(fmt.Sprintf("s3.%s", patp), alias); err != nil {
					return fmt.Errorf("set minio domain alias for %s: %w", patp, err)
				}
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.CustomS3Web = alias
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
	)
}

func toggleChopOnVereUpdate(patp string) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionChopOnUpgrade),
		transitionPlan[string]{
			EmitStart:  true,
			StartEvent: "loading",
			ClearEvent: "",
			ClearDelay: 3 * time.Second,
			ErrorEvent: func(error) string { return "error" },
		},
		transitionStep[string]{
			Run: func() error {
				currentConf := getUrbitConfigFn(patp)
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.ChopOnUpgrade = !currentConf.ChopOnUpgrade
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
	)
}

func deleteShip(patp string) error {
	settings := getStartramSettingsSnapshot()
	removeServices := false
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionDeleteShip),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "stopping",
			SuccessEvent: "done",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   1 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				// update DesiredStatus to 'stopped'
				contConf := getContainerStatesFn()
				patpConf := contConf[patp]
				patpConf.DesiredStatus = "stopped"
				contConf[patp] = patpConf
				updateContainerStateFn(patp, patpConf)
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					if err := docker.StopContainerByName(patp); err != nil {
						return fmt.Errorf("Couldn't stop docker container for %v: %v", patp, err)
					}
				}
				if err := docker.DeleteContainer(patp); err != nil {
					return fmt.Errorf("Couldn't delete docker container for %v: %v", patp, err)
				}
				removeServices = settings.WgRegistered
				return nil
			},
		},
		transitionStep[string]{
			Event: "removing-services",
			EmitWhen: func() bool {
				return removeServices
			},
			Run: func() error {
				if err := startram.SvcDelete(patp, "urbit"); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v: %v", patp, err))
				}
				if err := startram.SvcDelete("s3."+patp, "s3"); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v: %v", patp, err))
				}
				if err := docker.DeleteContainer("minio_" + patp); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't delete minio docker container for %v: %v", patp, err))
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "deleting",
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				customPath := shipConf.CustomPierLocation
				if err := shipcleanup.RollbackProvisioning(patp, shipcleanup.RollbackOptions{
					CustomPierPath:       customPath,
					RemoveContainerState: true,
				}); err != nil {
					zap.L().Error(fmt.Sprintf("Ship cleanup encountered errors for %v: %v", patp, err))
				}
				zap.L().Info(fmt.Sprintf("%v container deleted", patp))
				// remove from broadcast
				if err := broadcast.ReloadUrbits(); err != nil {
					zap.L().Error(fmt.Sprintf("Error updating broadcast: %v", err))
				}
				return nil
			},
		},
	)
}

func exportShip(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionExportShip),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "stopping",
			SuccessEvent: "ready",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   1 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "noboot"
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				// stop container
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					if err := docker.StopContainerByName(patp); err != nil {
						return err
					}
				}
				// whitelist the patp token pair
				if err := exporter.WhitelistContainer(patp, urbitPayload.Token); err != nil {
					return err
				}
				return nil
			},
		},
	)
}

func exportBucket(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionExportBucket),
		transitionPlan[string]{
			SuccessEvent: "ready",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   1 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				containerName := fmt.Sprintf("minio_%s", patp)
				// whitelist the patp token pair
				if err := exporter.WhitelistContainer(containerName, urbitPayload.Token); err != nil {
					return err
				}
				return nil
			},
		},
	)
}

func togglePower(patp string) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionTogglePower),
		transitionPlan[string]{
			EmitStart:  true,
			StartEvent: "loading",
			ErrorEvent: func(error) string { return "error" },
			ClearEvent: "",
			ClearDelay: 0,
		},
		transitionStep[string]{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return fmt.Errorf("Failed to get ship status for %s: %v", patp, err)
				}
				status, exists := statuses[patp]
				if !exists {
					return fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp)
				}
				isRunning := strings.Contains(status, "Up")
				if shipConf.BootStatus == "noboot" {
					if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
						conf.BootStatus = "boot"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					_, err := docker.StartContainer(patp, "vere")
					if err != nil {
						zap.L().Error(fmt.Sprintf("%v", err))
					}
				} else if shipConf.BootStatus == "boot" && isRunning {
					// set DesiredStatus before stopping to prevent auto-restart from die/stop event handlers
					if containerState, exists := getContainerStatesFn()[patp]; exists {
						containerState.DesiredStatus = "stopped"
						updateContainerStateFn(patp, containerState)
					}
					if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
						conf.BootStatus = "noboot"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					err := click.BarExit(patp)
					if err != nil {
						zap.L().Error(fmt.Sprintf("%v", err))
						if err := docker.StopContainerByName(patp); err != nil {
							zap.L().Error(fmt.Sprintf("%v", err))
						}
					}
				} else if shipConf.BootStatus == "boot" && !isRunning {
					_, err := docker.StartContainer(patp, "vere")
					if err != nil {
						zap.L().Error(fmt.Sprintf("%v", err))
					}
				}
				return nil
			},
		},
	)
}

func toggleDevMode(patp string) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionToggleDevMode),
		transitionPlan[string]{
			EmitStart:  true,
			StartEvent: "loading",
			ErrorEvent: func(error) string { return "error" },
			ClearEvent: "",
			ClearDelay: 0,
		},
		transitionStep[string]{
			Run: func() error {
				currentConf := getUrbitConfigFn(patp)
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.DevMode = !currentConf.DevMode
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				if err := urbitCleanDelete(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
				}
				return nil
			},
		},
	)
}

func rebuildContainer(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	return RunTransitionedOperation(patp, "rebuildContainer", "loading", "success", 3*time.Second, func() error {
		if err := urbitCleanDelete(patp); err != nil {
			zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
		}
		if shipConf.BootStatus != "noboot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				return fmt.Errorf("Failed to start for rebuild container %s: %v", patp, err)
			}
			return nil
		}
		if _, err := docker.CreateContainer(patp, "vere"); err != nil {
			return fmt.Errorf("Failed to create for rebuild container %s: %v", patp, err)
		}
		return nil
	})
}

func toggleNetwork(patp string) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionToggleNetwork),
		transitionPlan[string]{
			EmitStart:  true,
			StartEvent: "loading",
			ErrorEvent: func(error) string { return "error" },
			ClearEvent: "",
			ClearDelay: 0,
		},
		transitionStep[string]{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				currentNetwork := shipConf.Network
				settings := getStartramSettingsSnapshot()
				zap.L().Warn(fmt.Sprintf("%v", currentNetwork))
				if currentNetwork == "wireguard" {
					if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
						conf.Network = "bridge"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					if err := urbitCleanDelete(patp); err != nil {
						zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
					}
				} else if currentNetwork != "wireguard" && settings.WgRegistered {
					if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
						conf.Network = "wireguard"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					if err := urbitCleanDelete(patp); err != nil {
						zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
					}
				} else {
					return fmt.Errorf("No remote registration")
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
					}
				}
				return nil
			},
		},
	)
}

func toggleBootStatus(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	nextBootStatus := "ignore"
	if shipConf.BootStatus == "ignore" {
		statusMap, err := docker.GetShipStatus([]string{patp})
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to get ship status for %s", patp))
		}
		status, exists := statusMap[patp]
		if !exists {
			zap.L().Error(fmt.Sprintf("Running status for %s doesn't exist", patp))
		}
		if strings.Contains(status, "Up") {
			nextBootStatus = "boot"
		} else {
			nextBootStatus = "noboot"
		}
	}
	if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
		conf.BootStatus = nextBootStatus
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	return nil
}

func toggleAutoReboot(patp string) error {
	if err := loadUrbitConfigFn(patp); err != nil {
		return fmt.Errorf("Failed to load fresh urbit config: %w", err)
	}
	currentConf := getUrbitConfigFn(patp)
	if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
		conf.DisableShipRestarts = !currentConf.DisableShipRestarts
		return nil
	}); err != nil {
		return fmt.Errorf("Couldn't update urbit config: %w", err)
	}
	broadcast.BroadcastToClients()
	return nil
}

func toggleMinIOLink(patp string) error {
	var isLinked bool
	var endpoint string
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionToggleMinIOLink),
		transitionPlan[string]{
			EmitStart:  true,
			StartEvent: "loading",
			ErrorEvent: func(error) string { return "error" },
			ClearEvent: "",
			ClearDelay: 1 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				// todo: scry for actual info
				isLinked = shipConf.MinIOLinked
				endpoint = shipConf.CustomS3Web
				if endpoint == "" {
					endpoint = fmt.Sprintf("s3.%s", shipConf.WgURL)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "unlinking",
			EmitWhen: func() bool {
				return isLinked
			},
			Run: func() error {
				if err := click.UnlinkStorage(patp); err != nil {
					return fmt.Errorf("Failed to unlink MinIO information %s: %v", patp, err)
				}

				// Update config
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.MinIOLinked = false
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "unlink-success",
			EmitWhen: func() bool {
				return isLinked
			},
			Run: func() error {
				return nil
			},
		},
		transitionStep[string]{
			Event: "linking",
			EmitWhen: func() bool {
				return !isLinked
			},
			Run: func() error {
				// create service account
				svcAccount, err := docker.CreateMinIOServiceAccount(patp)
				if err != nil {
					return fmt.Errorf("Failed to create MinIO service account for %s: %v", patp, err)
				}

				// link to urbit
				if err := click.LinkStorage(patp, endpoint, svcAccount); err != nil {
					return fmt.Errorf("Failed to link MinIO information %s: %v", patp, err)
				}

				// Update config
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.MinIOLinked = true
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "success",
			EmitWhen: func() bool {
				return !isLinked
			},
			Run: func() error {
				return nil
			},
		},
	)
}

func handleLoom(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionLoom),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "loading",
			SuccessEvent: "done",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.LoomSize = urbitPayload.Payload.Value
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				if err := urbitCleanDelete(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
					}
				}
				return nil
			},
		},
	)
}

func handleSnapTime(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionSnapTime),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "loading",
			SuccessEvent: "done",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.SnapTime = urbitPayload.Payload.Value
					return nil
				}); err != nil {
					return fmt.Errorf("Couldn't update urbit config: %w", err)
				}
				if err := urbitCleanDelete(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Container deletion for rebuild-container failed: %v", err))
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't start %v: %v", patp, err))
					}
				}
				return nil
			},
		},
	)
}

func schedulePack(patp string, urbitPayload structs.WsUrbitPayload) error {
	frequency := urbitPayload.Payload.Frequency
	// frequency not 0
	if frequency < 1 {
		return fmt.Errorf("pack frequency cannot be 0!")
	}
	intervalType := urbitPayload.Payload.IntervalType
	switch intervalType {
	case "month", "week", "day":
		if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
			conf.MeldTime = urbitPayload.Payload.Time
			conf.MeldSchedule = true
			conf.MeldScheduleType = intervalType
			conf.MeldFrequency = frequency
			conf.MeldDay = urbitPayload.Payload.Day
			conf.MeldDate = urbitPayload.Payload.Date
			return nil
		}); err != nil {
			return fmt.Errorf("Failed to update pack schedule: %w", err)
		}
	default:
		return fmt.Errorf("Schedule pack unknown interval type: %v", intervalType)
	}
	broadcast.PublishSchedulePack("schedule")
	return nil
}

func pausePackSchedule(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
		conf.MeldSchedule = false
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to pause pack schedule: %w", err)
	}
	return nil
}

func setNewMaxPierSize(patp string, urbitPayload structs.WsUrbitPayload) error {
	if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
		conf.SizeLimit = urbitPayload.Payload.Value
		return nil
	}); err != nil {
		return fmt.Errorf("Failed to set new size limit for %s: %v", patp, err)
	}
	return nil
}

func rollChopPier(patp string) error {
	var isRunning bool
	return runUrbitTransition(
		patp,
		string(transition.UrbitTransitionRollChop),
		transitionPlan[string]{
			EmitStart:    true,
			StartEvent:   "rolling",
			SuccessEvent: "success",
			EmitSuccess:  true,
			ErrorEvent:   func(error) string { return "error" },
			ClearEvent:   "",
			ClearDelay:   3 * time.Second,
		},
		transitionStep[string]{
			Run: func() error {
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return wrapLifecycleError(patp, "Failed to get ship status", err)
				}
				status, exists := statuses[patp]
				if !exists {
					return fmt.Errorf("Failed to get ship status for %s: status doesn't exist!", patp)
				}
				isRunning = strings.Contains(status, "Up")
				return nil
			},
		},
		transitionStep[string]{
			Event: "stopping",
			EmitWhen: func() bool {
				return isRunning
			},
			Run: func() error {
				if err := click.BarExit(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop ship with |exit for roll & chop %s: %v", patp, err))
					if err = docker.StopContainerByName(patp); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to stop ship for roll & chop %s: %v", patp, err))
					}
				}
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for stop completion on %s before roll & chop: %w", patp, err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Run: func() error {
				// start ship as roll
				zap.L().Info(fmt.Sprintf("Attempting to roll %s", patp))
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "roll"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to roll", err)
				}
				if _, err := docker.StartContainer(patp, "vere"); err != nil {
					return wrapLifecycleError(patp, "Failed to start roll container", err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for roll to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for roll completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "chopping",
			Run: func() error {
				// start ship as chop
				zap.L().Info(fmt.Sprintf("Attempting to chop %s", patp))
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "chop"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to chop", err)
				}
				if _, err := docker.StartContainer(patp, "vere"); err != nil {
					return wrapLifecycleError(patp, "Failed to start chop container", err)
				}

				zap.L().Info(fmt.Sprintf("Waiting for chop to complete for %s", patp))
				if err := WaitComplete(patp); err != nil {
					return fmt.Errorf("Failed waiting for chop completion on %s: %w", patp, err)
				}
				return nil
			},
		},
		transitionStep[string]{
			Event: "starting",
			EmitWhen: func() bool {
				return isRunning
			},
			Run: func() error {
				if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
					conf.BootStatus = "boot"
					return nil
				}); err != nil {
					return wrapLifecycleError(patp, "Failed to update urbit config to chop", err)
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return wrapLifecycleError(patp, "Failed to start chop container", err)
				}
				return nil
			},
		},
	)
}
