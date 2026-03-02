package accesspoint

import (
	"errors"
	"net"
	"testing"
)

func TestCheckParametersRequiresNetInterfaceFn(t *testing.T) {
	rt := AccessPointRuntime{
		IP:       "192.168.1.1",
		SSID:     "GroundSeg",
		Password: "valid-password-123",
	}
	if err := checkParametersWithContext(rt); err == nil {
		t.Fatal("expected missing net interfaces function error")
	}
}

func TestCheckParametersPropagatesNetInterfaceErrors(t *testing.T) {
	sentinel := errors.New("net error")
	rt := AccessPointRuntime{
		IP:       "192.168.1.1",
		SSID:     "GroundSeg",
		Password: "valid-password-123",
		NetInterfacesFn: func() ([]net.Interface, error) {
			return nil, sentinel
		},
	}
	err := checkParametersWithContext(rt)
	if err == nil {
		t.Fatal("expected net error")
	}
	if err.Error() != sentinel.Error() {
		t.Fatalf("expected propagated error %v, got %v", sentinel, err)
	}
}

func TestCheckParametersValidatesWlanAndInetPresence(t *testing.T) {
	rt := AccessPointRuntime{
		Wlan:     "wlanX",
		IP:       "192.168.1.1",
		SSID:     "GroundSeg",
		Password: "valid-password-123",
		NetInterfacesFn: func() ([]net.Interface, error) {
			return []net.Interface{{Name: "wlan0"}, {Name: "eth0"}}, nil
		},
	}
	err := checkParametersWithContext(rt)
	if err == nil {
		t.Fatal("expected missing wlan interface error")
	}

	rt.Wlan = "wlan0"
	rt.Inet = "inetX"
	err = checkParametersWithContext(rt)
	if err == nil {
		t.Fatal("expected missing inet interface error")
	}
}

func TestCheckParametersValidatesIpSsidAndPasswordPresence(t *testing.T) {
	rt := AccessPointRuntime{
		Wlan:     "wlan0",
		IP:       "bad-ip",
		SSID:     "GroundSeg",
		Password: "valid-password-123",
		NetInterfacesFn: func() ([]net.Interface, error) {
			return []net.Interface{{Name: "wlan0"}}, nil
		},
	}
	if err := checkParametersWithContext(rt); err == nil {
		t.Fatal("expected invalid IP error")
	}

	rt.IP = "192.168.1.1"
	rt.SSID = ""
	if err := checkParametersWithContext(rt); err == nil {
		t.Fatal("expected missing SSID error")
	}

	rt.SSID = "GroundSeg"
	rt.Password = ""
	if err := checkParametersWithContext(rt); err == nil {
		t.Fatal("expected missing password error")
	}
}

func TestCheckParametersSuccess(t *testing.T) {
	rt := AccessPointRuntime{
		Wlan:     "wlan0",
		IP:       "192.168.1.1",
		SSID:     "GroundSeg",
		Password: "valid-password-123",
		NetInterfacesFn: func() ([]net.Interface, error) {
			return []net.Interface{{Name: "wlan0"}, {Name: "eth0"}}, nil
		},
	}
	if err := checkParametersWithContext(rt); err != nil {
		t.Fatalf("expected valid parameters, got error %v", err)
	}
}

func TestCheckParametersCallsDefaultContext(t *testing.T) {
	if err := checkParameters(); err == nil {
		t.Fatal("expected checkParameters to return validation error in test environment")
	}
}
