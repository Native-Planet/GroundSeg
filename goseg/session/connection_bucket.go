package session

import "github.com/gorilla/websocket"

func appendIfConnMissing[T any](existing []T, candidate T, connFor func(T) *websocket.Conn) []T {
	if connFor(candidate) == nil {
		return existing
	}
	for _, item := range existing {
		if connFor(item) == nil {
			continue
		}
		if connFor(item) == connFor(candidate) {
			return existing
		}
	}
	return append(existing, candidate)
}

func hasConn[T any](items []T, conn *websocket.Conn, connFor func(T) *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	for _, item := range items {
		if itemConn := connFor(item); itemConn != nil && itemConn == conn {
			return true
		}
	}
	return false
}

func hasConnInBuckets[T any](buckets map[string][]T, conn *websocket.Conn, connFor func(T) *websocket.Conn) bool {
	for _, items := range buckets {
		if hasConn(items, conn, connFor) {
			return true
		}
	}
	return false
}

func removeConn[T any](items []T, conn *websocket.Conn, connFor func(T) *websocket.Conn) ([]T, bool) {
	if len(items) == 0 || conn == nil {
		return items, false
	}
	filtered := items[:0]
	removed := false
	for _, item := range items {
		if itemConn := connFor(item); itemConn == conn {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered, removed
}

func removeConnFromBuckets[T any](buckets map[string][]T, conn *websocket.Conn, connFor func(T) *websocket.Conn) bool {
	if conn == nil {
		return false
	}
	removedAny := false
	for token, items := range buckets {
		filtered, removed := removeConn(items, conn, connFor)
		if !removed {
			continue
		}
		removedAny = true
		if len(filtered) == 0 {
			delete(buckets, token)
			continue
		}
		buckets[token] = filtered
	}
	return removedAny
}

func snapshotBuckets[T any](itemsByToken map[string][]T, cloneItem func(T) T) map[string][]T {
	if itemsByToken == nil {
		return nil
	}
	snapshot := make(map[string][]T, len(itemsByToken))
	for token, items := range itemsByToken {
		copyItems := make([]T, len(items))
		for i, item := range items {
			copyItems[i] = cloneItem(item)
		}
		snapshot[token] = copyItems
	}
	return snapshot
}

func snapshotSlice[T any](items []T, cloneItem func(T) T) []T {
	copyItems := make([]T, len(items))
	for i, item := range items {
		copyItems[i] = cloneItem(item)
	}
	return copyItems
}

func clonePtrConn(conn *websocket.Conn) *websocket.Conn {
	return conn
}
