package docker

import (
	"groundseg/structs"
	"testing"
)

func TestObjectStorePortsUseWireguardWhenStarTramRunning(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: true}
	shipConf := structs.UrbitDocker{
		HTTPPort:      8080,
		WgS3Port:      3111,
		WgConsolePort: 3112,
	}

	portConf := objectStorePorts(conf, shipConf)

	if !portConf.useWireguard {
		t.Fatalf("expected wireguard mode")
	}
	if portConf.listenS3Port != 3111 {
		t.Fatalf("expected wireguard s3 port 3111, got %d", portConf.listenS3Port)
	}
	if portConf.listenConsolePort != 3112 {
		t.Fatalf("expected wireguard console port 3112, got %d", portConf.listenConsolePort)
	}
	if portConf.hostS3Port != 0 || portConf.hostConsolePort != 0 {
		t.Fatalf("expected no host bindings in wireguard mode, got %+v", portConf)
	}
}

func TestObjectStorePortsUseOfflineHostBindingsWhenStarTramDisabled(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: false}
	shipConf := structs.UrbitDocker{HTTPPort: 8110}

	portConf := objectStorePorts(conf, shipConf)

	if portConf.useWireguard {
		t.Fatalf("expected offline mode")
	}
	if portConf.listenS3Port != offlineRustFSS3Port {
		t.Fatalf("expected internal s3 port %d, got %d", offlineRustFSS3Port, portConf.listenS3Port)
	}
	if portConf.listenConsolePort != offlineRustFSUIPort {
		t.Fatalf("expected internal console port %d, got %d", offlineRustFSUIPort, portConf.listenConsolePort)
	}
	if portConf.hostS3Port != 10110 {
		t.Fatalf("expected host s3 port 10110, got %d", portConf.hostS3Port)
	}
	if portConf.hostConsolePort != 9110 {
		t.Fatalf("expected host console port 9110, got %d", portConf.hostConsolePort)
	}
}

func TestObjectStoreLinkEndpointFallsBackToHostGatewayOffline(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: false}
	shipConf := structs.UrbitDocker{HTTPPort: 8080}

	endpoint := objectStoreLinkEndpoint(conf, shipConf)

	if endpoint != "http://host.docker.internal:10080" {
		t.Fatalf("unexpected offline endpoint: %s", endpoint)
	}
}

func TestObjectStoreLinkEndpointUsesLocalCustomDomainOffline(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: false}
	shipConf := structs.UrbitDocker{
		HTTPPort:         8080,
		CustomS3WebLocal: "local.storage.example.com",
	}

	endpoint := objectStoreLinkEndpoint(conf, shipConf)

	if endpoint != "local.storage.example.com" {
		t.Fatalf("unexpected offline custom endpoint: %s", endpoint)
	}
}

func TestObjectStoreLinkEndpointUsesRemoteCustomDomainWhenStarTramRunning(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: true}
	shipConf := structs.UrbitDocker{
		Network:           "wireguard",
		WgURL:             "sampel-palnet.nativeplanet.live",
		CustomS3WebLocal:  "local.storage.example.com",
		CustomS3WebRemote: "remote.storage.example.com",
	}

	endpoint := objectStoreLinkEndpoint(conf, shipConf)

	if endpoint != "remote.storage.example.com" {
		t.Fatalf("unexpected remote custom endpoint: %s", endpoint)
	}
}

func TestObjectStoreUsesLocalModeWhenShipIsNotRemote(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: true}
	shipConf := structs.UrbitDocker{
		Network:           "bridge",
		WgURL:             "sampel-palnet.nativeplanet.live",
		HTTPPort:          8080,
		CustomS3WebLocal:  "local.storage.example.com",
		CustomS3WebRemote: "remote.storage.example.com",
	}

	if ObjectStoreCustomDomainMode(conf, shipConf) != "local" {
		t.Fatalf("expected local mode for non-remote ship")
	}

	endpoint := objectStoreLinkEndpoint(conf, shipConf)
	if endpoint != "local.storage.example.com" {
		t.Fatalf("unexpected local endpoint for non-remote ship: %s", endpoint)
	}

	url := ObjectStoreConsoleURL("nativeplanet.local", conf, shipConf)
	if url != "http://nativeplanet.local:9080/rustfs/console/index.html" {
		t.Fatalf("unexpected console url for non-remote ship: %s", url)
	}
}

func TestSyncObjectStoreCustomDomainsKeepsLegacyCompatibilityField(t *testing.T) {
	shipConf := structs.UrbitDocker{
		CustomS3Web: "legacy.storage.example.com",
	}

	structs.SyncCustomS3Domains(&shipConf)

	if shipConf.CustomS3WebLocal != "legacy.storage.example.com" {
		t.Fatalf("expected local custom domain to inherit legacy value, got %q", shipConf.CustomS3WebLocal)
	}
	if shipConf.CustomS3WebRemote != "legacy.storage.example.com" {
		t.Fatalf("expected remote custom domain to inherit legacy value, got %q", shipConf.CustomS3WebRemote)
	}
	if shipConf.CustomS3Web != "legacy.storage.example.com" {
		t.Fatalf("expected legacy compatibility field to stay populated, got %q", shipConf.CustomS3Web)
	}
}

func TestSetObjectStoreCustomDomainPreservesLegacyLocalCompatibility(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: true}
	shipConf := structs.UrbitDocker{
		Network:           "wireguard",
		WgURL:             "sampel-palnet.nativeplanet.live",
		CustomS3Web:       "local.storage.example.com",
		CustomS3WebLocal:  "local.storage.example.com",
		CustomS3WebRemote: "remote.storage.example.com",
	}

	SetObjectStoreCustomDomain(conf, &shipConf, "new.remote.storage.example.com")

	if shipConf.CustomS3WebRemote != "new.remote.storage.example.com" {
		t.Fatalf("expected remote custom domain to update, got %q", shipConf.CustomS3WebRemote)
	}
	if shipConf.CustomS3WebLocal != "local.storage.example.com" {
		t.Fatalf("expected local custom domain to remain unchanged, got %q", shipConf.CustomS3WebLocal)
	}
	if shipConf.CustomS3Web != "local.storage.example.com" {
		t.Fatalf("expected legacy compatibility field to remain local/default, got %q", shipConf.CustomS3Web)
	}
}

func TestObjectStoreCustomDomainIgnoresNullString(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: false}
	shipConf := structs.UrbitDocker{
		HTTPPort:         8080,
		CustomS3WebLocal: "null",
	}

	if domain := ObjectStoreCustomDomain(conf, shipConf); domain != "" {
		t.Fatalf("expected null custom domain to be treated as empty, got %q", domain)
	}

	endpoint := objectStoreLinkEndpoint(conf, shipConf)
	if endpoint != "http://host.docker.internal:10080" {
		t.Fatalf("unexpected endpoint when custom domain is null: %s", endpoint)
	}
}

func TestObjectStoreConsoleURLUsesPublishedOfflineConsolePort(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: false}
	shipConf := structs.UrbitDocker{HTTPPort: 8080}

	url := ObjectStoreConsoleURL("nativeplanet.local", conf, shipConf)

	if url != "http://nativeplanet.local:9080/rustfs/console/index.html" {
		t.Fatalf("unexpected console url: %s", url)
	}
}
