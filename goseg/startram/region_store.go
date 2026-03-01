package startram

import (
	"groundseg/structs"
	"sync"
)

// RegionStore isolates mutable region cache state from package globals.
type RegionStore interface {
	Set(regions map[string]structs.StartramRegion)
	Snapshot() map[string]structs.StartramRegion
}

type inMemoryRegionStore struct {
	mu      sync.RWMutex
	regions map[string]structs.StartramRegion
}

func newInMemoryRegionStore() *inMemoryRegionStore {
	return &inMemoryRegionStore{
		regions: make(map[string]structs.StartramRegion),
	}
}

func (store *inMemoryRegionStore) Set(regions map[string]structs.StartramRegion) {
	store.mu.Lock()
	defer store.mu.Unlock()
	cloned := make(map[string]structs.StartramRegion, len(regions))
	for key, region := range regions {
		cloned[key] = region
	}
	store.regions = cloned
}

func (store *inMemoryRegionStore) Snapshot() map[string]structs.StartramRegion {
	store.mu.RLock()
	defer store.mu.RUnlock()
	cloned := make(map[string]structs.StartramRegion, len(store.regions))
	for key, region := range store.regions {
		cloned[key] = region
	}
	return cloned
}

var defaultRegionStore RegionStore = newInMemoryRegionStore()

func SetRegionStore(store RegionStore) {
	if store != nil {
		defaultRegionStore = store
	}
}

func RegionsSnapshot() map[string]structs.StartramRegion {
	return defaultRegionStore.Snapshot()
}
