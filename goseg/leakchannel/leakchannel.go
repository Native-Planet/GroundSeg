package leakchannel

type ActionChannel struct {
	Auth    bool
	Patp    string
	Type    string
	Content []byte
}

var (
	LeakAction = make(chan ActionChannel)
	Logout     = make(chan struct{})
)
