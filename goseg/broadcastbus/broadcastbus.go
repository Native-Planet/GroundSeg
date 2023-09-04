package broadcastbus

import "goseg/structs"

var (
	BroadcastBus = make(chan structs.Event, 100)
)
