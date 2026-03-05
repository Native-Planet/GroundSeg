package startramstate

import (
	"sync"
	"time"

	"groundseg/structs"
)

type Snapshot struct {
	Value     structs.StartramRetrieve
	UpdatedAt time.Time
	Fresh     bool
}

var (
	mu      sync.RWMutex
	current structs.StartramRetrieve
	updated time.Time
)

func Set(retrieve structs.StartramRetrieve) {
	mu.Lock()
	defer mu.Unlock()
	current = retrieve
	updated = time.Now()
}

func Get() structs.StartramRetrieve {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func GetSnapshot() Snapshot {
	mu.RLock()
	defer mu.RUnlock()
	return Snapshot{
		Value:     current,
		UpdatedAt: updated,
		Fresh:     !updated.IsZero(),
	}
}
