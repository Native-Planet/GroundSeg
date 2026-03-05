package containerstate

import (
	"sync"

	"groundseg/structs"
)

var (
	mu         sync.RWMutex
	containers = make(map[string]structs.ContainerState)
)

func Update(name string, state structs.ContainerState) {
	mu.Lock()
	containers[name] = state
	mu.Unlock()
}

func Delete(name string) {
	mu.Lock()
	delete(containers, name)
	mu.Unlock()
}

func Snapshot() map[string]structs.ContainerState {
	mu.RLock()
	defer mu.RUnlock()
	copyState := make(map[string]structs.ContainerState, len(containers))
	for name, state := range containers {
		copyState[name] = state
	}
	return copyState
}
