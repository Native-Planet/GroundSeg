package session

import (
	"github.com/gorilla/websocket"
	"sync"
)

type LogstreamRuntime interface {
	SysLogSessions() []*websocket.Conn
	SetSysLogSessions([]*websocket.Conn)
	SysSessionsToRemove() []*websocket.Conn
	SetSysSessionsToRemove([]*websocket.Conn)
	AddSysLogSession(*websocket.Conn)
	RemoveSysLogSessions()
	AddSysSessionToRemove(*websocket.Conn)
	DockerLogSessions() map[string]map[*websocket.Conn]bool
	SetDockerLogSessions(map[string]map[*websocket.Conn]bool)
	SetDockerLogSession(string, *websocket.Conn, bool)
	SetDockerLogSessionLive(string, *websocket.Conn, bool)
	RemoveDockerLogSession(string, *websocket.Conn)
	SystemLogMessages() <-chan []byte
	PublishSystemLog([]byte)
}

type logstreamSessionStore struct {
	sysLogMu           sync.RWMutex
	sysLogSessions     []*websocket.Conn
	sysSessionsToClear []*websocket.Conn

	dockerLogMu       sync.RWMutex
	dockerLogSessions map[string]map[*websocket.Conn]bool
}

func newLogstreamSessionStore() *logstreamSessionStore {
	return &logstreamSessionStore{
		dockerLogSessions: make(map[string]map[*websocket.Conn]bool),
	}
}

func (store *logstreamSessionStore) SysLogSessions() []*websocket.Conn {
	if store == nil {
		return nil
	}
	store.sysLogMu.RLock()
	defer store.sysLogMu.RUnlock()
	result := make([]*websocket.Conn, len(store.sysLogSessions))
	copy(result, store.sysLogSessions)
	return result
}

func (store *logstreamSessionStore) SetSysLogSessions(sessions []*websocket.Conn) {
	if store == nil {
		return
	}
	store.sysLogMu.Lock()
	defer store.sysLogMu.Unlock()
	if sessions == nil {
		store.sysLogSessions = nil
		return
	}
	store.sysLogSessions = append([]*websocket.Conn(nil), sessions...)
}

func (store *logstreamSessionStore) SysSessionsToRemove() []*websocket.Conn {
	if store == nil {
		return nil
	}
	store.sysLogMu.RLock()
	defer store.sysLogMu.RUnlock()
	result := make([]*websocket.Conn, len(store.sysSessionsToClear))
	copy(result, store.sysSessionsToClear)
	return result
}

func (store *logstreamSessionStore) SetSysSessionsToRemove(sessions []*websocket.Conn) {
	if store == nil {
		return
	}
	store.sysLogMu.Lock()
	defer store.sysLogMu.Unlock()
	if sessions == nil {
		store.sysSessionsToClear = nil
		return
	}
	store.sysSessionsToClear = append([]*websocket.Conn(nil), sessions...)
}

func (store *logstreamSessionStore) AddSysLogSession(conn *websocket.Conn) {
	if store == nil {
		return
	}
	store.sysLogMu.Lock()
	defer store.sysLogMu.Unlock()
	store.sysLogSessions = append(store.sysLogSessions, conn)
}

func (store *logstreamSessionStore) RemoveSysLogSessions() {
	if store == nil {
		return
	}
	store.sysLogMu.Lock()
	defer store.sysLogMu.Unlock()
	if len(store.sysSessionsToClear) == 0 {
		store.sysSessionsToClear = nil
		return
	}
	remove := make(map[*websocket.Conn]struct{}, len(store.sysSessionsToClear))
	for _, session := range store.sysSessionsToClear {
		remove[session] = struct{}{}
	}
	remaining := make([]*websocket.Conn, 0, len(store.sysLogSessions))
	for _, session := range store.sysLogSessions {
		if _, found := remove[session]; found {
			continue
		}
		remaining = append(remaining, session)
	}
	store.sysLogSessions = remaining
	store.sysSessionsToClear = nil
}

func (store *logstreamSessionStore) AddSysSessionToRemove(conn *websocket.Conn) {
	if store == nil {
		return
	}
	store.sysLogMu.Lock()
	defer store.sysLogMu.Unlock()
	store.sysSessionsToClear = append(store.sysSessionsToClear, conn)
}

func (store *logstreamSessionStore) DockerLogSessions() map[string]map[*websocket.Conn]bool {
	if store == nil {
		return nil
	}
	store.dockerLogMu.RLock()
	defer store.dockerLogMu.RUnlock()
	copyMap := make(map[string]map[*websocket.Conn]bool, len(store.dockerLogSessions))
	for container, sessions := range store.dockerLogSessions {
		containerCopy := make(map[*websocket.Conn]bool, len(sessions))
		for conn, live := range sessions {
			containerCopy[conn] = live
		}
		copyMap[container] = containerCopy
	}
	return copyMap
}

func (store *logstreamSessionStore) SetDockerLogSessions(sessions map[string]map[*websocket.Conn]bool) {
	if store == nil {
		return
	}
	store.dockerLogMu.Lock()
	defer store.dockerLogMu.Unlock()
	copyMap := make(map[string]map[*websocket.Conn]bool, len(sessions))
	for container, containerSessions := range sessions {
		containerCopy := make(map[*websocket.Conn]bool, len(containerSessions))
		for conn, live := range containerSessions {
			containerCopy[conn] = live
		}
		copyMap[container] = containerCopy
	}
	store.dockerLogSessions = copyMap
}

func (store *logstreamSessionStore) SetDockerLogSession(container string, conn *websocket.Conn, live bool) {
	if store == nil {
		return
	}
	store.dockerLogMu.Lock()
	defer store.dockerLogMu.Unlock()
	if _, ok := store.dockerLogSessions[container]; !ok {
		store.dockerLogSessions[container] = make(map[*websocket.Conn]bool)
	}
	store.dockerLogSessions[container][conn] = live
}

func (store *logstreamSessionStore) SetDockerLogSessionLive(container string, conn *websocket.Conn, live bool) {
	if store == nil {
		return
	}
	store.dockerLogMu.Lock()
	defer store.dockerLogMu.Unlock()
	if sessionMap, ok := store.dockerLogSessions[container]; ok {
		sessionMap[conn] = live
	}
}

func (store *logstreamSessionStore) RemoveDockerLogSession(container string, conn *websocket.Conn) {
	if store == nil {
		return
	}
	store.dockerLogMu.Lock()
	defer store.dockerLogMu.Unlock()
	if _, ok := store.dockerLogSessions[container]; !ok {
		return
	}
	delete(store.dockerLogSessions[container], conn)
	if len(store.dockerLogSessions[container]) == 0 {
		delete(store.dockerLogSessions, container)
	}
}

type logstreamRuntime struct {
	state          *sessionState
	sysLogMessages chan []byte
}

func (runtime *logstreamRuntime) stateForWrite() *sessionState {
	if runtime == nil || runtime.state == nil {
		return nil
	}
	return runtime.state
}

func (runtime *logstreamRuntime) stateForRead() *sessionState {
	if runtime == nil || runtime.state == nil {
		return nil
	}
	return runtime.state
}

func (runtime *logstreamRuntime) SysLogSessions() []*websocket.Conn {
	state := runtime.stateForRead()
	if state == nil || state.logstream == nil {
		return nil
	}
	return state.logstream.SysLogSessions()
}

func (runtime *logstreamRuntime) SetSysLogSessions(sessions []*websocket.Conn) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.SetSysLogSessions(sessions)
}

func (runtime *logstreamRuntime) SysSessionsToRemove() []*websocket.Conn {
	state := runtime.stateForRead()
	if state == nil || state.logstream == nil {
		return nil
	}
	return state.logstream.SysSessionsToRemove()
}

func (runtime *logstreamRuntime) SetSysSessionsToRemove(sessions []*websocket.Conn) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.SetSysSessionsToRemove(sessions)
}

func (runtime *logstreamRuntime) AddSysLogSession(conn *websocket.Conn) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.AddSysLogSession(conn)
}

func (runtime *logstreamRuntime) RemoveSysLogSessions() {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.RemoveSysLogSessions()
}

func (runtime *logstreamRuntime) AddSysSessionToRemove(conn *websocket.Conn) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.AddSysSessionToRemove(conn)
}

func (runtime *logstreamRuntime) DockerLogSessions() map[string]map[*websocket.Conn]bool {
	state := runtime.stateForRead()
	if state == nil || state.logstream == nil {
		return nil
	}
	return state.logstream.DockerLogSessions()
}

func (runtime *logstreamRuntime) SetDockerLogSessions(sessions map[string]map[*websocket.Conn]bool) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.SetDockerLogSessions(sessions)
}

func (runtime *logstreamRuntime) SetDockerLogSession(container string, conn *websocket.Conn, live bool) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.SetDockerLogSession(container, conn, live)
}

func (runtime *logstreamRuntime) SetDockerLogSessionLive(container string, conn *websocket.Conn, live bool) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.SetDockerLogSessionLive(container, conn, live)
}

func (runtime *logstreamRuntime) RemoveDockerLogSession(container string, conn *websocket.Conn) {
	state := runtime.stateForWrite()
	if state == nil || state.logstream == nil {
		return
	}
	state.logstream.RemoveDockerLogSession(container, conn)
}

func (runtime *logstreamRuntime) SystemLogMessages() <-chan []byte {
	if runtime == nil {
		return nil
	}
	return runtime.sysLogMessages
}

func (runtime *logstreamRuntime) PublishSystemLog(logData []byte) {
	if runtime == nil || runtime.sysLogMessages == nil {
		return
	}
	runtime.sysLogMessages <- logData
}

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
	state := snapshot()
	return &logstreamRuntime{
		state:          state,
		sysLogMessages: make(chan []byte, 100),
	}
}

func LogstreamRuntimeState() LogstreamRuntime {
	logstreamRuntimeState.RLock()
	defer logstreamRuntimeState.RUnlock()
	return logstreamRuntimeState.active
}

func SetLogstreamRuntime(rt LogstreamRuntime) {
	if rt == nil {
		rt = NewLogstreamRuntime()
	}
	logstreamRuntimeState.Lock()
	defer logstreamRuntimeState.Unlock()
	logstreamRuntimeState.active = rt
}

type sessionState struct {
	authMu        sync.RWMutex
	clientManager *ClientManager

	logstream *logstreamSessionStore
}

var (
	stateMu  sync.Mutex
	activeMu = &stateMu
	active   = newSessionState()
)

func newSessionState() *sessionState {
	return &sessionState{
		clientManager: NewClientManager(),
		logstream:     newLogstreamSessionStore(),
	}
}

func snapshot() *sessionState {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

func NewClientManager() *ClientManager {
	cm := &ClientManager{}
	cm.ensureState()
	return cm
}

func GetClientManager() *ClientManager {
	s := snapshot()
	s.authMu.Lock()
	defer s.authMu.Unlock()
	return s.clientManager
}

func SetClientManager(cm *ClientManager) {
	s := snapshot()
	s.authMu.Lock()
	defer s.authMu.Unlock()
	if cm == nil {
		s.clientManager = NewClientManager()
		return
	}
	s.clientManager = cm
}
