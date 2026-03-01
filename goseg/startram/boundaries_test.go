package startram

import (
	"strings"
	"testing"
	"time"

	"groundseg/structs"
)

type boundariesSyncer struct {
	calls int
}

func (s *boundariesSyncer) ApplyRetrieveState(structs.StartramRetrieve) error {
	s.calls++
	return nil
}

type boundariesPublisher struct {
	events []structs.Event
}

func (p *boundariesPublisher) Publish(event structs.Event) {
	p.events = append(p.events, event)
}

type boundariesRestoreWorker struct {
	calls []RestoreBackupRequest
}

func (w *boundariesRestoreWorker) Restore(req RestoreBackupRequest) error {
	w.calls = append(w.calls, req)
	return nil
}

func drainStartramEvents() {
	for {
		select {
		case <-eventBus:
		default:
			return
		}
	}
}

func TestSetRetrieveStateSyncerIgnoresNil(t *testing.T) {
	original := defaultRetrieveStateSyncer
	t.Cleanup(func() {
		defaultRetrieveStateSyncer = original
	})

	syncer := &boundariesSyncer{}
	SetRetrieveStateSyncer(syncer)
	SetRetrieveStateSyncer(nil)

	if err := ApplyRetrieveState(structs.StartramRetrieve{}); err != nil {
		t.Fatalf("ApplyRetrieveState returned error: %v", err)
	}
	if syncer.calls != 1 {
		t.Fatalf("expected configured syncer call count 1, got %d", syncer.calls)
	}
}

func TestSetEventPublisherIgnoresNilAndPublishEventUsesConfiguredPublisher(t *testing.T) {
	original := defaultEventPublisher
	t.Cleanup(func() {
		defaultEventPublisher = original
	})

	publisher := &boundariesPublisher{}
	SetEventPublisher(publisher)
	SetEventPublisher(nil)

	PublishEvent(structs.Event{Type: "configured-publisher"})
	if len(publisher.events) != 1 || publisher.events[0].Type != "configured-publisher" {
		t.Fatalf("unexpected published events: %+v", publisher.events)
	}
}

func TestPublishEventWithChannelPublisherEmitsToEventBus(t *testing.T) {
	original := defaultEventPublisher
	t.Cleanup(func() {
		defaultEventPublisher = original
	})

	drainStartramEvents()
	SetEventPublisher(channelEventPublisher{})

	PublishEvent(structs.Event{Type: "bus"})
	select {
	case event := <-Events():
		if event.Type != "bus" {
			t.Fatalf("unexpected event from bus: %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event bus publish")
	}
}

func TestRestoreWorkerRestoreValidatesAndDispatchesByMode(t *testing.T) {
	origDev := restoreBackupDevForWorker
	origProd := restoreBackupProdForWorker
	t.Cleanup(func() {
		restoreBackupDevForWorker = origDev
		restoreBackupProdForWorker = origProd
	})

	rw := restoreWorker{}
	if err := rw.Restore(RestoreBackupRequest{}); err == nil || !strings.Contains(err.Error(), "ship is required") {
		t.Fatalf("expected missing ship validation error, got %v", err)
	}

	devCalled := ""
	restoreBackupDevForWorker = func(ship string) error {
		devCalled = ship
		return nil
	}
	restoreBackupProdForWorker = func(req RestoreBackupRequest) error {
		t.Fatalf("did not expect prod restore for development mode: %+v", req)
		return nil
	}
	devReq := RestoreBackupRequest{Ship: "~zod", Mode: RestoreBackupModeDevelopment}
	if err := rw.Restore(devReq); err != nil {
		t.Fatalf("development restore returned error: %v", err)
	}
	if devCalled != "~zod" {
		t.Fatalf("expected dev restore for ~zod, got %q", devCalled)
	}

	prodCalled := false
	restoreBackupProdForWorker = func(req RestoreBackupRequest) error {
		prodCalled = true
		if req.Ship != "~bus" || req.Mode != RestoreBackupModeProduction {
			t.Fatalf("unexpected production request: %+v", req)
		}
		return nil
	}
	if err := rw.Restore(RestoreBackupRequest{Ship: "~bus", Mode: RestoreBackupModeProduction}); err != nil {
		t.Fatalf("production restore returned error: %v", err)
	}
	if !prodCalled {
		t.Fatal("expected production restore to be dispatched")
	}

	if err := rw.Restore(RestoreBackupRequest{Ship: "~nec", Mode: RestoreBackupMode("invalid")}); err == nil || !strings.Contains(err.Error(), "unsupported restore mode") {
		t.Fatalf("expected unsupported mode error, got %v", err)
	}
}

func TestSetRestoreWorkerIgnoresNil(t *testing.T) {
	original := defaultRestoreWorker
	t.Cleanup(func() {
		defaultRestoreWorker = original
	})

	worker := &boundariesRestoreWorker{}
	SetRestoreWorker(worker)
	SetRestoreWorker(nil)

	req := RestoreBackupRequest{Ship: "~zod", Mode: RestoreBackupModeDevelopment}
	if err := defaultRestoreWorker.Restore(req); err != nil {
		t.Fatalf("default restore worker returned error: %v", err)
	}
	if len(worker.calls) != 1 || worker.calls[0].Ship != "~zod" {
		t.Fatalf("expected configured restore worker to receive request, got %+v", worker.calls)
	}
}
