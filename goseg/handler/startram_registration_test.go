package handler

import (
	"groundseg/config"
	"groundseg/startram"
	"testing"
)

func TestConcurrentStartramRegistrationIsIgnored(t *testing.T) {
	startramRegistrationMu.Lock()
	defer startramRegistrationMu.Unlock()
	before := config.Conf()
	events := len(startram.EventBus)
	handleStartramRegister("unused", "us-west")
	after := config.Conf()
	if before.Pubkey != after.Pubkey || before.Privkey != after.Privkey {
		t.Fatal("duplicate registration rotated the key")
	}
	if len(startram.EventBus) != events {
		t.Fatal("duplicate registration changed the active registration UI")
	}
}
