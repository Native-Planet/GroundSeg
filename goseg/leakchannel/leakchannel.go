package leakchannel

type ActionChannel struct {
	Type    string
	Content []byte
}

var (
	LeakAction = make(chan ActionChannel)
)
