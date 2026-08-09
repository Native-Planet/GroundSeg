package docker

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"groundseg/config"
	"groundseg/structs"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"go.uber.org/zap"
)

const (
	beszelHubContainerName   = "netdata"
	beszelAgentContainerName = "beszel-agent"
	beszelHubURL             = "http://127.0.0.1:19999"
	beszelUserEmail          = "admin@groundseg.local"
	beszelConfigVersionLabel = "groundseg.beszel-config-version"
	beszelConfigVersion      = "1"
)

var (
	bundledBeszelAgentVersion = structs.VersionDetails{
		Amd64Sha256: "a45ca92579eca9b5e7304e2f5982de3c3e6325151e48dbc59e4fa0c5a28d55a1",
		Arm64Sha256: "d4b6f8811850a3d59cebc23d4f506bf05bdfcf42192e084609b8535d3ee0940d",
		Repo:        "registry.hub.docker.com/henrygd/beszel-agent",
		Tag:         "0.18.7",
	}
	beszelLoadMu       sync.Mutex
	beszelRuntimeState = struct {
		sync.RWMutex
		beszelRuntime
	}{}
)

type beszelRuntime struct {
	password string
	key      string
	token    string
}

func LoadBeszel() error {
	beszelLoadMu.Lock()
	defer beszelLoadMu.Unlock()

	password, err := randomBeszelSecret()
	if err != nil {
		return fmt.Errorf("generate Beszel bootstrap password: %w", err)
	}
	token, err := randomBeszelSecret()
	if err != nil {
		return fmt.Errorf("generate Beszel registration token: %w", err)
	}
	setBeszelRuntime(beszelRuntime{password: password})
	if err := config.UpdateNetdataConf(); err != nil {
		return fmt.Errorf("update netdata settings slot for Beszel: %w", err)
	}

	zap.L().Info("Loading Beszel Hub container in netdata slot")
	hubState, err := StartContainer(beszelHubContainerName, "netdata")
	if err != nil {
		return fmt.Errorf("start Beszel Hub: %w", err)
	}
	config.UpdateContainerState(beszelHubContainerName, hubState)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	key, err := bootstrapBeszel(ctx, beszelHubURL, token)
	if err != nil {
		return fmt.Errorf("bootstrap Beszel Hub: %w", err)
	}
	setBeszelRuntime(beszelRuntime{password: password, key: key, token: token})

	existingAgent, err := FindContainer(beszelAgentContainerName)
	if err != nil {
		return fmt.Errorf("find Beszel Agent container: %w", err)
	}
	if existingAgent != nil {
		if err := DeleteContainer(beszelAgentContainerName); err != nil {
			return fmt.Errorf("replace Beszel Agent container: %w", err)
		}
	}

	zap.L().Info("Loading Beszel Agent container")
	agentState, err := StartContainer(beszelAgentContainerName, "beszel-agent")
	if err != nil {
		return fmt.Errorf("start Beszel Agent: %w", err)
	}
	config.UpdateContainerState(beszelAgentContainerName, agentState)
	return nil
}

func beszelHubContainerConf(password string) (container.Config, container.HostConfig, error) {
	if password == "" {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("Beszel bootstrap password is empty")
	}
	image, err := beszelImageRef("netdata")
	if err != nil {
		return container.Config{}, container.HostConfig{}, err
	}

	containerConfig := container.Config{
		Image: image,
		Env: []string{
			"AUTO_LOGIN=" + beszelUserEmail,
			"USER_EMAIL=" + beszelUserEmail,
			"USER_PASSWORD=" + password,
		},
		ExposedPorts: nat.PortSet{"8090/tcp": struct{}{}},
		Volumes:      map[string]struct{}{beszelHubDataPath: struct{}{}},
		Labels:       map[string]string{beszelConfigVersionLabel: beszelConfigVersion},
	}
	hostConfig := container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		PortBindings: nat.PortMap{
			"8090/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "19999"}},
		},
		Binds: []string{"beszel_data:" + beszelHubDataPath},
	}
	return containerConfig, hostConfig, nil
}

const beszelHubDataPath = "/beszel_data"

func beszelAgentContainerConf(key string, token string) (container.Config, container.HostConfig, error) {
	if key == "" || token == "" {
		return container.Config{}, container.HostConfig{}, fmt.Errorf("Beszel Agent credentials are incomplete")
	}
	image, err := beszelImageRef("beszel-agent")
	if err != nil {
		return container.Config{}, container.HostConfig{}, err
	}

	containerConfig := container.Config{
		Image: image,
		Env: []string{
			"HUB_URL=" + beszelHubURL,
			"KEY=" + key,
			"TOKEN=" + token,
			"SYSTEM_NAME=GroundSeg",
			"DISABLE_SSH=true",
			"DATA_DIR=/var/lib/beszel-agent",
		},
		Volumes: map[string]struct{}{
			"/var/lib/beszel-agent": {},
			"/var/run/docker.sock":  {},
		},
		Labels: map[string]string{beszelConfigVersionLabel: beszelConfigVersion},
	}
	hostConfig := container.HostConfig{
		NetworkMode:   container.NetworkMode("host"),
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
		Binds: []string{
			"beszel_agent_data:/var/lib/beszel-agent",
			"/var/run/docker.sock:/var/run/docker.sock:ro",
		},
	}
	return containerConfig, hostConfig, nil
}

func beszelImageRef(containerType string) (string, error) {
	info, err := GetLatestContainerInfo(containerType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s@sha256:%s", info["repo"], info["tag"], info["hash"]), nil
}

func bootstrapBeszel(ctx context.Context, hubURL string, token string) (string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		key, err := fetchBeszelKey(ctx, client, hubURL)
		if err == nil {
			err = enableBeszelToken(ctx, client, hubURL, token)
			if err == nil {
				return key, nil
			}
		}
		lastErr = err

		select {
		case <-ctx.Done():
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			return "", lastErr
		case <-ticker.C:
		}
	}
}

func fetchBeszelKey(ctx context.Context, client *http.Client, hubURL string) (string, error) {
	var response struct {
		Key string `json:"key"`
	}
	if err := getBeszelJSON(ctx, client, hubURL+"/api/beszel/info", &response); err != nil {
		return "", err
	}
	if response.Key == "" {
		return "", fmt.Errorf("Beszel Hub returned an empty public key")
	}
	return response.Key, nil
}

func enableBeszelToken(ctx context.Context, client *http.Client, hubURL string, token string) error {
	query := url.Values{
		"enable":    {"1"},
		"permanent": {"1"},
		"token":     {token},
	}
	var response struct {
		Active    bool   `json:"active"`
		Permanent bool   `json:"permanent"`
		Token     string `json:"token"`
	}
	endpoint := hubURL + "/api/beszel/universal-token?" + query.Encode()
	if err := getBeszelJSON(ctx, client, endpoint, &response); err != nil {
		return err
	}
	if !response.Active || !response.Permanent || response.Token != token {
		return fmt.Errorf("Beszel Hub did not activate the permanent registration token")
	}
	return nil
}

func getBeszelJSON(ctx context.Context, client *http.Client, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", endpoint, response.Status)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode %s response: %w", endpoint, err)
	}
	return nil
}

func randomBeszelSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func setBeszelRuntime(runtime beszelRuntime) {
	beszelRuntimeState.Lock()
	beszelRuntimeState.beszelRuntime = runtime
	beszelRuntimeState.Unlock()
}

func currentBeszelRuntime() beszelRuntime {
	beszelRuntimeState.RLock()
	defer beszelRuntimeState.RUnlock()
	return beszelRuntimeState.beszelRuntime
}
