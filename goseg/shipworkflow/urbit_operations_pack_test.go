package shipworkflow

import (
	"testing"
	"time"

	"groundseg/transition"
)

func withUrbitTransitionTemplateStub(t *testing.T, fn func(string, urbitTransitionTemplate, ...transitionStep[string]) error) {
	t.Helper()
	previous := runUrbitTransitionTemplateFn
	runUrbitTransitionTemplateFn = fn
	t.Cleanup(func() {
		runUrbitTransitionTemplateFn = previous
	})
}

func withPackSleepStub(t *testing.T, fn func(time.Duration)) {
	t.Helper()
	previous := sleepFn
	sleepFn = fn
	t.Cleanup(func() {
		sleepFn = previous
	})
}

func TestRunPackUsesPackTransitionTemplate(t *testing.T) {
	patp := "~zod"
	var gotStep transitionStep[string]
	var gotSteps int

	withUrbitTransitionTemplateStub(t, func(gotPatp string, template urbitTransitionTemplate, steps ...transitionStep[string]) error {
		if gotPatp != patp {
			t.Fatalf("unexpected patp: %q", gotPatp)
		}
		if template.transitionType != string(transition.UrbitTransitionPack) {
			t.Fatalf("unexpected transition type: %q", template.transitionType)
		}
		if template.startEvent != "packing" || template.successEvent != "success" || template.clearEvent != "" || !template.emitSuccess {
			t.Fatalf("unexpected transition template: %#v", template)
		}
		gotSteps = len(steps)
		if gotSteps != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}
		gotStep = steps[0]
		return nil
	})

	if err := RunPack(patp); err != nil {
		t.Fatalf("RunPack returned error: %v", err)
	}

	if gotStep.Run == nil {
		t.Fatal("expected transition step to have Run function")
	}
}

func TestRunScheduledPackSleepsForDelay(t *testing.T) {
	patp := "~zod"
	withPackSleepStub(t, func(_ time.Duration) {})

	var gotTemplate urbitTransitionTemplate
	withUrbitTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, steps ...transitionStep[string]) error {
		gotTemplate = template
		if len(steps) != 1 {
			t.Fatalf("expected 1 step, got %d", len(steps))
		}
		return nil
	})

	if err := RunScheduledPack(patp, 2*time.Millisecond); err != nil {
		t.Fatalf("RunScheduledPack returned error: %v", err)
	}

	if gotTemplate.transitionType != string(transition.UrbitTransitionPack) {
		t.Fatalf("unexpected transition type: %q", gotTemplate.transitionType)
	}
}

func TestPackMeldAndRollChopUseDistinctTransitions(t *testing.T) {
	wantMeld := string(transition.UrbitTransitionPackMeld)
	wantRollChop := string(transition.UrbitTransitionRollChop)

	var gotMeld string
	withUrbitTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, steps ...transitionStep[string]) error {
		gotMeld = template.transitionType
		if len(steps) != 5 {
			t.Fatalf("expected 5 pack-meld steps, got %d", len(steps))
		}
		return nil
	})
	if err := packMeldPier("~zod"); err != nil {
		t.Fatalf("packMeldPier returned error: %v", err)
	}
	if gotMeld != wantMeld {
		t.Fatalf("expected %q transition, got %q", wantMeld, gotMeld)
	}

	var gotRollChop string
	withUrbitTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, steps ...transitionStep[string]) error {
		gotRollChop = template.transitionType
		if len(steps) != 5 {
			t.Fatalf("expected 5 roll/chop steps, got %d", len(steps))
		}
		return nil
	})
	if err := rollChopPier("~zod"); err != nil {
		t.Fatalf("rollChopPier returned error: %v", err)
	}
	if gotRollChop != wantRollChop {
		t.Fatalf("expected %q transition, got %q", wantRollChop, gotRollChop)
	}
}

func TestPackPierAndRunPackShareBehavior(t *testing.T) {
	patp := "~zod"
	withUrbitTransitionTemplateStub(t, func(_ string, template urbitTransitionTemplate, _ ...transitionStep[string]) error {
		return nil
	})
	if err := packPier(patp); err != nil {
		t.Fatalf("packPier returned error: %v", err)
	}

	if err := RunPack(patp); err != nil {
		t.Fatalf("RunPack returned error: %v", err)
	}
}
