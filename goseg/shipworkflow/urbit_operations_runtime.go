package shipworkflow

import (
	"fmt"
	"strings"
	"time"

	"groundseg/broadcast"
	"groundseg/click"
	"groundseg/docker"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

var runTransitionedOperationFn = RunTransitionedOperation

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
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionUrbitDomain),
		startEvent:     "loading",
		successEvent:   "done",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     time.Second,
	},
		transitionStep[string]{
			Event: "success",
			Run: func() error {
				alias := urbitPayload.Payload.Domain
				oldDomain := currentConf.WgURL
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("Failed to check Urbit domain alias for %s: %w", patp, err)
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
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionMinIODomain),
		startEvent:     "loading",
		successEvent:   "done",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     time.Second,
	},
		transitionStep[string]{
			Event: "success",
			Run: func() error {
				alias := urbitPayload.Payload.Domain
				oldDomain := fmt.Sprintf("s3.%s", currentConf.WgURL)
				areAliases, err := AreSubdomainsAliases(alias, oldDomain)
				if err != nil {
					return fmt.Errorf("Failed to check MinIO domain alias for %s: %w", patp, err)
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
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionChopOnUpgrade),
		startEvent:     "loading",
		clearEvent:     "",
		clearDelay:     3 * time.Second,
		err: func(error) string {
			return "error"
		},
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

func togglePower(patp string) error {
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionTogglePower),
		startEvent:     "loading",
		clearEvent:     "",
		clearDelay:     0,
		err: func(error) string {
			return "error"
		},
	},
		transitionStep[string]{
			Run: func() error {
				shipConf := getUrbitConfigFn(patp)
				statuses, err := docker.GetShipStatus([]string{patp})
				if err != nil {
					return fmt.Errorf("Failed to get ship status for %s: %w", patp, err)
				}
				status, exists := statuses[patp]
				if !exists {
					return fmt.Errorf("Failed to get ship status for %s: %w", patp, errShipStatusNotFound)
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
						return err
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
						if err := docker.StopContainerByName(patp); err != nil {
							return fmt.Errorf("failed to stop %s: %w", patp, err)
						}
						return fmt.Errorf("failed to stop %s with |exit: %w", patp, err)
					}
				} else if shipConf.BootStatus == "boot" && !isRunning {
					_, err := docker.StartContainer(patp, "vere")
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
	)
}

func toggleDevMode(patp string) error {
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionToggleDevMode),
		startEvent:     "loading",
		clearEvent:     "",
		clearDelay:     0,
		err: func(error) string {
			return "error"
		},
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
					return err
				}
				_, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return err
				}
				return nil
			},
		},
	)
}

func rebuildContainer(patp string) error {
	shipConf := getUrbitConfigFn(patp)
	return runTransitionedOperationFn(patp, "rebuildContainer", "loading", "success", 3*time.Second, func() error {
		if err := urbitCleanDelete(patp); err != nil {
			return err
		}
		if shipConf.BootStatus != "noboot" {
			if _, err := docker.StartContainer(patp, "vere"); err != nil {
				return fmt.Errorf("Failed to start for rebuild container %s: %w", patp, err)
			}
			return nil
		}
		if _, err := docker.CreateContainer(patp, "vere"); err != nil {
			return fmt.Errorf("Failed to create for rebuild container %s: %w", patp, err)
		}
		return nil
	})
}

func toggleNetwork(patp string) error {
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionToggleNetwork),
		startEvent:     "loading",
		clearEvent:     "",
		clearDelay:     0,
		err: func(error) string {
			return "error"
		},
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
						return err
					}
				} else if currentNetwork != "wireguard" && settings.WgRegistered {
					if err := persistShipConf(patp, func(conf *structs.UrbitDocker) error {
						conf.Network = "wireguard"
						return nil
					}); err != nil {
						return fmt.Errorf("Couldn't update urbit config: %w", err)
					}
					if err := urbitCleanDelete(patp); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("No remote registration")
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Couldn't start %v: %w", patp, err)
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
			return fmt.Errorf("failed to get ship status for %s: %w", patp, err)
		}
		status, exists := statusMap[patp]
		if !exists {
			return fmt.Errorf("running status for %s doesn't exist: %w", patp, errShipStatusNotFound)
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
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionToggleMinIOLink),
		startEvent:     "loading",
		clearEvent:     "",
		clearDelay:     1 * time.Second,
		err: func(error) string {
			return "error"
		},
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
					return fmt.Errorf("Failed to unlink MinIO information %s: %w", patp, err)
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
					return fmt.Errorf("Failed to create MinIO service account for %s: %w", patp, err)
				}

				// link to urbit
				if err := click.LinkStorage(patp, endpoint, svcAccount); err != nil {
					return fmt.Errorf("Failed to link MinIO information %s: %w", patp, err)
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
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionLoom),
		startEvent:     "loading",
		successEvent:   "done",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     time.Second,
		err: func(error) string {
			return "error"
		},
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
					return err
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Couldn't start %v: %w", patp, err)
					}
				}
				return nil
			},
		},
	)
}

func handleSnapTime(patp string, urbitPayload structs.WsUrbitPayload) error {
	return runUrbitTransitionTemplateFn(patp, urbitTransitionTemplate{
		transitionType: string(transition.UrbitTransitionSnapTime),
		startEvent:     "loading",
		successEvent:   "done",
		emitSuccess:    true,
		clearEvent:     "",
		clearDelay:     time.Second,
		err: func(error) string {
			return "error"
		},
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
					return err
				}
				if shipConf.BootStatus == "boot" {
					if _, err := docker.StartContainer(patp, "vere"); err != nil {
						return fmt.Errorf("Couldn't start %v: %w", patp, err)
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
		return fmt.Errorf("Schedule pack unknown interval type: %s", intervalType)
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
		return fmt.Errorf("Failed to set new size limit for %s: %w", patp, err)
	}
	return nil
}
