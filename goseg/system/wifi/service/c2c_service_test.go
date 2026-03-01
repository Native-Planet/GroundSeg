package service

import (
	"errors"
	"testing"
)

type stubC2CService struct {
	connectCalls int
	restartCalls int
	connectErr   error
	restartErr   error
	lastSSID     string
	lastPassword string
}

func (s *stubC2CService) ConnectToWiFi(ssid, password string) error {
	s.connectCalls++
	s.lastSSID = ssid
	s.lastPassword = password
	return s.connectErr
}

func (s *stubC2CService) RestartGroundSeg() error {
	s.restartCalls++
	return s.restartErr
}

func (s *stubC2CService) Execute(action C2CAction, ssid, password string) error {
	switch action {
	case ConnectAction:
		if err := s.ConnectToWiFi(ssid, password); err != nil {
			return err
		}
		return s.RestartGroundSeg()
	default:
		return UnsupportedC2CActionError{Namespace: "c2c", Action: action}
	}
}

func TestProcessC2CMessageDispatchesConnect(t *testing.T) {
	service := &stubC2CService{}
	cmd := C2CCommand{
		Action:   ConnectAction,
		SSID:     "mynetwork",
		Password: "s3cret",
	}

	gotErr := ProcessC2CMessage(cmd, func() C2CService { return service })
	if gotErr != nil {
		t.Fatalf("ProcessC2CMessage returned unexpected error: %v", gotErr)
	}
	if service.connectCalls != 1 {
		t.Fatalf("expected connect to be called once, got %d", service.connectCalls)
	}
	if service.restartCalls != 1 {
		t.Fatalf("expected restart to be called once, got %d", service.restartCalls)
	}
	if service.lastSSID != "mynetwork" || service.lastPassword != "s3cret" {
		t.Fatalf("unexpected connection payload values: ssid=%q password=%q", service.lastSSID, service.lastPassword)
	}
}

func TestProcessC2CMessageRejectsUnsupportedAction(t *testing.T) {
	service := &stubC2CService{}
	cmd := C2CCommand{
		Action: C2CAction("invalid"),
	}

	err := ProcessC2CMessage(cmd, func() C2CService { return service })
	var unsupported UnsupportedC2CActionError
	if !errors.As(err, &unsupported) {
		t.Fatalf("expected unsupported action error, got %v", err)
	}
}

func TestProcessC2CMessageRejectsNilServiceFactory(t *testing.T) {
	cmd := C2CCommand{
		Action: ConnectAction,
	}
	if err := ProcessC2CMessage(cmd, nil); err == nil {
		t.Fatal("expected nil service factory to fail")
	}
}

func TestProcessC2CMessageRejectsNilService(t *testing.T) {
	cmd := C2CCommand{
		Action: ConnectAction,
	}
	err := ProcessC2CMessage(cmd, func() C2CService { return nil })
	if err == nil {
		t.Fatal("expected nil service to fail")
	}
}
