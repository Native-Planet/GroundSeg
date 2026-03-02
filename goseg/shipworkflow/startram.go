package shipworkflow

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker/orchestration"
	"groundseg/internal/workflow"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

type startramRuntime struct {
	dispatchStartramToggleNetworkFn func(string) error
	startramDispatchUrbitPayloadFn  func(structs.WsUrbitPayload) error
	startramPublishEventFn          func(structs.Event)
	startramRuntimeFn               func() orchestration.StartramRuntime
	startramRecoverWireguardFleetFn func(piers []string, deleteMinioClient bool) error
}

func defaultStartramRuntime() startramRuntime {
	return startramRuntime{
		dispatchStartramToggleNetworkFn: ToggleNetwork,
		startramDispatchUrbitPayloadFn:  defaultDispatchUrbitPayload,
		startramPublishEventFn:          startram.PublishEvent,
		startramRuntimeFn: func() orchestration.StartramRuntime {
			return orchestration.NewStartramRuntime(
				orchestration.WithStartramServiceLoaders(
					broadcast.GetStartramServices,
					broadcast.LoadStartramRegions,
				),
			)
		},
		startramRecoverWireguardFleetFn: func(piers []string, deleteMinioClient bool) error {
			rt := orchestration.NewRuntime()
			return RecoverWireguardFleet(NewWireguardRecoveryRuntime(rt), piers, deleteMinioClient)
		},
	}
}

func withDefaultsStartramRuntime(runtime startramRuntime) startramRuntime {
	if runtime.dispatchStartramToggleNetworkFn == nil {
		runtime.dispatchStartramToggleNetworkFn = ToggleNetwork
	}
	if runtime.startramDispatchUrbitPayloadFn == nil {
		runtime.startramDispatchUrbitPayloadFn = defaultDispatchUrbitPayload
	}
	if runtime.startramPublishEventFn == nil {
		runtime.startramPublishEventFn = startram.PublishEvent
	}
	if runtime.startramRuntimeFn == nil {
		runtime.startramRuntimeFn = func() orchestration.StartramRuntime {
			return orchestration.NewStartramRuntime(
				orchestration.WithStartramServiceLoaders(
					broadcast.GetStartramServices,
					broadcast.LoadStartramRegions,
				),
			)
		}
	}
	if runtime.startramRecoverWireguardFleetFn == nil {
		runtime.startramRecoverWireguardFleetFn = func(piers []string, deleteMinioClient bool) error {
			rt := orchestration.NewRuntime()
			return RecoverWireguardFleet(NewWireguardRecoveryRuntime(rt), piers, deleteMinioClient)
		}
	}
	return runtime
}

func defaultDispatchUrbitPayload(payload structs.WsUrbitPayload) error {
	return dispatchUrbitPayloadWithRuntime(defaultStartramRuntime(), payload)
}

func dispatchUrbitPayloadWithRuntime(runtime startramRuntime, payload structs.WsUrbitPayload) error {
	if payload.Payload.Action == "" {
		return fmt.Errorf("no urbit action provided")
	}
	if payload.Payload.Patp == "" {
		return fmt.Errorf("no patp provided for urbit action %q", payload.Payload.Action)
	}

	switch payload.Payload.Action {
	case "toggle-network":
		return runtime.dispatchStartramToggleNetworkFn(payload.Payload.Patp)
	default:
		return fmt.Errorf("unsupported urbit action in startram dispatch: %q", payload.Payload.Action)
	}
}

func startramEvent(transitionType transition.EventType, data any) structs.Event {
	return structs.Event{
		Type: string(transitionType),
		Data: data,
	}
}

func runStartramTransitionWithRuntime(runtime startramRuntime, transitionType transition.EventType, plan transitionPlan[structs.Event], steps ...transitionStep[structs.Event]) error {
	runtime = withDefaultsStartramRuntime(runtime)
	if plan.ErrorEvent == nil {
		plan.ErrorEvent = func(err error) structs.Event {
			return startramEvent(transitionType, fmt.Sprintf("Error: %v", err))
		}
	}
	return runTransitionLifecycle[structs.Event](
		defaultWorkflowRuntime(),
		func(event structs.Event) { runtime.startramPublishEventFn(event) },
		plan,
		steps...,
	)
}

func HandleStartramServices() error {
	return runStartramServicesWithRuntime(defaultStartramRuntime())
}

func runStartramServicesWithRuntime(runtime startramRuntime) error {
	runtime = withDefaultsStartramRuntime(runtime)
	return runtime.startramRuntimeFn().GetStartramServicesFn()
}

func HandleStartramRegions() error {
	return runStartramRegionsWithRuntime(defaultStartramRuntime())
}

func runStartramRegionsWithRuntime(runtime startramRuntime) error {
	runtime = withDefaultsStartramRuntime(runtime)
	return runtime.startramRuntimeFn().LoadStartramRegionsFn()
}

func HandleStartramRestart() error {
	return runStartramRestartWithRuntime(defaultStartramRuntime())
}

func runStartramRestartWithRuntime(runtime startramRuntime) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	zap.L().Info("Restarting StarTram")
	settings := startramRuntime.StartramSettingsSnapshotFn()
	steps := []transitionStep[structs.Event]{
		{
			Run: func() error {
				if !settings.WgOn {
					return fmt.Errorf("startram is disabled")
				}
				return nil
			},
		},
	}
	if settings.WgOn {
		steps = append(steps,
			transitionStep[structs.Event]{
				Event: startramEvent(transition.StartramTransitionRestart, "urbits"),
			},
			transitionStep[structs.Event]{
				Event: startramEvent(transition.StartramTransitionRestart, "minios"),
			},
			transitionStep[structs.Event]{
				Run: func() error {
					zap.L().Info("Recreating containers")
					return runtime.startramRecoverWireguardFleetFn(settings.Piers, true)
				},
			},
		)
	}
	return runStartramTransitionTemplate(
		runtime,
		startramTransitionTemplate{
			transitionType: transition.StartramTransitionRestart,
			startEvent:     startramEvent(transition.StartramTransitionRestart, "startram"),
			successEvent:   startramEvent(transition.StartramTransitionRestart, string(transition.StartramTransitionDone)),
			emitSuccess:    true,
			clearEvent:     startramEvent(transition.StartramTransitionRestart, ""),
			clearDelay:     3 * time.Second,
		},
		steps...,
	)
}

func HandleStartramToggle() error {
	return runStartramToggleWithRuntime(defaultStartramRuntime())
}

func runStartramToggleWithRuntime(runtime startramRuntime) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	settings := startramRuntime.StartramSettingsSnapshotFn()
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionToggle,
		startEvent:     startramEvent(transition.StartramTransitionToggle, transition.StartramTransitionLoading),
		clearEvent:     startramEvent(transition.StartramTransitionToggle, nil),
		clearDelay:     3 * time.Second,
	},
		transitionStep[structs.Event]{
			Run: func() error {
				steps := []workflow.Step{}
				if settings.WgOn {
					steps = append(steps, workflow.Step{
						Name: "update config to disable wireguard",
						Run: func() error {
							return startramRuntime.UpdateConfTypedFn(config.WithWgOn(false))
						},
					})
					steps = append(steps, workflow.Step{
						Name: "stop wireguard container",
						Run: func() error {
							return startramRuntime.StopContainerByNameFn("wireguard")
						},
					})
					// toggle ships back to local
					for _, patp := range settings.Piers {
						dockerConfig := startramRuntime.UrbitConfFn(patp)
						if dockerConfig.Network == "wireguard" {
							patpCopy := patp
							payload := structs.WsUrbitPayload{
								Payload: structs.WsUrbitAction{
									Type:   "urbit",
									Action: "toggle-network",
									Patp:   patpCopy,
								},
							}
							steps = append(steps, workflow.Step{
								Name: fmt.Sprintf("dispatch toggle-network for %s", patpCopy),
								Run: func(payload structs.WsUrbitPayload) func() error {
									return func() error { return runtime.startramDispatchUrbitPayloadFn(payload) }
								}(payload),
							})
						}
					}
				} else {
					steps = append(steps, workflow.Step{
						Name: "update config to enable wireguard",
						Run: func() error {
							return startramRuntime.UpdateConfTypedFn(config.WithWgOn(true))
						},
					})
					steps = append(steps, workflow.Step{
						Name: "start wireguard container",
						Run: func() error {
							_, err := startramRuntime.StartContainerFn("wireguard", "wireguard")
							return err
						},
					})
				}
				// delete mc
				steps = appendShipContainerRebuildSteps(steps, shipContainerRebuildRuntime{
					DeleteContainerFn: startramRuntime.DeleteContainerFn,
					LoadMCFn:          startramRuntime.LoadMCFn,
					LoadMinIOsFn:      startramRuntime.LoadMinIOsFn,
				}, shipContainerRebuildOptions{
					piers:             settings.Piers,
					deletePiers:       false,
					deleteMinioClient: true,
					loadMinIOClient:   true,
					loadMinIOs:        true,
				})
				if joinedErr := workflow.Join(steps, func(err error) {
					zap.L().Error(err.Error())
				}); joinedErr != nil {
					return joinedErr
				}
				return nil
			},
		},
	)
}

func HandleStartramRegister(regCode, region string) error {
	return runStartramRegisterWithRuntime(defaultStartramRuntime(), regCode, region)
}

func runStartramRegisterWithRuntime(runtime startramRuntime, regCode, region string) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionRegister,
		startEvent:     startramEvent(transition.StartramTransitionRegister, transition.StartramTransitionLoading),
		successEvent:   startramEvent(transition.StartramTransitionRegister, transition.StartramTransitionComplete),
		emitSuccess:    true,
		clearEvent:     startramEvent(transition.StartramTransitionRegister, nil),
		clearDelay:     3 * time.Second,
	},
		transitionStep[structs.Event]{
			Run: func() error {
				// Reset key pair
				if err := startramRuntime.CycleWgKeyFn(); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					return fmt.Errorf("cycle wireguard key: %w", err)
				}
				// Register startram key
				if err := startram.Register(regCode, region); err != nil {
					zap.L().Error(fmt.Sprintf("Failed registration: %v", err))
					return fmt.Errorf("startram register: %w", err)
				}
				return nil
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionRegister, transition.StartramTransitionServicesAction),
			Run: func() error {
				if err := startram.RegisterExistingShips(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to register ships: %v", err))
					return fmt.Errorf("register existing ships: %w", err)
				}
				return nil
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionRegister, transition.StartramTransitionStarting),
			Run: func() error {
				if err := startramRuntime.LoadWireguardFn(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to start Wireguard: %v", err))
					return fmt.Errorf("start wireguard: %w", err)
				}
				return nil
			},
		},
	)
}

func HandleStartramEndpoint(endpoint string) error {
	return runStartramEndpointWithRuntime(defaultStartramRuntime(), endpoint)
}

func runStartramEndpointWithRuntime(runtime startramRuntime, endpoint string) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	settings := startramRuntime.StartramSettingsSnapshotFn()
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionEndpoint,
		startEvent:     startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionInit),
		successEvent:   startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionComplete),
		emitSuccess:    true,
		clearEvent:     startramEvent(transition.StartramTransitionEndpoint, nil),
		clearDelay:     3 * time.Second,
	},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionStopping),
			Run: func() error {
				if !settings.WgOn {
					return nil
				}
				if err := startramRuntime.StopContainerByNameFn("wireguard"); err != nil {
					return fmt.Errorf("stop wireguard: %w", err)
				}
				return nil
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionUnregistering),
			Run: func() error {
				if !settings.WgRegistered {
					return nil
				}
				for _, p := range settings.Piers {
					if err := startramRuntime.SvcDeleteFn(p, "urbit"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v", p))
					}
					if err := startramRuntime.SvcDeleteFn("s3."+p, "s3"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v", p))
					}
				}
				return nil
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionConfiguring),
			Run: func() error {
				return startramRuntime.CycleWgKeyFn()
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionFinalizing),
			Run: func() error {
				return startramRuntime.UpdateConfTypedFn(
					config.WithEndpointURL(endpoint),
					config.WithWgRegistered(false),
				)
			},
		},
	)
}

func HandleStartramCancel(key string, reset bool) error {
	return runStartramCancelWithRuntime(defaultStartramRuntime(), key, reset)
}

func runStartramCancelWithRuntime(runtime startramRuntime, key string, reset bool) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionCancel,
		clearEvent:     startramEvent(transition.StartramTransitionCancel, nil),
		clearDelay:     3 * time.Second,
	},
		transitionStep[structs.Event]{
			Run: func() error {
				if reset {
					for _, svc := range startramRuntime.GetStartramConfigFn().Subdomains {
						if err := startramRuntime.SvcDeleteFn(svc.URL, svc.SvcType); err != nil {
							zap.L().Error(fmt.Sprintf("Couldn't delete service %v: %v", svc.URL, err))
						}
					}
				}
				if err := startram.CancelSub(key); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't cancel subscription: %v", err))
					return fmt.Errorf("cancel subscription: %w", err)
				}
				if err := startramRuntime.CycleWgKeyFn(); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					return fmt.Errorf("cycle wireguard key: %w", err)
				}
				return nil
			},
		},
	)
}

func HandleStartramReminder(remind bool) error {
	return runStartramReminderWithRuntime(defaultStartramRuntime(), remind)
}

func runStartramReminderWithRuntime(runtime startramRuntime, remind bool) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	ships := startramRuntime.ShipSettingsSnapshotFn()
	var steps []workflow.Step
	for _, patp := range ships.Piers {
		pier := patp
		steps = append(steps, workflow.Step{
			Name: fmt.Sprintf("update startram reminder for %s", pier),
			Run: func() error {
				return startramRuntime.UpdateUrbitFeatureConfigFn(pier, func(featureConf *structs.UrbitFeatureConfig) error {
					featureConf.StartramReminder = remind
					return nil
				})
			},
		})
	}
	if joined := workflow.Join(steps, func(err error) {
		zap.L().Error(err.Error())
	}); joined != nil {
		return joined
	}
	return nil
}

func HandleStartramSetBackupPassword(password string) error {
	return runStartramSetBackupPasswordWithRuntime(defaultStartramRuntime(), password)
}

func runStartramSetBackupPasswordWithRuntime(runtime startramRuntime, password string) error {
	runtime = withDefaultsStartramRuntime(runtime)
	startramRuntime := runtime.startramRuntimeFn()
	err := startramRuntime.UpdateConfTypedFn(config.WithRemoteBackupPassword(password))
	if err != nil {
		return fmt.Errorf("set backup password: %w", err)
	}
	return nil
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
