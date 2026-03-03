package shipworkflow

import (
	"fmt"
	"time"

	"groundseg/config"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/internal/workflow"
	"groundseg/structs"
	"groundseg/transition"
	"groundseg/startram"

	"go.uber.org/zap"
)

func HandleStartramRestart() error {
	return runStartramRestartWithRuntime(defaultStartramRuntime())
}

func runStartramRestartWithRuntime(runtime startramRuntime) error {
	runtime = resolveStartramRuntime(runtime)
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
					return runtime.RecoverWireguardFleetFn(settings.Piers, true)
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
	runtime = resolveStartramRuntime(runtime)
	settings := runtime.StartramSettingsSnapshotFn()
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
									return func() error { return runtime.DispatchUrbitPayloadFn(payload) }
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
				if joinedErr := runOrchestrationSteps(steps, "startram toggle"); joinedErr != nil {
					return joinedErr
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
	runtime = resolveStartramRuntime(runtime)
	settings := runtime.StartramSettingsSnapshotFn()
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
	return runStartramCancelWithRuntime(defaultStartramRuntime(), key, reset)
}

func runStartramCancelWithRuntime(runtime startramRuntime, key string, reset bool) error {
	runtime = resolveStartramRuntime(runtime)
	return runStartramTransitionTemplate(runtime, startramTransitionTemplate{
		transitionType: transition.StartramTransitionCancel,
		clearEvent:     startramEvent(transition.StartramTransitionCancel, nil),
		clearDelay:     3 * time.Second,
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
	return runStartramReminderWithRuntime(defaultStartramRuntime(), remind)
}

func runStartramReminderWithRuntime(runtime startramRuntime, remind bool) error {
	runtime = resolveStartramRuntime(runtime)
	ships := runtime.ShipSettingsSnapshotFn()
	var steps []workflow.Step
	for _, patp := range ships.Piers {
		pier := patp
		steps = append(steps, workflow.Step{
			Name: fmt.Sprintf("update startram reminder for %s", pier),
			Run: func() error {
						return runtime.UpdateUrbitSectionFn(
							pier,
							dockerOrchestration.UrbitConfigSectionFeature,
							func(featureConf *structs.UrbitFeatureConfig) error {
						featureConf.StartramReminder = remind
						return nil
					})
			},
		})
	}
	if joined := runOrchestrationSteps(steps, "startram reminder"); joined != nil {
		return joined
	}
	return nil
}
