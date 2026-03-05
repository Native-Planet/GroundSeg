package shipworkflow

import (
	"fmt"
	"time"

	"groundseg/config"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"

	"go.uber.org/zap"
)

func runStartramRegisterWithRuntime(runtime startramRuntime, regCode, region string) error {
	runtime = resolveStartramRuntime(runtime)
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

func runStartramSetBackupPasswordWithRuntime(runtime startramRuntime, password string) error {
	runtime = resolveStartramRuntime(runtime)
	err := runtime.UpdateConfigTypedFn(config.WithRemoteBackupPassword(password))
	if err != nil {
		return fmt.Errorf("set backup password: %w", err)
	}
	return nil
}
