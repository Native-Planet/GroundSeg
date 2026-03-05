package lifecycle

import (
	"context"
	"sync"
	"time"

	"groundseg/config"
	"groundseg/session"
)

var (
	authInitOnce     sync.Once
	authLifecycleMu  sync.Mutex
	authCleanupStop  context.CancelFunc
	authCleanupEvery = func() *time.Ticker { return time.NewTicker(10 * time.Minute) }
	authStaleTimeout = 30 * time.Minute
)

func Initialize() error {
	return InitializeWithContext(context.Background())
}

func InitializeWithContext(ctx context.Context) error {
	return Start(ctx)
}

func Start(ctx context.Context) error {
	authInitOnce.Do(func() {
		clientManager := session.GetClientManager()
		if clientManager == nil {
			return
		}
		settings := config.AuthSettingsSnapshot()
		for key := range settings.AuthorizedSessions {
			clientManager.AddAuthClient(key, &session.MuConn{Active: false})
		}
	})
	if ctx == nil {
		ctx = context.Background()
	}
	authLifecycleMu.Lock()
	defer authLifecycleMu.Unlock()
	if authCleanupStop != nil {
		return nil
	}
	lifecycleCtx, cancel := context.WithCancel(ctx)
	authCleanupStop = cancel
	go runAuthCleanupLoop(lifecycleCtx)
	return nil
}

func Stop() {
	authLifecycleMu.Lock()
	stop := authCleanupStop
	authCleanupStop = nil
	authLifecycleMu.Unlock()
	if stop != nil {
		stop()
	}
}

func runAuthCleanupLoop(ctx context.Context) {
	ticker := authCleanupEvery()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			session.GetClientManager().CleanupStaleSessions(authStaleTimeout)
		}
	}
}
