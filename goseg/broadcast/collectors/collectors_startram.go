package collectors

import (
	"fmt"
	"go.uber.org/zap"
	"groundseg/startram"
	"groundseg/structs"
	"groundseg/transition"
)

func LoadStartramRegions() (map[string]structs.StartramRegion, error) {
	regions, err := startram.SyncRegions()
	if err != nil {
		return nil, fmt.Errorf("loading startram regions: %w", err)
	}
	return regions, nil
}

func GetStartramServices() error {
	return getStartramServices(defaultCollectorRuntime())
}

func getStartramServices(runtime collectorRuntime) error {
	zap.L().Info("Retrieving StarTram services info")
	res, err := runtime.StartramServicesRetrieverFn()
	if err != nil {
		return fmt.Errorf("retrieve startram services: %w", err)
	}
	zap.L().Info(fmt.Sprintf("%+v", res.Subdomains))
	return nil
}

type pierStartramSnapshot struct {
	remoteReadyByURL map[string]bool
}

func startramSnapshotForPiers(subdomains []structs.Subdomain) pierStartramSnapshot {
	readyByURL := make(map[string]bool, len(subdomains))
	for _, subdomain := range subdomains {
		readyByURL[subdomain.URL] = subdomain.Status == string(transition.StartramServiceStatusOk)
	}
	return pierStartramSnapshot{
		remoteReadyByURL: readyByURL,
	}
}
