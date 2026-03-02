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

type logstreamRuntime struct {
	sysLogMessages chan []byte
}

func (_ *logstreamRuntime) SysLogSessions() []*websocket.Conn {
	return SysLogSessions()
}

func (_ *logstreamRuntime) SetSysLogSessions(sessions []*websocket.Conn) {
	SetSysLogSessions(sessions)
}

func (_ *logstreamRuntime) SysSessionsToRemove() []*websocket.Conn {
	return SysSessionsToRemove()
}

func (_ *logstreamRuntime) SetSysSessionsToRemove(sessions []*websocket.Conn) {
	SetSysSessionsToRemove(sessions)
}

func (_ *logstreamRuntime) AddSysLogSession(conn *websocket.Conn) {
	AddSysLogSession(conn)
}

func (_ *logstreamRuntime) RemoveSysLogSessions() {
	RemoveSysLogSessions()
}

func (_ *logstreamRuntime) AddSysSessionToRemove(conn *websocket.Conn) {
	AddSysSessionToRemove(conn)
}

func (_ *logstreamRuntime) DockerLogSessions() map[string]map[*websocket.Conn]bool {
	return DockerLogSessions()
}

func (_ *logstreamRuntime) SetDockerLogSessions(sessions map[string]map[*websocket.Conn]bool) {
	SetDockerLogSessions(sessions)
}

func (_ *logstreamRuntime) SetDockerLogSession(container string, conn *websocket.Conn, live bool) {
	SetDockerLogSession(container, conn, live)
}

func (_ *logstreamRuntime) SetDockerLogSessionLive(container string, conn *websocket.Conn, live bool) {
	SetDockerLogSessionLive(container, conn, live)
}

func (_ *logstreamRuntime) RemoveDockerLogSession(container string, conn *websocket.Conn) {
	RemoveDockerLogSession(container, conn)
}

func (runtime *logstreamRuntime) SystemLogMessages() <-chan []byte {
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
	return &logstreamRuntime{
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
		return
	}
	logstreamRuntimeState.Lock()
	defer logstreamRuntimeState.Unlock()
	logstreamRuntimeState.active = rt
}

type sessionState struct {
	authMu        sync.RWMutex
	clientManager *ClientManager

	sysLogMu           sync.RWMutex
	sysLogSessions     []*websocket.Conn
	sysSessionsToClear []*websocket.Conn

	dockerLogMu       sync.RWMutex
	dockerLogSessions map[string]map[*websocket.Conn]bool
}

var (
	stateMu  sync.Mutex
	activeMu = &stateMu
	active   = newSessionState()
)

func newSessionState() *sessionState {
	return &sessionState{
		clientManager:     NewClientManager(),
		dockerLogSessions: make(map[string]map[*websocket.Conn]bool),
	}
}

func snapshot() *sessionState {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		AuthClients:   make(map[string][]*MuConn),
		UnauthClients: make(map[string][]*MuConn),
	}
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

func SysLogSessions() []*websocket.Conn {
	s := snapshot()
	s.sysLogMu.RLock()
	defer s.sysLogMu.RUnlock()
	result := make([]*websocket.Conn, len(s.sysLogSessions))
	copy(result, s.sysLogSessions)
	return result
}

func SetSysLogSessions(sessions []*websocket.Conn) {
	s := snapshot()
	s.sysLogMu.Lock()
	defer s.sysLogMu.Unlock()
	if sessions == nil {
		s.sysLogSessions = nil
		return
	}
	cloned := make([]*websocket.Conn, len(sessions))
	copy(cloned, sessions)
	s.sysLogSessions = cloned
}

func SysSessionsToRemove() []*websocket.Conn {
	s := snapshot()
	s.sysLogMu.RLock()
	defer s.sysLogMu.RUnlock()
	result := make([]*websocket.Conn, len(s.sysSessionsToClear))
	copy(result, s.sysSessionsToClear)
	return result
}

func SetSysSessionsToRemove(sessions []*websocket.Conn) {
	s := snapshot()
	s.sysLogMu.Lock()
	defer s.sysLogMu.Unlock()
	if sessions == nil {
		s.sysSessionsToClear = nil
		return
	}
	cloned := make([]*websocket.Conn, len(sessions))
	copy(cloned, sessions)
	s.sysSessionsToClear = cloned
}

func AddSysLogSession(conn *websocket.Conn) {
	s := snapshot()
	s.sysLogMu.Lock()
	defer s.sysLogMu.Unlock()
	s.sysLogSessions = append(s.sysLogSessions, conn)
}

func RemoveSysLogSessions() {
	s := snapshot()
	s.sysLogMu.Lock()
	defer s.sysLogMu.Unlock()
	if len(s.sysSessionsToClear) == 0 {
		s.sysSessionsToClear = nil
		return
	}
	remove := make(map[*websocket.Conn]struct{}, len(s.sysSessionsToClear))
	for _, session := range s.sysSessionsToClear {
		remove[session] = struct{}{}
	}
	remaining := make([]*websocket.Conn, 0, len(s.sysLogSessions))
	for _, session := range s.sysLogSessions {
		if _, found := remove[session]; found {
			continue
		}
		remaining = append(remaining, session)
	}
	s.sysLogSessions = remaining
	s.sysSessionsToClear = nil
}

func AddSysSessionToRemove(conn *websocket.Conn) {
	s := snapshot()
	s.sysLogMu.Lock()
	defer s.sysLogMu.Unlock()
	s.sysSessionsToClear = append(s.sysSessionsToClear, conn)
}

func DockerLogSessions() map[string]map[*websocket.Conn]bool {
	s := snapshot()
	s.dockerLogMu.RLock()
	defer s.dockerLogMu.RUnlock()
	copyMap := make(map[string]map[*websocket.Conn]bool, len(s.dockerLogSessions))
	for container, sessions := range s.dockerLogSessions {
		containerCopy := make(map[*websocket.Conn]bool, len(sessions))
		for conn, live := range sessions {
			containerCopy[conn] = live
		}
		copyMap[container] = containerCopy
	}
	return copyMap
}

func SetDockerLogSessions(sessions map[string]map[*websocket.Conn]bool) {
	s := snapshot()
	s.dockerLogMu.Lock()
	defer s.dockerLogMu.Unlock()
	copyMap := make(map[string]map[*websocket.Conn]bool, len(sessions))
	for container, containerSessions := range sessions {
		containerCopy := make(map[*websocket.Conn]bool, len(containerSessions))
		for conn, live := range containerSessions {
			containerCopy[conn] = live
		}
		copyMap[container] = containerCopy
	}
	s.dockerLogSessions = copyMap
}

func SetDockerLogSession(container string, conn *websocket.Conn, live bool) {
	s := snapshot()
	s.dockerLogMu.Lock()
	defer s.dockerLogMu.Unlock()
	if _, ok := s.dockerLogSessions[container]; !ok {
		s.dockerLogSessions[container] = make(map[*websocket.Conn]bool)
	}
	s.dockerLogSessions[container][conn] = live
}

func RemoveDockerLogSession(container string, conn *websocket.Conn) {
	s := snapshot()
	s.dockerLogMu.Lock()
	defer s.dockerLogMu.Unlock()
	if _, ok := s.dockerLogSessions[container]; !ok {
		return
	}
	delete(s.dockerLogSessions[container], conn)
	if len(s.dockerLogSessions[container]) == 0 {
		delete(s.dockerLogSessions, container)
	}
}

func SetDockerLogSessionLive(container string, conn *websocket.Conn, live bool) {
	s := snapshot()
	s.dockerLogMu.Lock()
	defer s.dockerLogMu.Unlock()
	if sessionMap, ok := s.dockerLogSessions[container]; ok {
		sessionMap[conn] = live
	}
}
