package structs

import "groundseg/session"

var (
	WsEventBus      = session.WsEventBus
	InactiveSession = session.InactiveSession
)

type ClientManager = session.ClientManager
type MuConn = session.MuConn
type WsChanEvent = session.WsChanEvent
