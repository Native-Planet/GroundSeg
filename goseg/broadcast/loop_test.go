package broadcast

import (
	"errors"
	"strings"
	"testing"
	"time"

	"groundseg/leak"
	"groundseg/session"
	"groundseg/structs"
)

type countingInfoCollector struct {
	systemCalled *bool
}

func (collector countingInfoCollector) CollectAppsInfo() structs.Apps { return structs.Apps{} }

func (collector countingInfoCollector) CollectProfileInfo(map[string]structs.StartramRegion) structs.Profile {
	return structs.Profile{}
}

func (collector countingInfoCollector) CollectSystemInfo() structs.System {
	if collector.systemCalled != nil {
		*collector.systemCalled = true
	}
	return structs.System{}
}

func TestPreserveTransitionHelpers(t *testing.T) {
	oldState := structs.AuthBroadcast{
		System: structs.System{
			Transition: structs.SystemTransitionBroadcast{
				WifiConnect: "connected",
			},
		},
		Profile: structs.Profile{
			Startram: structs.Startram{
				Transition: structs.StartramTransition{
					Endpoint: "updating",
				},
			},
		},
		Urbits: map[string]structs.Urbit{
			"zod": {
				Transition: structs.UrbitTransitionBroadcast{
					Pack: "running",
				},
			},
		},
	}
	newSystem := PreserveSystemTransitions(oldState, structs.System{})
	if newSystem.Transition.WifiConnect != "connected" {
		t.Fatalf("expected preserved system transition, got %+v", newSystem.Transition)
	}
	newProfile := PreserveProfileTransitions(oldState, structs.Profile{})
	if newProfile.Startram.Transition.Endpoint != "updating" {
		t.Fatalf("expected preserved profile transition, got %+v", newProfile.Startram.Transition)
	}
	newUrbits := PreserveUrbitsTransitions(oldState, map[string]structs.Urbit{})
	if newUrbits["zod"].Transition.Pack != "running" {
		t.Fatalf("expected preserved urbit transition, got %+v", newUrbits["zod"].Transition)
	}
}

func TestRunBroadcastTickSkipsWhenNoSessionsOrLeaks(t *testing.T) {
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return session.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{}
		}
	})
	constructed := false
	runtime.stateRuntime.infoCollector = countingInfoCollector{systemCalled: &constructed}

	runBroadcastTickWithRuntime(runtime)
	if constructed {
		t.Fatal("expected tick to skip expensive construction when no observers are connected")
	}
}

func TestRunBroadcastTickWithRuntimeRequiresRuntimeSentinel(t *testing.T) {
	if err := runBroadcastTickWithRuntime(nil); err == nil {
		t.Fatal("expected missing runtime error")
	} else if !errors.Is(err, ErrBroadcastRuntimeRequired) {
		t.Fatalf("expected ErrBroadcastRuntimeRequired, got %v", err)
	}
}

func TestRunBroadcastTickBuildsStateAndPreservesTransitions(t *testing.T) {
	withIsolatedBroadcastDefaults(t)

	oldState := structs.AuthBroadcast{
		System: structs.System{
			Transition: structs.SystemTransitionBroadcast{BugReport: "loading"},
		},
		Profile: structs.Profile{
			Startram: structs.Startram{
				Transition: structs.StartramTransition{Restart: "running"},
			},
		},
		Urbits: map[string]structs.Urbit{
			"zod": {
				Transition: structs.UrbitTransitionBroadcast{Pack: "packing"},
			},
		},
	}
	DefaultBroadcastStateRuntime().UpdateBroadcast(oldState)

	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return session.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		system := structs.System{}
		system.Info.Usage.CPU = 42
		apps := structs.Apps{}
		apps.Penpai.Info.ActiveModel = "llama"
		profile := structs.Profile{}
		profile.Startram.Info.Endpoint = "api.example.com"
		rt.stateRuntime.infoCollector = testInfoCollector{
			system:  system,
			apps:    apps,
			profile: profile,
		}
		rt.stateRuntime.pierCollector = testPierCollector{
			urbits: map[string]structs.Urbit{"zod": {}},
		}
	})
	var updated structs.AuthBroadcast
	runtime.updateBroadcastFn = func(next structs.AuthBroadcast) error {
		updated = next
		return DefaultBroadcastStateRuntime().UpdateBroadcast(next)
	}
	broadcastCalls := 0
	runtime.broadcastToClientsFn = func() error {
		broadcastCalls++
		return nil
	}

	runBroadcastTickWithRuntime(runtime)

	if updated.System.Info.Usage.CPU != 42 {
		t.Fatalf("expected reconstructed system info, got %+v", updated.System.Info.Usage)
	}
	if updated.System.Transition.BugReport != "loading" {
		t.Fatalf("expected preserved system transition, got %+v", updated.System.Transition)
	}
	if updated.Urbits["zod"].Transition.Pack != "packing" {
		t.Fatalf("expected preserved urbit transition, got %+v", updated.Urbits["zod"].Transition)
	}
	if updated.Profile.Startram.Transition.Restart != "running" {
		t.Fatalf("expected preserved profile transition, got %+v", updated.Profile.Startram.Transition)
	}
	if updated.Profile.Startram.Info.Endpoint != "api.example.com" {
		t.Fatalf("expected updated profile info, got %+v", updated.Profile.Startram.Info)
	}
	if updated.Apps.Penpai.Info.ActiveModel != "llama" {
		t.Fatalf("expected updated apps info, got %+v", updated.Apps.Penpai.Info)
	}
	if broadcastCalls != 1 {
		t.Fatalf("expected one broadcast call, got %d", broadcastCalls)
	}
}

func TestRunBroadcastTickHandlesPierInfoError(t *testing.T) {
	withIsolatedBroadcastDefaults(t)

	initialState := structs.AuthBroadcast{
		System: structs.System{Transition: structs.SystemTransitionBroadcast{Error: []string{"old-error"}}},
		Urbits: map[string]structs.Urbit{
			"zod": {
				Transition: structs.UrbitTransitionBroadcast{Pack: "packing"},
			},
		},
	}
	DefaultBroadcastStateRuntime().UpdateBroadcast(initialState)
	t.Cleanup(func() {
		DefaultBroadcastStateRuntime().UpdateBroadcast(structs.AuthBroadcast{})
	})

	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return session.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.stateRuntime.infoCollector = testInfoCollector{}
		rt.stateRuntime.pierCollector = testPierCollector{
			err: errors.New("failed"),
		}
		rt.broadcastToClientsFn = func() error { return nil }
	})

	gotErr := runBroadcastTickWithRuntime(runtime)
	if gotErr == nil {
		t.Fatal("expected pier info collector error")
	}

	state := DefaultBroadcastStateRuntime().GetState()
	if got := state.Urbits["zod"]; got.Transition.Pack != "packing" {
		t.Fatalf("expected prior urbit transition to remain, got %+v", got.Transition)
	}
	if got := state.System.Transition.Error; len(got) == 0 || got[0] != gotErr.Error() {
		t.Fatalf("expected system transition error to be recorded, got %+v", got)
	}
}

func TestRunBroadcastTickRecordsPierInfoError(t *testing.T) {
	withIsolatedBroadcastDefaults(t)

	collectorErr := errors.New("collector failure")

	originalState := DefaultBroadcastStateRuntime().GetState()
	t.Cleanup(func() {
		DefaultBroadcastStateRuntime().UpdateBroadcast(originalState)
	})

	DefaultBroadcastStateRuntime().UpdateBroadcast(structs.AuthBroadcast{
		System: structs.System{
			Transition: structs.SystemTransitionBroadcast{
				Error: []string{},
			},
		},
	})

	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return session.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.stateRuntime.infoCollector = testInfoCollector{}
		rt.stateRuntime.pierCollector = testPierCollector{
			err: collectorErr,
		}
		rt.broadcastToClientsFn = func() error { return nil }
	})

	gotErr := runBroadcastTickWithRuntime(runtime)
	if gotErr == nil {
		t.Fatal("expected collector error to propagate")
	}
	if !errors.Is(gotErr, collectorErr) {
		t.Fatalf("unexpected tick error: %v", gotErr)
	}

	state := DefaultBroadcastStateRuntime().GetState()
	if len(state.System.Transition.Error) == 0 || state.System.Transition.Error[0] != gotErr.Error() {
		t.Fatalf("expected system transition error to be recorded, got %+v", state.System.Transition.Error)
	}
}

func TestRunBroadcastTickReturnsBroadcastError(t *testing.T) {
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return session.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.stateRuntime.infoCollector = testInfoCollector{}
		rt.stateRuntime.pierCollector = testPierCollector{
			urbits: map[string]structs.Urbit{},
		}
	})
	expectedErr := errors.New("broadcast transport failure")
	runtime.broadcastToClientsFn = func() error { return expectedErr }

	if got := runBroadcastTickWithRuntime(runtime); !errors.Is(got, expectedErr) {
		t.Fatalf("expected broadcast error propagation, got %v", got)
	}
}

func TestRunBroadcastLoopReportsTickErrors(t *testing.T) {
	t.Parallel()

	eventCh := make(chan error, 1)
	stopCh := make(chan struct{})
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return session.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.stateRuntime.infoCollector = testInfoCollector{}
		rt.stateRuntime.pierCollector = testPierCollector{
			urbits: map[string]structs.Urbit{},
		}
		rt.broadcastToClientsFn = func() error { return errors.New("broadcast transport failure") }
		rt.tickErrorFn = func(err error) {
			eventCh <- err
		}
		rt.tickInterval = 10 * time.Millisecond
	})

	go runBroadcastLoop(runtime, stopCh)

	defer close(stopCh)
	var gotErr error
	select {
	case gotErr = <-eventCh:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected tick error to be reported")
	}
	if gotErr == nil || !strings.Contains(gotErr.Error(), "broadcast transport failure") {
		t.Fatalf("unexpected tick error %v", gotErr)
	}
}

func TestStartBroadcastLoopGuardsDuplicateStarts(t *testing.T) {
	withIsolatedBroadcastDefaults(t)

	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return nil
		}
	})

	StopBroadcastLoop()
	if err := StartBroadcastLoopWithRuntime(runtime); err != nil {
		t.Fatalf("expected initial broadcast loop start to succeed: %v", err)
	}
	if err := StartBroadcastLoopWithRuntime(runtime); !errors.Is(err, errBroadcastLoopAlreadyRunning) {
		t.Fatalf("expected duplicate broadcast loop start to be rejected, got %v", err)
	}

	StopBroadcastLoop()
}

func TestStopBroadcastLoopWithRuntimeStopsCustomController(t *testing.T) {
	withIsolatedBroadcastDefaults(t)

	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager { return nil }
		rt.loopController = newBroadcastLoopController()
	})

	if err := StartBroadcastLoopWithRuntime(runtime); err != nil {
		t.Fatalf("expected custom runtime loop start to succeed: %v", err)
	}
	StopBroadcastLoopWithRuntime(runtime)
	if err := StartBroadcastLoopWithRuntime(runtime); err != nil {
		t.Fatalf("expected loop to restart after runtime-scoped stop: %v", err)
	}
	StopBroadcastLoopWithRuntime(runtime)
}
