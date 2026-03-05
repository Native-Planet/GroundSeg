package defaults

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"groundseg/structs"
	"sync"
)

//go:embed version_defaults.json
var defaultVersionJSON []byte

type VersionDefaultsProvider interface {
	VersionDefaults() (structs.Version, error)
}

type staticVersionDefaultsProvider struct {
	version  structs.Version
	parseErr error
}

func (provider staticVersionDefaultsProvider) VersionDefaults() (structs.Version, error) {
	return cloneVersion(provider.version), provider.parseErr
}

var (
	versionDefaultsParseErr error
	versionDefaultsMu       sync.RWMutex
	versionDefaultsProv     VersionDefaultsProvider
)

var defaultVersionDefaults structs.Version

func init() {
	parsed, err := parseDefaultVersionMetadata(defaultVersionJSON)
	if err != nil {
		versionDefaultsParseErr = fmt.Errorf("error loading default version metadata: %w", err)
	}
	defaultVersionDefaults = parsed
	versionDefaultsProv = staticVersionDefaultsProvider{
		version:  parsed,
		parseErr: versionDefaultsParseErr,
	}
}

func parseDefaultVersionMetadata(rawJSON []byte) (structs.Version, error) {
	var version structs.Version
	if len(rawJSON) == 0 {
		return structs.Version{}, fmt.Errorf("no embedded default version metadata available")
	}
	if err := json.Unmarshal(rawJSON, &version); err != nil {
		return structs.Version{}, err
	}
	return version, nil
}

func cloneVersion(input structs.Version) structs.Version {
	if len(input.Groundseg) == 0 {
		return structs.Version{Groundseg: map[string]structs.Channel{}}
	}
	out := structs.Version{Groundseg: make(map[string]structs.Channel, len(input.Groundseg))}
	for channel, values := range input.Groundseg {
		out.Groundseg[channel] = values
	}
	return out
}

func SetVersionDefaultsProvider(provider VersionDefaultsProvider) {
	versionDefaultsMu.Lock()
	defer versionDefaultsMu.Unlock()
	if provider == nil {
		provider = staticVersionDefaultsProvider{
			version:  defaultVersionDefaults,
			parseErr: versionDefaultsParseErr,
		}
	}
	versionDefaultsProv = provider
}

func DefaultVersionParseError() error {
	return versionDefaultsParseErr
}

func DefaultVersionDefaults() (structs.Version, error) {
	versionDefaultsMu.RLock()
	provider := versionDefaultsProv
	versionDefaultsMu.RUnlock()
	if provider == nil {
		return structs.Version{}, fmt.Errorf("default version provider is not configured")
	}
	return provider.VersionDefaults()
}
