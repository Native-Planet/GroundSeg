package routines

import (
	"groundseg/structs"
	"testing"

	eventtypes "github.com/docker/docker/api/types/events"
)

func TestIsCurrentContainerEvent(t *testing.T) {
	state := structs.ContainerState{ID: "abcdef1234567890"}

	tests := []struct {
		name    string
		eventID string
		want    bool
	}{
		{name: "full match", eventID: "abcdef1234567890", want: true},
		{name: "short event id", eventID: "abcdef12", want: true},
		{name: "mismatch", eventID: "123456abcdef", want: false},
		{name: "empty event id", eventID: "", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := eventtypes.Message{Actor: eventtypes.Actor{ID: tt.eventID}}
			if got := isCurrentContainerEvent(state, event); got != tt.want {
				t.Fatalf("isCurrentContainerEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCurrentContainerEventUsesMessageIDFallback(t *testing.T) {
	state := structs.ContainerState{ID: "abcdef1234567890"}
	event := eventtypes.Message{ID: "abcdef12"}

	if !isCurrentContainerEvent(state, event) {
		t.Fatal("expected message ID fallback to match current container")
	}
}
