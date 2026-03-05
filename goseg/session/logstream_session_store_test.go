package session

import (
	"errors"
	"testing"
)

func TestLogstreamSessionStoreNilReceiverSessionMethodsAreNoop(t *testing.T) {
	var store *logstreamSessionStore

	if got := store.SysLogSessions(); got != nil {
		t.Fatalf("expected nil SysLogSessions for nil receiver, got %v", got)
	}
	if got := store.SysSessionsToRemove(); got != nil {
		t.Fatalf("expected nil SysSessionsToRemove for nil receiver, got %v", got)
	}
	if got := store.DockerLogSessions(); got != nil {
		t.Fatalf("expected nil DockerLogSessions for nil receiver, got %v", got)
	}

	// Nil receiver mutators are contractually benign no-ops.
	store.SetSysLogSessions(nil)
	store.SetSysSessionsToRemove(nil)
	store.AddSysLogSession(nil)
	store.RemoveSysLogSessions()
	store.AddSysSessionToRemove(nil)
	store.SetDockerLogSessions(nil)
	store.SetDockerLogSession("vere", nil, true)
	store.SetDockerLogSessionLive("vere", nil, false)
	store.RemoveDockerLogSession("vere", nil)
}

func TestLogstreamSessionStorePublishSystemLogReturnsErrorForNilReceiver(t *testing.T) {
	var store *logstreamSessionStore
	if err := store.PublishSystemLog([]byte("entry")); !errors.Is(err, ErrSystemLogBusNotDefined) {
		t.Fatalf("expected nil store publish error %v, got %v", ErrSystemLogBusNotDefined, err)
	}
}

func TestLogstreamSessionStorePublishSystemLogReturnsBusErrors(t *testing.T) {
	store := newLogstreamSessionStore()
	store.systemLogBus = newSystemLogMessageBus(1)

	if err := store.PublishSystemLog([]byte("first")); err != nil {
		t.Fatalf("publish first payload: %v", err)
	}
	if err := store.PublishSystemLog([]byte("second")); !errors.Is(err, ErrSystemLogBusFull) {
		t.Fatalf("expected full bus error %v, got %v", ErrSystemLogBusFull, err)
	}
}
