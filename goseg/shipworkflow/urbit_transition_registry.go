package shipworkflow

import (
	"fmt"
	"time"

	"groundseg/internal/transitionlifecycle"
	"groundseg/structs"
	"groundseg/transition"
)

type urbitTransitionCommandDescriptor struct {
	template urbitTransitionTemplate
	stepsFn  func(patp string, payload structs.WsUrbitPayload) []transitionStep[string]
	reducer  transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition]
}

type urbitTransitionRunner func(string, structs.WsUrbitPayload) error

var urbitTransitionCommandDescriptors = map[transition.UrbitTransitionType]urbitTransitionCommandDescriptor{
	transition.UrbitTransitionUrbitDomain: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionUrbitDomain),
			startEvent:     "loading",
			successEvent:   "done",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildUrbitDomainSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.UrbitDomain = value
		}),
	},
	transition.UrbitTransitionMinIODomain: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionMinIODomain),
			startEvent:     "loading",
			successEvent:   "done",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildMinIODomainSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.MinIODomain = value
		}),
	},
	transition.UrbitTransitionChopOnUpgrade: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionChopOnUpgrade),
			startEvent:     "loading",
			clearEvent:     "",
			clearDelay:     3 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildToggleChopOnVereUpdateSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.ChopOnUpgrade = value
		}),
	},
	transition.UrbitTransitionTogglePower: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionTogglePower),
			startEvent:     "loading",
			clearEvent:     "",
			clearDelay:     0,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildTogglePowerSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.TogglePower = value
		}),
	},
	transition.UrbitTransitionToggleDevMode: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionToggleDevMode),
			startEvent:     "loading",
			clearEvent:     "",
			clearDelay:     0,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildToggleDevModeSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.ToggleDevMode = value
		}),
	},
	transition.UrbitTransitionToggleNetwork: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionToggleNetwork),
			startEvent:     "loading",
			clearEvent:     "",
			clearDelay:     0,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildToggleNetworkSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.ToggleNetwork = value
		}),
	},
	transition.UrbitTransitionRebuildContainer: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionRebuildContainer),
			startEvent:     "loading",
			successEvent:   "success",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     3 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildRebuildContainerSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.RebuildContainer = value
		}),
	},
	transition.UrbitTransitionDeleteShip: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionDeleteShip),
			startEvent:     "stopping",
			successEvent:   "done",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     1 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildDeleteShipSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.DeleteShip = value
		}),
	},
	transition.UrbitTransitionExportShip: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionExportShip),
			startEvent:     "stopping",
			successEvent:   "ready",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     1 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildExportShipSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.ExportShip = value
		}),
	},
	transition.UrbitTransitionExportBucket: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionExportBucket),
			successEvent:   "ready",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     1 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildExportBucketSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.ExportBucket = value
		}),
	},
	transition.UrbitTransitionToggleMinIOLink: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionToggleMinIOLink),
			startEvent:     "loading",
			clearEvent:     "",
			clearDelay:     1 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildToggleMinIOLinkSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.ToggleMinIOLink = value
		}),
	},
	transition.UrbitTransitionLoom: {
		template: urbitTransitionTemplate{
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
		stepsFn: buildHandleLoomSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.Loom = value
		}),
	},
	transition.UrbitTransitionSnapTime: {
		template: urbitTransitionTemplate{
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
		stepsFn: buildHandleSnapTimeSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.SnapTime = value
		}),
	},
	transition.UrbitTransitionPack: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionPack),
			startEvent:     "packing",
			successEvent:   "success",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     3 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildPackSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.Pack = value
		}),
	},
	transition.UrbitTransitionPackMeld: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionPackMeld),
			startEvent:     "packing",
			successEvent:   "success",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     3 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildPackMeldSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.PackMeld = value
		}),
	},
	transition.UrbitTransitionRollChop: {
		template: urbitTransitionTemplate{
			transitionType: string(transition.UrbitTransitionRollChop),
			startEvent:     "rolling",
			successEvent:   "success",
			emitSuccess:    true,
			clearEvent:     "",
			clearDelay:     3 * time.Second,
			err: func(error) string {
				return "error"
			},
		},
		stepsFn: buildRollChopSteps,
		reducer: urbitTransitionStringReducer(func(state *structs.UrbitTransitionBroadcast, value string) {
			state.RollChop = value
		}),
	},
}

func runUrbitTransitionFromCommandRegistry(patp string, transitionType transition.UrbitTransitionType, payload structs.WsUrbitPayload) error {
	spec, ok := urbitTransitionCommandDescriptors[transitionType]
	if !ok {
		return fmt.Errorf("unrecognized urbit transition: %s", transitionType)
	}
	return runUrbitTransitionTemplateFn(patp, spec.template, spec.stepsFn(patp, payload)...)
}

func UrbitTransitionRunners() map[transition.UrbitTransitionType]urbitTransitionRunner {
	runners := make(map[transition.UrbitTransitionType]urbitTransitionRunner, len(urbitTransitionCommandDescriptors))
	for transitionType := range urbitTransitionCommandDescriptors {
		transitionType := transitionType
		runners[transitionType] = func(patp string, payload structs.WsUrbitPayload) error {
			return runUrbitTransitionFromCommandRegistry(patp, transitionType, payload)
		}
	}
	return runners
}

func UrbitTransitionReducerMap() map[transition.UrbitTransitionType]transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition] {
	reducers := make(map[transition.UrbitTransitionType]transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition], len(urbitTransitionCommandDescriptors))
	for transitionType, descriptor := range urbitTransitionCommandDescriptors {
		reducers[transitionType] = descriptor.reducer
	}
	return reducers
}

func urbitTransitionStringReducer(
	setter func(*structs.UrbitTransitionBroadcast, string),
) transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition] {
	return func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		setter(state, event.Event)
		return true
	}
}

func urbitTransitionIntReducer(
	setter func(*structs.UrbitTransitionBroadcast, int),
) transitionlifecycle.Reducer[transition.UrbitTransitionType, structs.UrbitTransitionBroadcast, structs.UrbitTransition] {
	return func(state *structs.UrbitTransitionBroadcast, event structs.UrbitTransition) bool {
		setter(state, event.Value)
		return true
	}
}
