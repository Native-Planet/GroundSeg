package system

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseNmcliWifiDevicesReturnsPartialWarning(t *testing.T) {
	devices, err := parseNmcliWifiDevices("wlan0:wifi\nmalformed-row\n")
	if !reflect.DeepEqual(devices, []string{"wlan0"}) {
		t.Fatalf("unexpected parsed devices: %v", devices)
	}
	if err == nil {
		t.Fatal("expected partial parse warning")
	}
	var partialErr wifiNmcliPartialParseError
	if !errors.As(err, &partialErr) {
		t.Fatalf("expected wifiNmcliPartialParseError, got %T: %v", err, err)
	}
}

func TestSplitNmcliRecordHandlesEscapedSeparators(t *testing.T) {
	got := splitNmcliRecord(`wlan0:wifi:escaped\:ssid`)
	want := []string{"wlan0", "wifi", "escaped:ssid"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected nmcli record split: got %v want %v", got, want)
	}
}
