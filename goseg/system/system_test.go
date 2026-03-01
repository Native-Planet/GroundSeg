package system

import (
	"testing"

	"groundseg/structs"
)

func TestOctalToAscii(t *testing.T) {
	got, err := octalToAscii(`cpu\040temp\072\040ok`)
	if err != nil {
		t.Fatalf("octalToAscii returned error: %v", err)
	}
	if got != "cpu temp: ok" {
		t.Fatalf("unexpected conversion result: %q", got)
	}
}

func TestIsDevMounted(t *testing.T) {
	unmounted := structs.BlockDev{Mountpoints: []string{"", ""}}
	if IsDevMounted(unmounted) {
		t.Fatal("expected unmounted device to return false")
	}

	mounted := structs.BlockDev{Mountpoints: []string{"", "/media/data"}}
	if !IsDevMounted(mounted) {
		t.Fatal("expected mounted device to return true")
	}
}
