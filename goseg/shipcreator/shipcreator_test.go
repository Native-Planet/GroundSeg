package shipcreator

import (
	"path/filepath"
	"reflect"
	"testing"

	"groundseg/structs"
)

type mockConfigService struct {
	conf              structs.SysConfig
	urbits            map[string]structs.UrbitDocker
	updatedUrbitConf  map[string]structs.UrbitDocker
	updatedPiers      []string
	updateUrbitErr    error
	updatePiersErr    error
	updateUrbitCalled bool
	updatePiersCalled bool
}

func (m *mockConfigService) ExistingPiers() []string {
	return append([]string(nil), m.conf.Piers...)
}

func (m *mockConfigService) ExistingShipPorts() ([]int, []int) {
	httpPorts := make([]int, 0, len(m.conf.Piers))
	amesPorts := make([]int, 0, len(m.conf.Piers))
	for _, pier := range m.conf.Piers {
		uConf := m.urbits[pier]
		httpPorts = append(httpPorts, uConf.HTTPPort)
		amesPorts = append(amesPorts, uConf.AmesPort)
	}
	return httpPorts, amesPorts
}

func (m *mockConfigService) SaveShipConfig(patp string, conf structs.UrbitDocker) error {
	m.updateUrbitCalled = true
	if m.updatedUrbitConf == nil {
		m.updatedUrbitConf = make(map[string]structs.UrbitDocker)
	}
	m.updatedUrbitConf[patp] = conf
	return m.updateUrbitErr
}

func (m *mockConfigService) SavePiers(piers []string) error {
	m.updatePiersCalled = true
	m.updatedPiers = append([]string(nil), piers...)
	return m.updatePiersErr
}

func TestCreateUrbitConfigAssignsPortsAndCustomDrive(t *testing.T) {
	mock := &mockConfigService{
		conf: structs.SysConfig{ConnectivityConfig: structs.ConnectivityConfig{Piers: []string{"~bus", "~nec"}}},
		urbits: map[string]structs.UrbitDocker{
			"~bus": {
				UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
					HTTPPort: 8081,
					AmesPort: 34344,
				},
			},
			"~nec": {
				UrbitRuntimeConfig: structs.UrbitRuntimeConfig{
					HTTPPort: 8083,
					AmesPort: 34346,
				},
			},
		},
	}
	svc := NewService(mock)

	if err := svc.CreateUrbitConfig("~zod", "/mnt/data"); err != nil {
		t.Fatalf("CreateUrbitConfig returned error: %v", err)
	}
	if !mock.updateUrbitCalled {
		t.Fatal("expected SaveShipConfig to be called")
	}
	created, exists := mock.updatedUrbitConf["~zod"]
	if !exists {
		t.Fatalf("expected created urbit config for ~zod, got %+v", mock.updatedUrbitConf)
	}
	if created.PierName != "~zod" {
		t.Fatalf("unexpected pier name: %q", created.PierName)
	}
	if created.HTTPPort != 8082 || created.AmesPort != 34345 {
		t.Fatalf("unexpected assigned ports: http=%d ames=%d", created.HTTPPort, created.AmesPort)
	}
	wantPath := filepath.Join("/mnt/data", "~zod")
	if created.CustomPierLocation != wantPath {
		t.Fatalf("unexpected custom pier path: got %v want %v", created.CustomPierLocation, wantPath)
	}
}

func TestGetOpenUrbitPortsDefaultsWhenNoPiers(t *testing.T) {
	mock := &mockConfigService{conf: structs.SysConfig{ConnectivityConfig: structs.ConnectivityConfig{Piers: []string{}}}, urbits: map[string]structs.UrbitDocker{}}
	svc := NewService(mock)

	httpPort, amesPort := svc.getOpenUrbitPorts()
	if httpPort != 8081 || amesPort != 34344 {
		t.Fatalf("unexpected default ports: http=%d ames=%d", httpPort, amesPort)
	}
}

func TestAppendSysConfigPierAddsOnlyWhenMissing(t *testing.T) {
	mock := &mockConfigService{conf: structs.SysConfig{ConnectivityConfig: structs.ConnectivityConfig{Piers: []string{"~bus"}}}}
	svc := NewService(mock)

	if err := svc.AppendSysConfigPier("~zod"); err != nil {
		t.Fatalf("AppendSysConfigPier returned error: %v", err)
	}
	if !reflect.DeepEqual(mock.updatedPiers, []string{"~bus", "~zod"}) {
		t.Fatalf("unexpected piers after append: %+v", mock.updatedPiers)
	}

	mock2 := &mockConfigService{conf: structs.SysConfig{ConnectivityConfig: structs.ConnectivityConfig{Piers: []string{"~bus"}}}}
	svc2 := NewService(mock2)
	if err := svc2.AppendSysConfigPier("~bus"); err != nil {
		t.Fatalf("AppendSysConfigPier(existing) returned error: %v", err)
	}
	if !reflect.DeepEqual(mock2.updatedPiers, []string{"~bus"}) {
		t.Fatalf("expected no duplicate pier entry, got %+v", mock2.updatedPiers)
	}
}

func TestRemoveSysConfigPierRemovesAllInstances(t *testing.T) {
	mock := &mockConfigService{conf: structs.SysConfig{ConnectivityConfig: structs.ConnectivityConfig{Piers: []string{"~zod", "~bus", "~zod"}}}}
	svc := NewService(mock)

	if err := svc.RemoveSysConfigPier("~zod"); err != nil {
		t.Fatalf("RemoveSysConfigPier returned error: %v", err)
	}
	if !reflect.DeepEqual(mock.updatedPiers, []string{"~bus"}) {
		t.Fatalf("unexpected piers after removal: %+v", mock.updatedPiers)
	}
}

func TestContains(t *testing.T) {
	if !contains([]int{1, 3, 5, 7}, 5) {
		t.Fatal("expected contains to find existing value")
	}
	if contains([]int{1, 3, 5, 7}, 2) {
		t.Fatal("expected contains to return false for missing value")
	}
}
