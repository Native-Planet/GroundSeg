package startram

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

type failingRetrieveStateSyncer struct {
	err error
}

func (syncer failingRetrieveStateSyncer) ApplyRetrieveState(structs.StartramRetrieve) error {
	return syncer.err
}

type spyStartramConfigService struct {
	settings             config.StartramSettings
	setWgRegisteredCalls []bool
	setConfigs           []structs.StartramRetrieve
}

func (service *spyStartramConfigService) StartramSettingsSnapshot() config.StartramSettings {
	return service.settings
}

func (service *spyStartramConfigService) IsWgRegistered() bool {
	return service.settings.WgRegistered
}

func (service *spyStartramConfigService) SetWgRegistered(registered bool) error {
	service.setWgRegisteredCalls = append(service.setWgRegisteredCalls, registered)
	service.settings.WgRegistered = registered
	return nil
}

func (service *spyStartramConfigService) SetStartramConfig(retrieve structs.StartramRetrieve) {
	service.setConfigs = append(service.setConfigs, retrieve)
}

func (service *spyStartramConfigService) BasePath() string {
	return ""
}

type spyEventPublisher struct {
	events []structs.Event
}

func (publisher *spyEventPublisher) Publish(event structs.Event) {
	publisher.events = append(publisher.events, event)
}

func TestSyncRetrieveAppliesRetrievedState(t *testing.T) {
	originalAPIClient := defaultAPIClient
	originalConfigService := defaultConfigService
	originalRetrieveSyncer := defaultRetrieveStateSyncer
	t.Cleanup(func() {
		defaultAPIClient = originalAPIClient
		defaultConfigService = originalConfigService
		defaultRetrieveStateSyncer = originalRetrieveSyncer
	})

	service := &spyStartramConfigService{
		settings: config.StartramSettings{
			EndpointURL: "api.example.com",
			Pubkey:      "abc123DEF4560K",
		},
	}
	syncer := &stubRetrieveStateSyncer{}
	SetConfigService(service)
	SetRetrieveStateSyncer(syncer)
	SetAPIClient(stubStartramAPIClient{
		getFn: func(url string) (*http.Response, error) {
			if !strings.Contains(url, "/v1/retrieve") {
				t.Fatalf("unexpected retrieve URL: %s", url)
			}
			return newStartramHTTPResponse(http.StatusOK, `{"status":"active","subdomains":[]}`), nil
		},
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			t.Fatal("post should not be called by SyncRetrieve")
			return nil, nil
		},
	})

	retrieve, err := SyncRetrieve()
	if err != nil {
		t.Fatalf("SyncRetrieve returned error: %v", err)
	}
	if retrieve.Status != "active" {
		t.Fatalf("unexpected retrieve status: %q", retrieve.Status)
	}
	if syncer.calls != 1 {
		t.Fatalf("expected ApplyRetrieveState to be called once, got %d", syncer.calls)
	}
}

func TestSyncRetrievePropagatesApplyStateError(t *testing.T) {
	originalAPIClient := defaultAPIClient
	originalConfigService := defaultConfigService
	originalRetrieveSyncer := defaultRetrieveStateSyncer
	t.Cleanup(func() {
		defaultAPIClient = originalAPIClient
		defaultConfigService = originalConfigService
		defaultRetrieveStateSyncer = originalRetrieveSyncer
	})

	service := &spyStartramConfigService{
		settings: config.StartramSettings{
			EndpointURL: "api.example.com",
			Pubkey:      "abc123DEF4560K",
		},
	}
	SetConfigService(service)
	SetRetrieveStateSyncer(failingRetrieveStateSyncer{err: errors.New("apply failed")})
	SetAPIClient(stubStartramAPIClient{
		getFn: func(string) (*http.Response, error) {
			return newStartramHTTPResponse(http.StatusOK, `{"status":"active","subdomains":[]}`), nil
		},
		postFn: func(string, string, io.Reader) (*http.Response, error) {
			t.Fatal("post should not be called by SyncRetrieve")
			return nil, nil
		},
	})

	_, err := SyncRetrieve()
	if err == nil {
		t.Fatal("expected SyncRetrieve to fail when ApplyRetrieveState fails")
	}
	if !strings.Contains(err.Error(), "apply failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestApplyRetrieveStateUpdatesConfigAndPublishesEvent(t *testing.T) {
	originalConfigService := defaultConfigService
	originalRetrieveSyncer := defaultRetrieveStateSyncer
	originalEventPublisher := defaultEventPublisher
	t.Cleanup(func() {
		defaultConfigService = originalConfigService
		defaultRetrieveStateSyncer = originalRetrieveSyncer
		defaultEventPublisher = originalEventPublisher
	})

	service := &spyStartramConfigService{
		settings: config.StartramSettings{
			WgRegistered: false,
		},
	}
	publisher := &spyEventPublisher{}
	SetConfigService(service)
	SetRetrieveStateSyncer(retrieveStateSyncer{})
	SetEventPublisher(publisher)

	retrieve := structs.StartramRetrieve{
		Status: "active",
	}
	if err := ApplyRetrieveState(retrieve); err != nil {
		t.Fatalf("ApplyRetrieveState returned error: %v", err)
	}
	if len(service.setConfigs) != 1 || service.setConfigs[0].Status != "active" {
		t.Fatalf("expected retrieve state to be persisted once, got %+v", service.setConfigs)
	}
	if len(service.setWgRegisteredCalls) != 1 || !service.setWgRegisteredCalls[0] {
		t.Fatalf("expected registration status to be set to true, got %+v", service.setWgRegisteredCalls)
	}
	if len(publisher.events) != 1 || publisher.events[0].Type != "retrieve" {
		t.Fatalf("expected one retrieve event, got %+v", publisher.events)
	}
}
