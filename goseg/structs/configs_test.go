package structs

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUrbitDockerUnmarshalJSONParsesKnownFields(t *testing.T) {
	raw := map[string]any{
		"pier_name":             "zod",
		"http_port":             float64(8080),
		"ames_port":             float64(34343),
		"wg_http_port":          float64(9443),
		"network":               "wireguard",
		"minio_linked":          true,
		"custom_pier_location":  "/groundseg-1",
		"disable_ship_restarts": true,
		"startram_reminder":     false,
		"chop_on_upgrade":       false,
		"remote_tlon_backup":    false,
		"local_tlon_backup":     false,
		"snap_time":             float64(120),
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal raw json: %v", err)
	}
	var ship UrbitDocker
	if err := json.Unmarshal(data, &ship); err != nil {
		t.Fatalf("unmarshal UrbitDocker: %v", err)
	}
	if ship.PierName != "zod" || ship.HTTPPort != 8080 || ship.AmesPort != 34343 {
		t.Fatalf("unexpected basic fields: %+v", ship)
	}
	if ship.WgHTTPPort != 9443 || ship.Network != "wireguard" {
		t.Fatalf("unexpected wg/network fields: %+v", ship)
	}
	if !ship.MinIOLinked {
		t.Fatal("expected minio_linked true")
	}
	if ship.CustomPierLocation != "/groundseg-1" {
		t.Fatalf("unexpected custom pier location: %+v", ship.CustomPierLocation)
	}
	if ship.DisableShipRestarts != true {
		t.Fatalf("unexpected disable_ship_restarts: %+v", ship.DisableShipRestarts)
	}
	if ship.StartramReminder != false || ship.ChopOnUpgrade != false {
		t.Fatalf("unexpected reminder/chop values: reminder=%+v chop=%+v", ship.StartramReminder, ship.ChopOnUpgrade)
	}
	if ship.RemoteTlonBackup || ship.LocalTlonBackup {
		t.Fatalf("expected backup flags false, got remote=%v local=%v", ship.RemoteTlonBackup, ship.LocalTlonBackup)
	}
	if ship.SnapTime != 120 {
		t.Fatalf("unexpected snap_time: %d", ship.SnapTime)
	}
}

func TestUrbitDockerUnmarshalJSONRejectsInvalidTypes(t *testing.T) {
	raw := map[string]any{
		"minio_linked":          "not-bool",
		"disable_ship_restarts": 1,
		"startram_reminder":     map[string]any{"value": true},
		"chop_on_upgrade":       nil,
		"remote_tlon_backup":    nil,
		"local_tlon_backup":     "yes",
		"custom_pier_location":  123,
		"http_port":             "8080",
		"pier_name":             3,
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal raw json: %v", err)
	}
	var ship UrbitDocker
	if err := json.Unmarshal(data, &ship); err != nil {
		return
	}
	t.Fatal("expected unmarshal with invalid types to fail")
}

func TestUrbitDockerUnmarshalJSONRejectsInvalidJSON(t *testing.T) {
	var ship UrbitDocker
	if err := json.Unmarshal([]byte("{invalid"), &ship); err == nil {
		t.Fatal("expected invalid json error")
	}
}

func TestUpdateConnectivityConfigRejectsNilCallback(t *testing.T) {
	conf := &SysConfig{}
	err := conf.UpdateConnectivityConfig(nil)
	if err == nil {
		t.Fatal("expected UpdateConnectivityConfig nil callback error")
	}
}

func TestUpdateConnectivityConfigWritesWithCorrectType(t *testing.T) {
	conf := &SysConfig{}
	err := conf.UpdateConnectivityConfig(func(c *ConnectivityConfig) {
		c.EndpointURL = "wss://example"
	})
	if err != nil {
		t.Fatalf("expected section update success, got: %v", err)
	}
	if conf.Connectivity.EndpointURL != "wss://example" {
		t.Fatalf("expected endpoint URL to be updated, got %q", conf.Connectivity.EndpointURL)
	}
}

func TestRuntimeConfigMarshalUsesCanonicalCfgDir(t *testing.T) {
	raw, err := json.Marshal(RuntimeConfig{CfgDir: "/tmp/groundseg"})
	if err != nil {
		t.Fatalf("marshal runtime config: %v", err)
	}
	payload := string(raw)
	if !strings.Contains(payload, `"cfgDir":"\/tmp\/groundseg"`) &&
		!strings.Contains(payload, `"cfgDir":"/tmp/groundseg"`) {
		t.Fatalf("expected canonical cfgDir key in payload: %s", payload)
	}
	if strings.Contains(payload, `"CFG_DIR"`) {
		t.Fatalf("expected legacy CFG_DIR key to be omitted from payload: %s", payload)
	}
}

func TestRuntimeConfigUnmarshalAcceptsLegacyCfgDir(t *testing.T) {
	var cfg RuntimeConfig
	if err := json.Unmarshal([]byte(`{"CFG_DIR":"/legacy/path"}`), &cfg); err != nil {
		t.Fatalf("unmarshal legacy cfg dir: %v", err)
	}
	if cfg.CfgDir != "/legacy/path" {
		t.Fatalf("expected legacy cfg dir to populate CfgDir, got %q", cfg.CfgDir)
	}
}

func TestRuntimeConfigUnmarshalPrefersCanonicalCfgDir(t *testing.T) {
	var cfg RuntimeConfig
	if err := json.Unmarshal([]byte(`{"cfgDir":"/canonical","CFG_DIR":"/legacy"}`), &cfg); err != nil {
		t.Fatalf("unmarshal cfg dir variants: %v", err)
	}
	if cfg.CfgDir != "/canonical" {
		t.Fatalf("expected canonical cfgDir to win, got %q", cfg.CfgDir)
	}
}
