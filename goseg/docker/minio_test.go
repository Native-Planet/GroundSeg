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

func TestObjectStoreConsoleURLUsesPublishedOfflineConsolePort(t *testing.T) {
	conf := structs.SysConfig{WgRegistered: true, WgOn: false}
	shipConf := structs.UrbitDocker{HTTPPort: 8080}

	url := ObjectStoreConsoleURL("nativeplanet.local", conf, shipConf)

	if url != "http://nativeplanet.local:9080/rustfs/console/index.html" {
		t.Fatalf("unexpected console url: %s", url)
	}
}
