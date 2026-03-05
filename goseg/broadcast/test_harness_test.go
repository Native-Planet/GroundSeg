package broadcast

import "testing"

func withIsolatedBroadcastDefaults(t *testing.T) *broadcastStateRuntime {
	t.Helper()

	originalStateRuntime := DefaultBroadcastStateRuntime()
	originalLoopController := defaultLoopController()

	isolatedStateRuntime := SetDefaultBroadcastStateRuntime(NewBroadcastStateRuntime())
	isolatedLoopController := SetDefaultBroadcastLoopController(newBroadcastLoopController())

	t.Cleanup(func() {
		StopBroadcastLoopWithRuntime(&broadcastLoopRuntime{
			stateRuntime:   isolatedStateRuntime,
			loopController: isolatedLoopController,
		})
		SetDefaultBroadcastStateRuntime(originalStateRuntime)
		SetDefaultBroadcastLoopController(originalLoopController)
	})

	return isolatedStateRuntime
}
