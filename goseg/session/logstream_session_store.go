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
	sysLogMessages     chan []byte

	dockerLogMu       sync.RWMutex
	dockerLogSessions map[string]map[*websocket.Conn]bool
}

func newLogstreamSessionStore() *logstreamSessionStore {
	return &logstreamSessionStore{
		dockerLogSessions: make(map[string]map[*websocket.Conn]bool),
		sysLogMessages:    make(chan []byte, 100),
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

func (store *logstreamSessionStore) SystemLogMessages() <-chan []byte {
	if store == nil {
		return nil
	}
	store.sysLogMu.RLock()
	defer store.sysLogMu.RUnlock()
	return store.sysLogMessages
}

func (store *logstreamSessionStore) PublishSystemLog(logData []byte) {
	if store == nil || store.sysLogMessages == nil {
		return
	}
	store.sysLogMu.Lock()
	messageBus := store.sysLogMessages
	store.sysLogMu.Unlock()
	messageBus <- logData
}
