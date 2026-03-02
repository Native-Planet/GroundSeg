package session

import (
	"github.com/gorilla/websocket"
	"sync"
)

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
