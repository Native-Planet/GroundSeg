package shipworkflow

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"groundseg/structs"
	"groundseg/transition"
)

func TestRunStartramTransitionWithRuntimePublishesDefaultErrorEvent(t *testing.T) {
	t.Parallel()

	var emitted []structs.Event
	runtime := defaultStartramRuntime()
	runtime.PublishEventFn = func(event structs.Event) {
		emitted = append(emitted, event)
	}

	err := runStartramTransitionWithRuntime(
		runtime,
		transition.StartramTransitionToggle,
		transitionPlan[structs.Event]{
			EmitStart:  true,
			StartEvent: startramEvent(transition.StartramTransitionToggle, "start"),
			ClearEvent: startramEvent(transition.StartramTransitionToggle, ""),
			ClearDelay: 0,
		},
		transitionStep[structs.Event]{
			Run: func() error {
				return errors.New("boom")
			},
		},
	)
	if err == nil {
		t.Fatal("expected transition lifecycle failure")
	}

	if len(emitted) == 0 {
		t.Fatal("expected transition events to be emitted")
	}
	foundErrorEvent := false
	for _, event := range emitted {
		if event.Type != string(transition.StartramTransitionToggle) {
			continue
		}
		if strings.Contains(fmt.Sprint(event.Data), "boom") {
			foundErrorEvent = true
			break
		}
	}
	if !foundErrorEvent {
		t.Fatalf("expected emitted events to include boom error data, got %+v", emitted)
	}
}
