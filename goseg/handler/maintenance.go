package handler

import (
	"fmt"
	"sync"
)

var shipMaintenance = struct {
	sync.Mutex
	active map[string]string
}{
	active: make(map[string]string),
}

func beginShipMaintenance(patp, action string) (func(), error) {
	shipMaintenance.Lock()
	defer shipMaintenance.Unlock()
	if current, exists := shipMaintenance.active[patp]; exists {
		return nil, fmt.Errorf("%s already has %s maintenance running", patp, current)
	}
	shipMaintenance.active[patp] = action
	return func() {
		shipMaintenance.Lock()
		delete(shipMaintenance.active, patp)
		shipMaintenance.Unlock()
	}, nil
}
