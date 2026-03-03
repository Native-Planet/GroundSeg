package session

import "sync"

type sessionRuntime struct {
	mu             sync.RWMutex
	clientManager  *ClientManager
	logstreamState LogstreamRuntime
}

func newSessionRuntime() *sessionRuntime {
	return &sessionRuntime{
		clientManager:  NewClientManager(),
		logstreamState: newLogstreamSessionStore(),
	}
}

var defaultSessionRuntime = newSessionRuntime()

var logstreamRuntimeState struct {
	sync.RWMutex
	active LogstreamRuntime
}

func init() {
	logstreamRuntimeState.Lock()
	logstreamRuntimeState.active = NewLogstreamRuntime()
	logstreamRuntimeState.Unlock()
}

func NewLogstreamRuntime() LogstreamRuntime {
	return snapshotLogstreamRuntime()
}

func snapshotLogstreamRuntime() LogstreamRuntime {
	defaultSessionRuntime.mu.RLock()
	defer defaultSessionRuntime.mu.RUnlock()
	return defaultSessionRuntime.logstreamState
}

func LogstreamRuntimeState() LogstreamRuntime {
	return snapshotLogstreamRuntime()
}

func SetLogstreamRuntime(rt LogstreamRuntime) {
	if rt == nil {
		rt = NewLogstreamRuntime()
	}
	defaultSessionRuntime.mu.Lock()
	defer defaultSessionRuntime.mu.Unlock()
	defaultSessionRuntime.logstreamState = rt
}

func NewClientManager() *ClientManager {
	cm := &ClientManager{}
	cm.ensureState()
	return cm
}

func GetClientManager() *ClientManager {
	defaultSessionRuntime.mu.Lock()
	defer defaultSessionRuntime.mu.Unlock()
	if defaultSessionRuntime.clientManager == nil {
		defaultSessionRuntime.clientManager = NewClientManager()
	}
	return defaultSessionRuntime.clientManager
}

func SetClientManager(cm *ClientManager) {
	defaultSessionRuntime.mu.Lock()
	defer defaultSessionRuntime.mu.Unlock()
	if cm == nil {
		defaultSessionRuntime.clientManager = NewClientManager()
		return
	}
	defaultSessionRuntime.clientManager = cm
}
