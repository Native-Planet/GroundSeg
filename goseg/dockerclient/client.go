package dockerclient

import (
	"context"
	"sync"
	"time"

	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

const pingTimeout = 5 * time.Second

var (
	versionMu       sync.RWMutex
	cachedVersion   string
	versionDetected bool
)

// New returns a Docker client that pings the daemon once to discover the API
// version the server expects. Subsequent calls reuse the cached version so we
// don't renegotiate every time we touch the Docker daemon.
func New(extraOpts ...client.Opt) (*client.Client, error) {
	apiVersion, err := getAPIVersion()
	if err != nil {
		return nil, err
	}

	opts := []client.Opt{client.FromEnv}
	if apiVersion == "" {
		opts = append(opts, client.WithAPIVersionNegotiation())
	} else {
		opts = append(opts, client.WithVersion(apiVersion))
	}
	opts = append(opts, extraOpts...)
	return client.NewClientWithOpts(opts...)
}

func getAPIVersion() (string, error) {
	versionMu.RLock()
	if versionDetected {
		defer versionMu.RUnlock()
		return cachedVersion, nil
	}
	versionMu.RUnlock()

	versionMu.Lock()
	defer versionMu.Unlock()
	if versionDetected {
		return cachedVersion, nil
	}

	version, err := detectAPIVersion()
	if err != nil {
		return "", err
	}
	cachedVersion = version
	versionDetected = true
	return cachedVersion, nil
}

func detectAPIVersion() (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()

	ping, err := cli.Ping(ctx)
	if err != nil {
		zap.L().Warn("Failed to ping Docker daemon while discovering API version; falling back to negotiation", zap.Error(err))
		return "", nil
	}
	if ping.APIVersion == "" {
		zap.L().Warn("Docker daemon did not report an API version; falling back to negotiation")
		return "", nil
	}

	clientVersion := api.DefaultVersion
	switch {
	case versions.GreaterThan(ping.APIVersion, clientVersion):
		zap.L().Info("Docker daemon requires newer API version than library default",
			zap.String("client_api_version", clientVersion),
			zap.String("daemon_api_version", ping.APIVersion))
	case versions.LessThan(ping.APIVersion, clientVersion):
		zap.L().Debug("Docker daemon API version is older than client default",
			zap.String("client_api_version", clientVersion),
			zap.String("daemon_api_version", ping.APIVersion))
	default:
		zap.L().Debug("Docker daemon API version matches client default",
			zap.String("api_version", clientVersion))
	}

	return ping.APIVersion, nil
}
