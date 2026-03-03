package shipworkflow

import "testing"

func TestUrbitTransitionRunnerRegistryCompleteness(t *testing.T) {
	reducers := UrbitTransitionReducerMap()
	runners := UrbitTransitionRunners()
	if len(runners) != len(reducers) {
		t.Fatalf("runner registry length mismatch: %d runners, %d reducers", len(runners), len(reducers))
	}

	for transitionType := range reducers {
		if _, ok := runners[transitionType]; !ok {
			t.Fatalf("missing transition runner for %s", transitionType)
		}
	}
}
