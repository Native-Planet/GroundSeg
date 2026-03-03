package subsystem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"groundseg/dockerclient"
	"groundseg/logger"
	"groundseg/structs"
	"groundseg/transition"

	eventtypes "github.com/docker/docker/api/types/events"
)

var (
	eventBus           = make(chan structs.Event, 100)
	DisableShipRestart = false
	dockerHealthLoopMu sync.Mutex
	healthLoopRunning  bool
	healthLoopStopFn   context.CancelFunc

	dockerTransitionFailureLimit  = 5
	dockerTransitionFailureWindow = 3 * time.Minute
)

func StartDockerHealthLoops(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil
	}
	return startDockerHealthLoop(ctx)
}

func StartDockerSubsystem(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := startDockerHealthLoop(ctx); err != nil {
		return err
	}
	defer StopDockerHealthLoops()
	errs := make(chan error, 2)
	go func() {
		errs <- dockerListenerWithContext(ctx)
	}()
	go func() {
		errs <- dockerSubscriptionHandler(ctx)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			return err
		}
	}
}

func StopDockerHealthLoops() {
	dockerHealthLoopMu.Lock()
	defer dockerHealthLoopMu.Unlock()

	if !healthLoopRunning || healthLoopStopFn == nil {
		return
	}
	healthLoopStopFn()
	healthLoopRunning = false
	healthLoopStopFn = nil
}

// dockerListenerWithContext runs until ctx is canceled.
func dockerListenerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cli, err := dockerclient.New()
	if err != nil {
		return fmt.Errorf("initialize Docker client: %w", err)
	}
	messages, errs := cli.Events(ctx, eventtypes.ListOptions{})
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Docker event listener")
			return nil
		case event := <-messages:
			if ctx.Err() != nil {
				return nil
			}
			// Convert the Docker event to our custom event and send it to the EventBus.
			select {
			case eventBus <- structs.Event{Type: string(event.Action), Data: event}:
			case <-ctx.Done():
				return nil
			}
		case err := <-errs:
			if err == nil {
				if ctx.Err() != nil {
					return nil
				}
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("docker event error: %w", err)
		}
	}
}

func dockerSubscriptionHandler(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	rt := newDockerRoutineRuntime()
	failureCount := 0
	var lastFailure time.Time
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Docker subscription handler")
			return nil
		case event := <-eventBus:
			dockerEvent, ok := event.Data.(eventtypes.Message)
			if !ok {
				logger.Error("Failed to assert Docker event data type")
				continue
			}
			contName := dockerEvent.Actor.Attributes["name"]
			action := transition.DockerAction(dockerEvent.Action)
			plan, ok := dockerSubscriptionPlanForAction(action)
			if !ok {
				logger.Debug(fmt.Sprintf("%s event: %s", contName, dockerEvent.Action))
				failureCount = 0
				continue
			}
			if err := runDockerTransitionWorkflow(rt, contName, plan); err != nil {
				failureCount, lastFailure = handleSubscriptionFailure(failureCount, lastFailure, err)
				logger.Warn(fmt.Sprintf("Docker transition failed for %s: %v", contName, err))
				if failureCount >= dockerTransitionFailureLimit {
					return fmt.Errorf("docker subscription handler failed after %d consecutive transition failures: %w", dockerTransitionFailureLimit, err)
				}
				continue
			}
			failureCount = 0
		}
	}
}

func handleSubscriptionFailure(count int, lastFailure time.Time, err error) (int, time.Time) {
	nextFailure := time.Now()
	if lastFailure.IsZero() || nextFailure.Sub(lastFailure) > dockerTransitionFailureWindow {
		return 1, nextFailure
	}
	logger.Warn(fmt.Sprintf("Recent docker transition failures in window: %d", count+1))
	return count + 1, nextFailure
}

func startDockerHealthLoop(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	dockerHealthLoopMu.Lock()
	defer dockerHealthLoopMu.Unlock()
	if healthLoopRunning {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	healthLoopRunning = true
	healthLoopStopFn = cancel

	go func() {
		Check502Loop(loopCtx)
		dockerHealthLoopMu.Lock()
		if healthLoopRunning {
			healthLoopRunning = false
			healthLoopStopFn = nil
		}
		dockerHealthLoopMu.Unlock()
	}()
	return nil
}
