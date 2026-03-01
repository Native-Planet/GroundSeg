package broadcast

import (
	"errors"
	"testing"

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
		rt.getClientManager = func() *structs.ClientManager {
			return auth.NewClientManager()
		}
		rt.getLickStatuses = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{}
		}
	})
	constructed := false
	runtime.constructSystemInfo = func() structs.System {
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
	UpdateBroadcast(oldState)

	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManager = func() *structs.ClientManager {
			return auth.NewClientManager()
		}
		rt.getLickStatuses = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.constructSystemInfo = func() structs.System {
			system := structs.System{}
			system.Info.Usage.CPU = 42
			return system
		}
		rt.constructPierInfo = func() (map[string]structs.Urbit, error) {
			return map[string]structs.Urbit{"zod": {}}, nil
		}
		rt.constructAppsInfo = func() structs.Apps {
			apps := structs.Apps{}
			apps.Penpai.Info.ActiveModel = "llama"
			return apps
		}
		rt.constructProfileInfo = func() structs.Profile {
			profile := structs.Profile{}
			profile.Startram.Info.Endpoint = "api.example.com"
			return profile
		}
	})
	var updated structs.AuthBroadcast
	runtime.updateBroadcast = func(next structs.AuthBroadcast) {
		updated = next
		UpdateBroadcast(next)
	}
	broadcastCalls := 0
	runtime.broadcastToClients = func() error {
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
	UpdateBroadcast(structs.AuthBroadcast{
		Urbits: map[string]structs.Urbit{"zod": {}},
	})
	runtime := newTestBroadcastLoopRuntime(func(rt *broadcastLoopRuntime) {
		rt.getClientManager = func() *structs.ClientManager {
			return auth.NewClientManager()
		}
		rt.getLickStatuses = func() map[string]leak.LickStatus {
			return map[string]leak.LickStatus{"zod": {}}
		}
		rt.constructSystemInfo = func() structs.System { return structs.System{} }
		rt.constructPierInfo = func() (map[string]structs.Urbit, error) {
			return nil, errors.New("failed")
		}
		rt.constructAppsInfo = func() structs.Apps { return structs.Apps{} }
		rt.constructProfileInfo = func() structs.Profile { return structs.Profile{} }
		rt.broadcastToClients = func() error { return nil }
	})
	runtime.updateBroadcast = func(next structs.AuthBroadcast) {
		UpdateBroadcast(next)
	}

	// Should not panic even when pier info builder returns error and nil map.
	runBroadcastTickWithRuntime(runtime)
}
