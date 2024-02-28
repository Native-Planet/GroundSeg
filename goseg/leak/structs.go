package leak

type LickStatus struct {
	Symlink string
	Auth    bool
}

type AuthEvent struct {
	Type        string `json:"type"`
	PayloadType string `json:"payloadType"`
	Error       string `json:"error"`
}
