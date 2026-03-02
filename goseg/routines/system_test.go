package routines

import (
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"groundseg/structs"
)

func resetSystemSeamsForTest(t *testing.T) {
	t.Helper()
	origUpdate := updateCheckForAptLoop
	origConf := configForSystemRoutine
	origInterfaces := netInterfacesForSystemRoutine
	origAddrs := interfaceAddrsForSystemRoutine
	t.Cleanup(func() {
		updateCheckForAptLoop = origUpdate
		configForSystemRoutine = origConf
		netInterfacesForSystemRoutine = origInterfaces
		interfaceAddrsForSystemRoutine = origAddrs
	})
}

func TestAptUpdateCheckIntervalUsesConfiguredCadence(t *testing.T) {
	dayConf := structs.SysConfig{}
	dayConf.LinuxUpdates.Interval = "day"
	dayConf.LinuxUpdates.Value = 2
	if got := aptUpdateCheckInterval(dayConf); got != 48*time.Hour {
		t.Fatalf("unexpected day interval: got %v want %v", got, 48*time.Hour)
	}

	weekConf := structs.SysConfig{}
	weekConf.LinuxUpdates.Interval = "week"
	weekConf.LinuxUpdates.Value = 1
	if got := aptUpdateCheckInterval(weekConf); got != 7*24*time.Hour {
		t.Fatalf("unexpected week interval: got %v want %v", got, 7*24*time.Hour)
	}

	defaultConf := structs.SysConfig{}
	defaultConf.LinuxUpdates.Interval = "month"
	defaultConf.LinuxUpdates.Value = 3
	if got := aptUpdateCheckInterval(defaultConf); got != 90*24*time.Hour {
		t.Fatalf("unexpected default interval: got %v want %v", got, 90*24*time.Hour)
	}

	zeroConf := structs.SysConfig{}
	zeroConf.LinuxUpdates.Interval = "day"
	zeroConf.LinuxUpdates.Value = 0
	if got := aptUpdateCheckInterval(zeroConf); got != 24*time.Hour {
		t.Fatalf("expected zero interval to default to one multiplier: got %v want %v", got, 24*time.Hour)
	}

	negativeConf := structs.SysConfig{}
	negativeConf.LinuxUpdates.Interval = "week"
	negativeConf.LinuxUpdates.Value = -1
	if got := aptUpdateCheckInterval(negativeConf); got != 7*24*time.Hour {
		t.Fatalf("expected negative interval to default to one multiplier: got %v want %v", got, 7*24*time.Hour)
	}
}

func TestGetAllIPsFiltersLocalAndIPv6Addresses(t *testing.T) {
	resetSystemSeamsForTest(t)

	netInterfacesForSystemRoutine = func() ([]net.Interface, error) {
		return []net.Interface{{Name: "eth0"}, {Name: "wlan0"}}, nil
	}
	interfaceAddrsForSystemRoutine = func(i net.Interface) ([]net.Addr, error) {
		switch i.Name {
		case "eth0":
			return []net.Addr{
				&net.IPNet{IP: net.ParseIP("192.168.1.10")},
				&net.IPNet{IP: net.ParseIP("127.0.0.1")},
				&net.IPNet{IP: net.ParseIP("172.16.0.5")},
				&net.IPNet{IP: net.ParseIP("172.21.0.9")},
				&net.IPNet{IP: net.ParseIP("10.0.0.2")},
				&net.IPNet{IP: net.ParseIP("fe80::1")},
			}, nil
		case "wlan0":
			return []net.Addr{&net.IPAddr{IP: net.ParseIP("192.168.1.11")}}, nil
		default:
			return []net.Addr{}, nil
		}
	}

	ips, err := getAllIPs()
	if err != nil {
		t.Fatalf("getAllIPs returned error: %v", err)
	}
	want := []string{"192.168.1.10", "10.0.0.2", "192.168.1.11"}
	if !reflect.DeepEqual(ips, want) {
		t.Fatalf("unexpected IP list: got %v want %v", ips, want)
	}
}

func TestGetAllIPsPropagatesInterfaceErrors(t *testing.T) {
	resetSystemSeamsForTest(t)

	netInterfacesForSystemRoutine = func() ([]net.Interface, error) {
		return nil, errors.New("interfaces failed")
	}

	if _, err := getAllIPs(); err == nil {
		t.Fatal("expected getAllIPs to return interface enumeration error")
	}
}

func TestGetAllIPsPropagatesAddressErrors(t *testing.T) {
	resetSystemSeamsForTest(t)

	netInterfacesForSystemRoutine = func() ([]net.Interface, error) {
		return []net.Interface{{Name: "eth0"}}, nil
	}
	interfaceAddrsForSystemRoutine = func(net.Interface) ([]net.Addr, error) {
		return nil, errors.New("addr read failed")
	}

	if _, err := getAllIPs(); err == nil {
		t.Fatal("expected getAllIPs to return address read error")
	}
}
