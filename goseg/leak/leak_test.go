package leak

import (
	"errors"
	"net"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"groundseg/structs"
)

func resetLeakSeamsForTest(t *testing.T) {
	t.Helper()
	originalNow := leakNow
	originalSleep := leakSleep
	originalConf := leakConf
	originalUrbitConf := leakUrbitConf
	originalLookupEnv := leakLookupEnv
	originalStat := leakStat
	originalMakeSymlink := leakMakeSymlink
	originalMakeConnection := leakMakeConnection
	originalListener := leakListener
	originalUpdateBroadcast := leakUpdateBroadcast

	t.Cleanup(func() {
		leakNow = originalNow
		leakSleep = originalSleep
		leakConf = originalConf
		leakUrbitConf = originalUrbitConf
		leakLookupEnv = originalLookupEnv
		leakStat = originalStat
		leakMakeSymlink = originalMakeSymlink
		leakMakeConnection = originalMakeConnection
		leakListener = originalListener
		leakUpdateBroadcast = originalUpdateBroadcast
	})
}

func TestGetLickStatusesReflectsCurrentState(t *testing.T) {
	resetLeakStateForTest(t)
	setLickStatusesForTest(map[string]LickStatus{
		"~zod": {Symlink: "/tmp/zod", Auth: true},
	})

	statuses := GetLickStatuses()
	want := LickStatus{Symlink: "/tmp/zod", Auth: true}
	if got, exists := statuses["~zod"]; !exists || got != want {
		t.Fatalf("unexpected status map content: %+v", statuses)
	}
}

func TestProcessLeakBroadcastSkipsUpdatesWithinThrottleWindow(t *testing.T) {
	resetLeakSeamsForTest(t)

	base := time.Unix(1_700_000_000, 0)
	leakNow = func() time.Time { return base.Add(500 * time.Millisecond) }
	called := false
	leakUpdateBroadcast = func(oldBroadcast, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, error) {
		called = true
		return newBroadcast, nil
	}

	old := structs.AuthBroadcast{Type: "old", AuthLevel: "old"}
	incoming := structs.AuthBroadcast{Type: "incoming", AuthLevel: "incoming"}
	updated, updatedLastRcv, err := processLeakBroadcast(old, base, incoming)
	if err != nil {
		t.Fatalf("processLeakBroadcast returned error: %v", err)
	}
	if called {
		t.Fatal("expected update function not to run during throttle window")
	}
	if !reflect.DeepEqual(updated, old) {
		t.Fatalf("expected old broadcast unchanged, got %+v", updated)
	}
	if !updatedLastRcv.Equal(base) {
		t.Fatalf("expected last receive time to stay unchanged, got %v", updatedLastRcv)
	}
}

func TestProcessLeakBroadcastAppliesStructureMetadataAndUpdates(t *testing.T) {
	resetLeakSeamsForTest(t)

	base := time.Unix(1_700_000_000, 0)
	now := base.Add(2 * time.Second)
	leakNow = func() time.Time { return now }
	leakUpdateBroadcast = func(oldBroadcast, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, error) {
		if newBroadcast.Type != "structure" {
			t.Fatalf("expected type to be normalized to structure, got %q", newBroadcast.Type)
		}
		if newBroadcast.AuthLevel != "authorized" {
			t.Fatalf("expected auth level to be authorized, got %q", newBroadcast.AuthLevel)
		}
		newBroadcast.Profile.Startram.Info.Running = true
		return newBroadcast, nil
	}

	updated, updatedLastRcv, err := processLeakBroadcast(
		structs.AuthBroadcast{},
		base,
		structs.AuthBroadcast{Type: "ignored", AuthLevel: "ignored"},
	)
	if err != nil {
		t.Fatalf("processLeakBroadcast returned error: %v", err)
	}
	if !updated.Profile.Startram.Info.Running {
		t.Fatalf("expected updated payload from update function, got %+v", updated)
	}
	if !updatedLastRcv.Equal(now) {
		t.Fatalf("expected last receive time to update to now, got %v", updatedLastRcv)
	}
}

func TestProcessLeakBroadcastReturnsErrorFromUpdate(t *testing.T) {
	resetLeakSeamsForTest(t)

	base := time.Unix(1_700_000_000, 0)
	now := base.Add(2 * time.Second)
	leakNow = func() time.Time { return now }
	leakUpdateBroadcast = func(oldBroadcast, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, error) {
		return structs.AuthBroadcast{}, errors.New("update failed")
	}

	old := structs.AuthBroadcast{Type: "old"}
	updated, updatedLastRcv, err := processLeakBroadcast(old, base, structs.AuthBroadcast{})
	if err == nil {
		t.Fatal("expected update error")
	}
	if !reflect.DeepEqual(updated, old) {
		t.Fatalf("expected old broadcast to be returned on error, got %+v", updated)
	}
	if !updatedLastRcv.Equal(now) {
		t.Fatalf("expected last receive time to move forward, got %v", updatedLastRcv)
	}
}

func TestConnectDevSocketNoopsWhenConnectionFails(t *testing.T) {
	resetLeakSeamsForTest(t)

	calledPath := ""
	listenerCalled := false
	leakMakeConnection = func(sockLocation string) net.Conn {
		calledPath = sockLocation
		return nil
	}
	leakListener = func(patp string, conn net.Conn, info LickStatus) {
		listenerCalled = true
	}

	connectDevSocket("/tmp/devsock")
	if calledPath != filepath.Join("/tmp/devsock", "groundseg.sock") {
		t.Fatalf("unexpected dev socket path: %q", calledPath)
	}
	if listenerCalled {
		t.Fatal("listener should not be called when connection is nil")
	}
}

func TestConnectDevSocketStartsDevListenerWhenConnectionSucceeds(t *testing.T) {
	resetLeakSeamsForTest(t)

	server, client := net.Pipe()
	defer server.Close()

	called := make(chan struct{}, 1)
	leakMakeConnection = func(sockLocation string) net.Conn {
		if sockLocation != filepath.Join("/tmp/devsock", "groundseg.sock") {
			t.Fatalf("unexpected dev socket path: %q", sockLocation)
		}
		return client
	}
	leakListener = func(patp string, conn net.Conn, info LickStatus) {
		if patp != "dev" {
			t.Fatalf("unexpected listener patp: %q", patp)
		}
		if !info.Auth {
			t.Fatal("expected dev listener to be authorized")
		}
		if info.Symlink != "/tmp/devsock" {
			t.Fatalf("unexpected symlink: %q", info.Symlink)
		}
		conn.Close()
		called <- struct{}{}
	}

	connectDevSocket("/tmp/devsock")

	select {
	case <-called:
	case <-time.After(2 * time.Second):
		t.Fatal("dev listener was not invoked")
	}
}
