package chop

import (
	"testing"
	"time"

	"groundseg/structs"
)

func resetChopSeamsForTest(t *testing.T) {
	t.Helper()
	originalConf := ConfForChop
	originalUrbitConf := UrbitConfForChop
	originalStats := ContainerStatsForChop
	originalChopPier := ChopPierForChop
	originalSleep := sleepForChop

	t.Cleanup(func() {
		ConfForChop = originalConf
		UrbitConfForChop = originalUrbitConf
		ContainerStatsForChop = originalStats
		ChopPierForChop = originalChopPier
		sleepForChop = originalSleep
	})
}

func TestRunChopAtLimitPassSkipsShipsWithoutSizeLimit(t *testing.T) {
	resetChopSeamsForTest(t)

	ConfForChop = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"~zod"}}
	}
	UrbitConfForChop = func(string) structs.UrbitDocker {
		return structs.UrbitDocker{
			UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
				SizeLimit: 0,
			},
		}
	}
	ContainerStatsForChop = func(string) structs.ContainerStats {
		return structs.ContainerStats{DiskUsage: 100 * (int64(1) << 30)}
	}

	called := false
	ChopPierForChop = func(string) error {
		called = true
		return nil
	}

	RunAtLimitPass()
	if called {
		t.Fatal("expected chopPier not to be called when size limit is zero")
	}
}

func TestRunChopAtLimitPassTriggersChopAtOrAboveLimit(t *testing.T) {
	resetChopSeamsForTest(t)

	ConfForChop = func() structs.SysConfig {
		return structs.SysConfig{Piers: []string{"~zod", "~bus"}}
	}
	UrbitConfForChop = func(patp string) structs.UrbitDocker {
		switch patp {
		case "~zod":
			return structs.UrbitDocker{
				UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
					SizeLimit: 5,
				},
			}
		case "~bus":
			return structs.UrbitDocker{
				UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
					SizeLimit: 6,
				},
			}
		default:
			return structs.UrbitDocker{}
		}
	}
	ContainerStatsForChop = func(patp string) structs.ContainerStats {
		switch patp {
		case "~zod":
			return structs.ContainerStats{DiskUsage: 5 * (int64(1) << 30)}
		case "~bus":
			return structs.ContainerStats{DiskUsage: 5 * (int64(1) << 30)}
		default:
			return structs.ContainerStats{}
		}
	}

	calls := make(chan string, 2)
	ChopPierForChop = func(patp string) error {
		calls <- patp
		return nil
	}

	RunAtLimitPass()

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
