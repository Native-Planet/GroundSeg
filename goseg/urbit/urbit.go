package urbit

type LickSettings struct {
	Available bool
	Auth      bool
}

var (
	lickStatus = make(map[string]LickSettings)
)

func FindLickPorts() {
	// loop through conf.Piers
	// add to Lick settings if port is available
	// else remove from Lick settings
}

func BroadcastToUrbit() {
	// loop through lickStatus
	// if available
	// if authed send full broadcast
	// if !authed send only ship info
	// todo: add flood control
}
