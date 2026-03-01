package defaults

import (
	"encoding/json"
	"reflect"
	"testing"

	"groundseg/structs"
)

func TestDefaultVersionTextParsesToVersionInfo(t *testing.T) {
	var parsed structs.Version
	if err := json.Unmarshal([]byte(DefaultVersionText), &parsed); err != nil {
		t.Fatalf("DefaultVersionText should be valid JSON: %v", err)
	}
	if !reflect.DeepEqual(parsed, VersionInfo) {
		t.Fatalf("VersionInfo should match DefaultVersionText parse result")
	}
}

func TestDefaultVersionContainsExpectedChannels(t *testing.T) {
	for _, channelName := range []string{"canary", "edge", "latest"} {
		channel, ok := VersionInfo.Groundseg[channelName]
		if !ok {
			t.Fatalf("expected channel %q in default version metadata", channelName)
		}
		if channel.Groundseg.Major == 0 || channel.Groundseg.Minor == 0 {
			t.Fatalf("expected semantic version fields for %q groundseg binary", channelName)
		}
		for name, details := range map[string]structs.VersionDetails{
			"manual":    channel.Manual,
			"minio":     channel.Minio,
			"miniomc":   channel.Miniomc,
			"netdata":   channel.Netdata,
			"vere":      channel.Vere,
			"webui":     channel.Webui,
			"wireguard": channel.Wireguard,
		} {
			if details.Repo == "" {
				t.Fatalf("expected repo for %s in channel %s", name, channelName)
			}
			if details.Amd64Sha256 == "" || details.Arm64Sha256 == "" {
				t.Fatalf("expected sha256 hashes for %s in channel %s", name, channelName)
			}
		}
	}
}
