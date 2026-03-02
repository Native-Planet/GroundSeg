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

var (
	dispatchStartramToggleNetworkFn = ToggleNetwork
	startramDispatchUrbitPayloadFn  = defaultDispatchUrbitPayload
	startramRuntimeFn               = func() orchestration.StartramRuntime {
		return orchestration.NewStartramRuntime(
			orchestration.WithStartramServiceLoaders(
				broadcast.GetStartramServices,
				broadcast.LoadStartramRegions,
			),
		)
	}
	startramRecoverWireguardFleetFn = func(piers []string, deleteMinioClient bool) error {
		rt := orchestration.NewRuntime()
		return RecoverWireguardFleet(NewWireguardRecoveryRuntime(rt), piers, deleteMinioClient)
	}
)

func defaultDispatchUrbitPayload(payload structs.WsUrbitPayload) error {
	if payload.Payload.Action == "" {
		return fmt.Errorf("no urbit action provided")
	}
	if payload.Payload.Patp == "" {
		return fmt.Errorf("no patp provided for urbit action %q", payload.Payload.Action)
	}

	switch payload.Payload.Action {
	case "toggle-network":
		return dispatchStartramToggleNetworkFn(payload.Payload.Patp)
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

func runStartramTransitionWithRuntime(transitionType transition.EventType, plan transitionPlan[structs.Event], steps ...transitionStep[structs.Event]) error {
	if plan.ErrorEvent == nil {
		plan.ErrorEvent = func(err error) structs.Event {
			return startramEvent(transitionType, fmt.Sprintf("Error: %v", err))
		}
	}
	return runTransitionLifecycleWithRuntime[structs.Event](
		defaultWorkflowRuntime(),
		func(event structs.Event) {
			startram.PublishEvent(event)
		},
		plan,
		steps...,
	)
}

func HandleStartramServices() error {
	return startramRuntimeFn().GetStartramServicesFn()
}

func HandleStartramRegions() error {
	return startramRuntimeFn().LoadStartramRegionsFn()
}

func HandleStartramRestart() error {
	runtime := startramRuntimeFn()
	zap.L().Info("Restarting StarTram")
	settings := runtime.StartramSettingsSnapshotFn()
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
					return startramRecoverWireguardFleetFn(settings.Piers, true)
				},
			},
		)
	}
	return runStartramTransitionWithRuntime(
		transition.StartramTransitionRestart,
		transitionPlan[structs.Event]{
			EmitStart:    true,
			StartEvent:   startramEvent(transition.StartramTransitionRestart, "startram"),
			SuccessEvent: startramEvent(transition.StartramTransitionRestart, string(transition.StartramTransitionDone)),
			EmitSuccess:  true,
			ClearEvent:   startramEvent(transition.StartramTransitionRestart, ""),
			ClearDelay:   3 * time.Second,
		},
		steps...,
	)
}

func HandleStartramToggle() error {
	runtime := startramRuntimeFn()
	settings := runtime.StartramSettingsSnapshotFn()
	return runStartramTransitionWithRuntime(
		transition.StartramTransitionToggle,
		transitionPlan[structs.Event]{
			EmitStart:  true,
			StartEvent: startramEvent(transition.StartramTransitionToggle, transition.StartramTransitionLoading),
			ClearEvent: startramEvent(transition.StartramTransitionToggle, nil),
			ClearDelay: 3 * time.Second,
		},
		transitionStep[structs.Event]{
			Run: func() error {
				steps := []workflow.Step{}
				if settings.WgOn {
					steps = append(steps, workflow.Step{
						Name: "update config to disable wireguard",
						Run: func() error {
							return runtime.UpdateConfTypedFn(config.WithWgOn(false))
						},
					})
					steps = append(steps, workflow.Step{
						Name: "stop wireguard container",
						Run: func() error {
							return runtime.StopContainerByNameFn("wireguard")
						},
					})
					// toggle ships back to local
					for _, patp := range settings.Piers {
						dockerConfig := runtime.UrbitConfFn(patp)
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
									return func() error { return startramDispatchUrbitPayloadFn(payload) }
								}(payload),
							})
						}
					}
				} else {
					steps = append(steps, workflow.Step{
						Name: "update config to enable wireguard",
						Run: func() error {
							return runtime.UpdateConfTypedFn(config.WithWgOn(true))
						},
					})
					steps = append(steps, workflow.Step{
						Name: "start wireguard container",
						Run: func() error {
							_, err := runtime.StartContainerFn("wireguard", "wireguard")
							return err
						},
					})
				}
				// delete mc
				steps = appendShipContainerRebuildSteps(steps, shipContainerRebuildRuntime{
					DeleteContainerFn: runtime.DeleteContainerFn,
					LoadMCFn:          runtime.LoadMCFn,
					LoadMinIOsFn:      runtime.LoadMinIOsFn,
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
	runtime := startramRuntimeFn()
	return runStartramTransitionWithRuntime(
		transition.StartramTransitionRegister,
		transitionPlan[structs.Event]{
			EmitStart:    true,
			StartEvent:   startramEvent(transition.StartramTransitionRegister, transition.StartramTransitionLoading),
			SuccessEvent: startramEvent(transition.StartramTransitionRegister, transition.StartramTransitionComplete),
			EmitSuccess:  true,
			ClearEvent:   startramEvent(transition.StartramTransitionRegister, nil),
			ClearDelay:   3 * time.Second,
		},
		transitionStep[structs.Event]{
			Run: func() error {
				// Reset key pair
				if err := runtime.CycleWgKeyFn(); err != nil {
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
				if err := runtime.LoadWireguardFn(); err != nil {
					zap.L().Error(fmt.Sprintf("Unable to start Wireguard: %v", err))
					return fmt.Errorf("start wireguard: %w", err)
				}
				return nil
			},
		},
	)
}

func HandleStartramEndpoint(endpoint string) error {
	runtime := startramRuntimeFn()
	settings := runtime.StartramSettingsSnapshotFn()
	return runStartramTransitionWithRuntime(
		transition.StartramTransitionEndpoint,
		transitionPlan[structs.Event]{
			EmitStart:    true,
			StartEvent:   startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionInit),
			SuccessEvent: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionComplete),
			EmitSuccess:  true,
			ClearEvent:   startramEvent(transition.StartramTransitionEndpoint, nil),
			ClearDelay:   3 * time.Second,
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionStopping),
			Run: func() error {
				if !settings.WgOn {
					return nil
				}
				if err := runtime.StopContainerByNameFn("wireguard"); err != nil {
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
					if err := runtime.SvcDeleteFn(p, "urbit"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't remove urbit anchor for %v", p))
					}
					if err := runtime.SvcDeleteFn("s3."+p, "s3"); err != nil {
						zap.L().Error(fmt.Sprintf("Couldn't remove s3 anchor for %v", p))
					}
				}
				return nil
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionConfiguring),
			Run: func() error {
				return runtime.CycleWgKeyFn()
			},
		},
		transitionStep[structs.Event]{
			Event: startramEvent(transition.StartramTransitionEndpoint, transition.StartramTransitionFinalizing),
			Run: func() error {
				return runtime.UpdateConfTypedFn(
					config.WithEndpointURL(endpoint),
					config.WithWgRegistered(false),
				)
			},
		},
	)
}

func HandleStartramCancel(key string, reset bool) error {
	runtime := startramRuntimeFn()
	return runStartramTransitionWithRuntime(
		transition.StartramTransitionCancel,
		transitionPlan[structs.Event]{
			ClearEvent: startramEvent(transition.StartramTransitionCancel, nil),
			ClearDelay: 3 * time.Second,
			ErrorEvent: func(err error) structs.Event {
				return startramEvent(transition.StartramTransitionCancel, fmt.Sprintf("Error: %v", err))
			},
		},
		transitionStep[structs.Event]{
			Run: func() error {
				if reset {
					for _, svc := range runtime.GetStartramConfigFn().Subdomains {
						if err := runtime.SvcDeleteFn(svc.URL, svc.SvcType); err != nil {
							zap.L().Error(fmt.Sprintf("Couldn't delete service %v: %v", svc.URL, err))
						}
					}
				}
				if err := startram.CancelSub(key); err != nil {
					zap.L().Error(fmt.Sprintf("Couldn't cancel subscription: %v", err))
					return fmt.Errorf("cancel subscription: %w", err)
				}
				if err := runtime.CycleWgKeyFn(); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					return fmt.Errorf("cycle wireguard key: %w", err)
				}
				return nil
			},
		},
	)
}

func HandleStartramReminder(remind bool) error {
	runtime := startramRuntimeFn()
	ships := runtime.ShipSettingsSnapshotFn()
	var steps []workflow.Step
	for _, patp := range ships.Piers {
		pier := patp
		steps = append(steps, workflow.Step{
			Name: fmt.Sprintf("update startram reminder for %s", pier),
			Run: func() error {
				return runtime.UpdateUrbitFn(pier, func(shipConf *structs.UrbitDocker) error {
					shipConf.StartramReminder = remind
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
	runtime := startramRuntimeFn()
	err := runtime.UpdateConfTypedFn(config.WithRemoteBackupPassword(password))
	if err != nil {
		return fmt.Errorf("set backup password: %w", err)
	}
	return nil
}

func HandleStartramUploadBackup(patp string) error {
	return runStartramTransitionWithRuntime(
		transition.StartramTransitionUploadBackup,
		transitionPlan[structs.Event]{
			EmitStart:  true,
			StartEvent: startramEvent(transition.StartramTransitionUploadBackup, "upload"),
			ClearEvent: startramEvent(transition.StartramTransitionUploadBackup, nil),
			ClearDelay: 3 * time.Second,
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
