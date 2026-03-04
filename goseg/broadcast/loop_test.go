package broadcast

import (
	"errors"
	"testing"
	"time"

	"groundseg/auth"
	"groundseg/leak"
	"groundseg/structs"
)

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
			return auth.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{}
		}
	})
	constructed := false
	runtime.constructSystemInfoFn = func() structs.System {
		constructed = true
		return structs.System{}
	}

	runBroadcastTickWithRuntime(runtime)
	if constructed {
		t.Fatal("expected tick to skip expensive construction when no observers are connected")
	}
}

func TestRunBroadcastTickBuildsStateAndPreservesTransitions(t *testing.T) {
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
			return auth.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.constructSystemInfoFn = func() structs.System {
			system := structs.System{}
			system.Info.Usage.CPU = 42
			return system
		}
		rt.constructPierInfoFn = func() (map[string]structs.Urbit, error) {
			return map[string]structs.Urbit{"zod": {}}, nil
		}
		rt.constructAppsInfoFn = func() structs.Apps {
			apps := structs.Apps{}
			apps.Penpai.Info.ActiveModel = "llama"
			return apps
		}
		rt.constructProfileInfoFn = func() structs.Profile {
			profile := structs.Profile{}
			profile.Startram.Info.Endpoint = "api.example.com"
			return profile
		}
	})
	var updated structs.AuthBroadcast
	runtime.updateBroadcastFn = func(next structs.AuthBroadcast) {
		updated = next
		DefaultBroadcastStateRuntime().UpdateBroadcast(next)
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
	saved := structs.Urbit{}
	saved.Transition = structs.UrbitTransitionBroadcast{Pack: "packing"}
	saved.Info.LusCode = "0v0"

	DefaultBroadcastStateRuntime().UpdateBroadcast(structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{
			"zod": saved,
		},
	})
	var updated structs.AuthBroadcast
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return auth.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.constructSystemInfoFn = func() structs.System { return structs.System{} }
		rt.constructPierInfoFn = func() (map[string]structs.Urbit, error) {
			return nil, errors.New("failed")
		}
		rt.constructAppsInfoFn = func() structs.Apps { return structs.Apps{} }
		rt.constructProfileInfoFn = func() structs.Profile { return structs.Profile{} }
		rt.broadcastToClientsFn = func() error { return nil }
	})
	runtime.updateBroadcastFn = func(next structs.AuthBroadcast) { updated = next }

	// Should not panic even when pier info builder returns error and nil map.
	runBroadcastTickWithRuntime(runtime)

	if got := updated.Urbits["zod"]; got.Transition.Pack != "packing" {
		t.Fatalf("expected prior urbit transition to be preserved, got %+v", got.Transition)
	}
	if got := updated.Urbits["zod"]; got.Info.LusCode != "0v0" {
		t.Fatalf("expected prior urbit info to be preserved, got %+v", got.Info)
	}
}

func TestRunBroadcastTickReturnsBroadcastError(t *testing.T) {
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return auth.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.constructSystemInfoFn = func() structs.System { return structs.System{} }
		rt.constructPierInfoFn = func() (map[string]structs.Urbit, error) { return map[string]structs.Urbit{}, nil }
		rt.constructAppsInfoFn = func() structs.Apps { return structs.Apps{} }
		rt.constructProfileInfoFn = func() structs.Profile { return structs.Profile{} }
	})
	expectedErr := errors.New("broadcast transport failure")
	runtime.broadcastToClientsFn = func() error { return expectedErr }

	if got := runBroadcastTickWithRuntime(runtime); got == nil || got.Error() != expectedErr.Error() {
		t.Fatalf("expected broadcast error propagation, got %v", got)
	}
}

func TestRunBroadcastLoopReportsTickErrors(t *testing.T) {
	t.Parallel()

	eventCh := make(chan error, 1)
	stopCh := make(chan struct{})
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return auth.NewClientManager()
		}
		rt.getLickStatusesFn = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.constructSystemInfoFn = func() structs.System { return structs.System{} }
		rt.constructPierInfoFn = func() (map[string]structs.Urbit, error) {
			return map[string]structs.Urbit{}, nil
		}
		rt.constructAppsInfoFn = func() structs.Apps { return structs.Apps{} }
		rt.constructProfileInfoFn = func() structs.Profile { return structs.Profile{} }
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
	if gotErr == nil || gotErr.Error() != "broadcast transport failure" {
		t.Fatalf("unexpected tick error %v", gotErr)
	}
}

func TestStartBroadcastLoopGuardsDuplicateStarts(t *testing.T) {
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManagerFn = func() *structs.ClientManager {
			return nil
		}
	})

	StopBroadcastLoop()
	if !StartBroadcastLoopWithRuntime(runtime) {
		t.Fatal("expected initial broadcast loop start to succeed")
	}
	if StartBroadcastLoopWithRuntime(runtime) {
		t.Fatal("expected duplicate broadcast loop start to be rejected")
	}

	StopBroadcastLoop()
}
