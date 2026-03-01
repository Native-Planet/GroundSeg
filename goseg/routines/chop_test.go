package routines

import (
	"testing"
	"time"

	"groundseg/structs"
)

func resetChopSeamsForTest(t *testing.T) {
	t.Helper()
	originalConf := confForChop
	originalUrbitConf := urbitConfForChop
	originalStats := getContainerStatsForChop
	originalChopPier := chopPierForChop
	originalSleep := sleepForChop

	t.Cleanup(func() {
		confForChop = originalConf
		urbitConfForChop = originalUrbitConf
		getContainerStatsForChop = originalStats
		chopPierForChop = originalChopPier
		sleepForChop = originalSleep
	})
}

func TestRunChopAtLimitPassSkipsShipsWithoutSizeLimit(t *testing.T) {
	resetChopSeamsForTest(t)

	confForChop = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"~zod"}}
	}
	urbitConfForChop = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{SizeLimit: 0}
	}
	getContainerStatsForChop = func(string) structs.ContainerStats {
		return structs.ContainerStats{DiskUsage: 100 * bytesPerGiB}
	}

	called := false
	chopPierForChop = func(string, structs.UrbitDocker) error {
		called = true
		return nil
	}

	runChopAtLimitPass()
	if called {
		t.Fatal("expected chopPier not to be called when size limit is zero")
	}
}

func TestRunChopAtLimitPassTriggersChopAtOrAboveLimit(t *testing.T) {
	resetChopSeamsForTest(t)

	confForChop = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"~zod", "~bus"}}
	}
	urbitConfForChop = func(patp string) structs.UrbitDocker {
		switch patp {
		case "~zod":
			return structs.UrbitDocker{SizeLimit: 5}
		case "~bus":
			return structs.UrbitDocker{SizeLimit: 6}
		default:
			return structs.UrbitDocker{}
		}
	}
	getContainerStatsForChop = func(patp string) structs.ContainerStats {
		switch patp {
		case "~zod":
			return structs.ContainerStats{DiskUsage: 5 * bytesPerGiB}
		case "~bus":
			return structs.ContainerStats{DiskUsage: 5 * bytesPerGiB}
		default:
			return structs.ContainerStats{}
		}
	}

	calls := make(chan string, 2)
	chopPierForChop = func(patp string, _ structs.UrbitDocker) error {
		calls <- patp
		return nil
	}

	runChopAtLimitPass()

	select {
	case patp := <-calls:
		if patp != "~zod" {
			t.Fatalf("unexpected chop call for %s", patp)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected chop call for ship at size limit")
	}

	select {
	case extra := <-calls:
		t.Fatalf("unexpected extra chop call for %s", extra)
	case <-time.After(100 * time.Millisecond):
	}
}
