package docker

import (
	"encoding/json"
	"groundseg/config"
	"groundseg/structs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
)

func TestShipNetworkRecovery(t *testing.T) {
	for _, tt := range []struct {
		name          string
		bootStatus    string
		network       string
		attached      string
		state         string
		targetError   int
		attachedError int
		stopError     int
		wantError     bool
		wantActions   []string
	}{
		{name: "noboot with removed target", bootStatus: "noboot", network: "wireguard", attached: "missing-wg", state: "exited", wantActions: []string{"remove", "create"}},
		{name: "noboot with replaced target", bootStatus: "noboot", network: "wireguard", attached: "other-wg", state: "created", wantActions: []string{"remove", "create"}},
		{name: "noboot with current target", bootStatus: "noboot", network: "wireguard", attached: "current-wg", state: "exited"},
		{name: "noboot with target name", bootStatus: "noboot", network: "wireguard", attached: "wireguard", state: "created"},
		{name: "local noboot", bootStatus: "noboot", network: "none", state: "exited"},
		{name: "start created ship with removed target", bootStatus: "boot", network: "wireguard", attached: "missing-wg", state: "created", wantActions: []string{"remove", "create", "start"}},
		{name: "start exited ship with removed target", bootStatus: "boot", network: "wireguard", attached: "missing-wg", state: "exited", wantActions: []string{"remove", "create", "start"}},
		{name: "running ship with replaced target", bootStatus: "boot", network: "wireguard", attached: "other-wg", state: "running", wantActions: []string{"stop", "remove", "create", "start"}},
		{name: "current running ship", bootStatus: "boot", network: "wireguard", attached: "current-wg", state: "running"},
		{name: "missing wireguard", bootStatus: "noboot", network: "wireguard", attached: "missing-wg", state: "exited", targetError: 404, wantError: true},
		{name: "unavailable wireguard", bootStatus: "boot", network: "wireguard", attached: "missing-wg", state: "created", targetError: 500, wantError: true},
		{name: "attached inspection fails", bootStatus: "noboot", network: "wireguard", attached: "other-wg", state: "exited", attachedError: 500, wantError: true},
		{name: "stop fails", bootStatus: "boot", network: "wireguard", attached: "other-wg", state: "running", stopError: 500, wantError: true, wantActions: []string{"stop"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			basePath, dockerDir := config.BasePath, config.DockerDir
			configs := config.UrbitsConfig
			t.Cleanup(func() {
				config.BasePath, config.DockerDir = basePath, dockerDir
				config.UrbitsConfig = configs
			})
			config.BasePath = t.TempDir()
			config.DockerDir = filepath.Join(config.BasePath, "volumes")
			config.UrbitsConfig = make(map[string]structs.UrbitDocker)
			pierPath := filepath.Join(config.DockerDir, "zod", "_data")
			for _, dir := range []string{pierPath, filepath.Join(config.BasePath, "settings", "pier")} {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
			}
			dataPath := filepath.Join(pierPath, "pier-data")
			if err := os.WriteFile(dataPath, []byte("preserved"), 0600); err != nil {
				t.Fatal(err)
			}
			ship := structs.UrbitDocker{PierName: "zod", BootStatus: tt.bootStatus, Network: tt.network, LoomSize: 31}
			raw, _ := json.Marshal(ship)
			if err := os.WriteFile(filepath.Join(config.BasePath, "settings", "pier", "zod.json"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := config.LoadUrbitConfig("zod"); err != nil {
				t.Fatal(err)
			}

			exists, id, state := true, "ship-id", tt.state
			mode := "default"
			if tt.attached != "" {
				mode = "container:" + tt.attached
			}
			var actions []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := r.URL.Path
				if strings.HasPrefix(path, "/v") {
					_, path, _ = strings.Cut(path[1:], "/")
					path = "/" + path
				}
				w.Header().Set("Content-Type", "application/json")
				respond := func(value any) { _ = json.NewEncoder(w).Encode(value) }
				fail := func(code int) { w.WriteHeader(code); respond(map[string]string{"message": "inspection unavailable"}) }
				switch {
				case path == "/_ping":
					w.Header().Set("API-Version", "1.51")
				case strings.HasPrefix(path, "/images/") && strings.HasSuffix(path, "/json"):
					respond(map[string]string{"Id": "sha256:image-id"})
				case path == "/containers/json":
					list := []container.Summary{}
					if exists {
						list = append(list, container.Summary{ID: id, Names: []string{"/zod"}, State: state, Image: "vere:test", ImageID: "sha256:image-id"})
					}
					respond(list)
				case path == "/containers/wireguard/json" || path == "/containers/current-wg/json":
					if tt.targetError != 0 {
						fail(tt.targetError)
					} else {
						respond(map[string]string{"Id": "current-wg"})
					}
				case path == "/containers/missing-wg/json":
					fail(404)
				case path == "/containers/other-wg/json":
					if tt.attachedError != 0 {
						fail(tt.attachedError)
					} else {
						respond(map[string]string{"Id": "other-wg"})
					}
				case path == "/containers/zod/json":
					if !exists {
						fail(404)
					} else {
						respond(map[string]any{"Id": id, "State": map[string]any{"Status": state, "Running": state == "running"}, "HostConfig": map[string]string{"NetworkMode": mode}})
					}
				case r.Method == "POST" && path == "/containers/ship-id/stop":
					actions = append(actions, "stop")
					if tt.stopError != 0 {
						fail(tt.stopError)
					} else {
						state = "exited"
						w.WriteHeader(204)
					}
				case r.Method == "DELETE" && path == "/containers/ship-id":
					if r.URL.Query().Get("v") == "1" || r.URL.Query().Get("force") == "1" {
						t.Error("network repair must preserve volumes and remove without force")
					}
					actions = append(actions, "remove")
					exists = false
					w.WriteHeader(204)
				case r.Method == "POST" && path == "/containers/create":
					if exists {
						fail(409)
						return
					}
					var body struct{ HostConfig container.HostConfig }
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body.HostConfig.NetworkMode != "container:wireguard" {
						t.Errorf("unexpected network: %s", body.HostConfig.NetworkMode)
					}
					if len(body.HostConfig.Mounts) != 1 || body.HostConfig.Mounts[0].Source != "zod" {
						t.Errorf("pier mount changes: %+v", body.HostConfig.Mounts)
					}
					actions = append(actions, "create")
					exists, id, state, mode = true, "new-ship-id", "created", "container:current-wg"
					w.WriteHeader(201)
					respond(map[string]string{"Id": id})
				case r.Method == "POST" && path == "/containers/zod/start":
					actions = append(actions, "start")
					state = "running"
					w.WriteHeader(204)
				default:
					t.Errorf("unexpected Docker request: %s %s", r.Method, r.URL)
					fail(500)
				}
			}))
			defer server.Close()
			t.Setenv("DOCKER_HOST", server.URL)
			t.Setenv("DOCKER_TLS_VERIFY", "")
			t.Setenv("DOCKER_CERT_PATH", "")
			var info structs.ContainerState
			var err error
			if tt.bootStatus == "noboot" {
				info, err = CreateContainer("zod", "vere")
			} else {
				info, err = StartContainer("zod", "vere")
			}
			if (err != nil) != tt.wantError {
				t.Fatalf("error = %v, want error = %v", err, tt.wantError)
			}
			if !tt.wantError && tt.bootStatus == "noboot" {
				info, err = CreateContainer("zod", "vere")
				if err != nil {
					t.Fatalf("repeated reconciliation: %v", err)
				}
			}
			if !reflect.DeepEqual(actions, tt.wantActions) {
				t.Errorf("actions = %v, want %v", actions, tt.wantActions)
			}
			if !tt.wantError && (info.ID != id || info.ActualStatus != state) {
				t.Errorf("incorrect container state: %+v", info)
			}
			if !tt.wantError && tt.bootStatus == "noboot" && (state == "running" || info.DesiredStatus != "stopped") {
				t.Error("noboot ship must remain stopped")
			}
			if err := config.LoadUrbitConfig("zod"); err != nil {
				t.Fatal(err)
			}
			if got := config.UrbitConf("zod").BootStatus; got != tt.bootStatus {
				t.Errorf("boot status = %s, want %s", got, tt.bootStatus)
			}
			if data, err := os.ReadFile(dataPath); err != nil || string(data) != "preserved" {
				t.Fatalf("pier data changes: %q, %v", data, err)
			}
		})
	}
}
