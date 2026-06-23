package docker

import "testing"

func TestIsMaintenanceBootStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: "pack", want: true},
		{status: "meld", want: true},
		{status: "chop", want: true},
		{status: "rollchop", want: true},
		{status: "prep", want: true},
		{status: "roll", want: true},
		{status: "boot", want: false},
		{status: "noboot", want: false},
		{status: "ignore", want: false},
	}

	for _, test := range tests {
		if got := IsMaintenanceBootStatus(test.status); got != test.want {
			t.Fatalf("IsMaintenanceBootStatus(%q) = %v, want %v", test.status, got, test.want)
		}
	}
}

func TestPersistentBootStatusAfterContainerBuild(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{status: "pack", want: "noboot"},
		{status: "meld", want: "noboot"},
		{status: "chop", want: "noboot"},
		{status: "rollchop", want: "noboot"},
		{status: "noboot", want: "noboot"},
		{status: "ignore", want: "ignore"},
		{status: "boot", want: "boot"},
		{status: "prep", want: "boot"},
		{status: "roll", want: "boot"},
	}

	for _, test := range tests {
		if got := PersistentBootStatusAfterContainerBuild(test.status); got != test.want {
			t.Fatalf("PersistentBootStatusAfterContainerBuild(%q) = %q, want %q", test.status, got, test.want)
		}
	}
}
