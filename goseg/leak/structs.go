package leak

import (
	"net"
	"sync"
)

type LickStatus struct {
	Conn    net.Conn
	Symlink string
	Auth    bool
	Mu      sync.RWMutex
}
