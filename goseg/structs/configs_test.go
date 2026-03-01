package structs

import (
	"encoding/json"
	"testing"
)

func TestToInt(t *testing.T) {
	if got := toInt(float64(42)); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	if got := toInt("42"); got != 0 {
		t.Fatalf("expected fallback 0 for non-number, got %d", got)
	}
}

func TestToBoolAndToString(t *testing.T) {
	if !toBool(true, false) {
		t.Fatal("expected true bool conversion")
	}
	if toBool("true", false) {
		t.Fatal("expected default false for invalid bool type")
	}
	if !toBool(nil, true) {
		t.Fatal("expected default true for nil bool value")
	}
	if toString("abc") != "abc" {
		t.Fatal("expected string conversion")
	}
	if toString(123) != "" {
		t.Fatal("expected empty string fallback for non-string")
	}
}

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

func TestUrbitDockerUnmarshalJSONUsesSafeDefaultsForInvalidTypes(t *testing.T) {
	raw := map[string]any{
		"minio_linked":          "not-bool",
		"disable_ship_restarts": "nope",
		"startram_reminder":     nil,
		"chop_on_upgrade":       nil,
		"remote_tlon_backup":    nil,
		"local_tlon_backup":     nil,
		"custom_pier_location":  123,
	}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal raw json: %v", err)
	}
	var ship UrbitDocker
	if err := json.Unmarshal(data, &ship); err != nil {
		t.Fatalf("unmarshal with invalid types should not fail: %v", err)
	}
	if ship.MinIOLinked {
		t.Fatal("invalid minio_linked type should default to false")
	}
	if ship.DisableShipRestarts != false {
		t.Fatalf("invalid disable_ship_restarts type should default false, got %+v", ship.DisableShipRestarts)
	}
	if ship.StartramReminder != true || ship.ChopOnUpgrade != true {
		t.Fatalf("nil reminder/chop should default true, got reminder=%+v chop=%+v", ship.StartramReminder, ship.ChopOnUpgrade)
	}
	if !ship.RemoteTlonBackup || !ship.LocalTlonBackup {
		t.Fatalf("nil backup flags should default true, got remote=%v local=%v", ship.RemoteTlonBackup, ship.LocalTlonBackup)
	}
	if ship.CustomPierLocation != "" {
		t.Fatalf("invalid custom_pier_location type should coerce to empty string, got %+v", ship.CustomPierLocation)
	}
}

func TestUrbitDockerUnmarshalJSONRejectsInvalidJSON(t *testing.T) {
	var ship UrbitDocker
	if err := json.Unmarshal([]byte("{invalid"), &ship); err == nil {
		t.Fatal("expected invalid json error")
	}
}
