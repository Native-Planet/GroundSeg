package defaults

import (
	"reflect"
	"testing"

	"groundseg/structs"
)

func TestDefaultVersionDefaultsParses(t *testing.T) {
	parsed, err := DefaultVersionDefaults()
	if err != nil {
		t.Fatalf("DefaultVersionDefaults should return parseable metadata: %v", err)
	}
	raw, err := DefaultVersionDefaults()
	if err != nil {
		t.Fatalf("DefaultVersionDefaults should return parseable metadata again: %v", err)
	}
	if !reflect.DeepEqual(parsed, raw) {
		t.Fatalf("Version defaults should be repeatable and stable")
	}
}

func TestDefaultVersionContainsExpectedChannels(t *testing.T) {
	version, err := DefaultVersionDefaults()
	if err != nil {
		t.Fatalf("DefaultVersionDefaults should return version defaults: %v", err)
	}
	for _, channelName := range []string{"canary", "edge", "latest"} {
		channel, ok := version.Groundseg[channelName]
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
