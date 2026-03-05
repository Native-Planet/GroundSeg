package subsystem

import (
	"context"
	"fmt"
	"sync"
	"time"

	"groundseg/docker/internal/closeutil"
	"groundseg/dockerclient"
	"groundseg/internal/workflow"
	"groundseg/logger"
	"groundseg/structs"
	"groundseg/transition"

	eventtypes "github.com/docker/docker/api/types/events"
)

type dockerSubsystemRuntime struct {
	events dockerSubsystemEventBus
	health dockerHealthLoopState
}

type dockerEventStreamClient interface {
	Events(context.Context, eventtypes.ListOptions) (<-chan eventtypes.Message, <-chan error)
	Close() error
}

type dockerSubsystemEventBus struct {
	channel chan structs.Event
}

type dockerHealthLoopState struct {
	mu      sync.Mutex
	running bool
	stopFn  context.CancelFunc
	loopID  int
}

func newDockerSubsystemRuntime() *dockerSubsystemRuntime {
	return &dockerSubsystemRuntime{
		events: dockerSubsystemEventBus{
			channel: make(chan structs.Event, 100),
		},
	}
}

var (
	DisableShipRestart = false
	errEventBusFull    = fmt.Errorf("docker event bus is full")

	dockerTransitionFailureLimit  = 5
	dockerTransitionFailureWindow = 3 * time.Minute

	defaultDockerSubsystemRuntime = newDockerSubsystemRuntime()
	newDockerEventStreamClient    = func() (dockerEventStreamClient, error) { return dockerclient.New() }
)

func normalizeDockerSubsystemRuntime(runtime *dockerSubsystemRuntime) *dockerSubsystemRuntime {
	if runtime == nil {
		return defaultDockerSubsystemRuntime
	}
	return runtime
}

func StartDockerHealthLoops(ctx context.Context) error {
	return StartDockerHealthLoopsWithRuntime(ctx, defaultDockerSubsystemRuntime)
}

func StartDockerHealthLoopsWithRuntime(ctx context.Context, runtime *dockerSubsystemRuntime) error {
	runtime = normalizeDockerSubsystemRuntime(runtime)
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil
	}
	runtime.health.mu.Lock()
	defer runtime.health.mu.Unlock()
	if runtime.health.running {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	runtime.health.loopID++
	loopID := runtime.health.loopID
	runtime.health.running = true
	runtime.health.stopFn = cancel

	go func() {
		Check502Loop(loopCtx)
		runtime.health.mu.Lock()
		if runtime.health.running && runtime.health.loopID == loopID {
			runtime.health.running = false
			runtime.health.stopFn = nil
		}
		runtime.health.mu.Unlock()
	}()
	return nil
}

func StartDockerSubsystem(ctx context.Context) error {
	return StartDockerSubsystemWithRuntime(ctx, defaultDockerSubsystemRuntime)
}

// RunDockerSubsystem blocks until context cancellation or worker failure.
func RunDockerSubsystem(ctx context.Context) error {
	return RunDockerSubsystemWithRuntime(ctx, defaultDockerSubsystemRuntime)
}

func StartDockerSubsystemWithRuntime(ctx context.Context, runtime *dockerSubsystemRuntime) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Err() != nil {
		return nil
	}
	runtime = normalizeDockerSubsystemRuntime(runtime)
	go func() {
		if err := RunDockerSubsystemWithRuntime(ctx, runtime); err != nil && ctx.Err() == nil {
			logger.Error(fmt.Sprintf("Docker subsystem exited with error: %v", err))
		}
	}()
	return nil
}

// RunDockerSubsystemWithRuntime blocks until context cancellation or worker failure.
func RunDockerSubsystemWithRuntime(ctx context.Context, runtime *dockerSubsystemRuntime) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runtime = normalizeDockerSubsystemRuntime(runtime)
	if err := StartDockerHealthLoopsWithRuntime(ctx, runtime); err != nil {
		return err
	}
	defer StopDockerHealthLoopsWithRuntime(runtime)
	return workflow.RunUntilDoneOrWorkerResult(
		ctx,
		func(workerCtx context.Context) error {
			return dockerListenerWithRuntime(runtime, workerCtx)
		},
		func(workerCtx context.Context) error {
			return dockerSubscriptionHandlerWithRuntime(runtime, workerCtx)
		},
	)
}

func StopDockerHealthLoops() {
	StopDockerHealthLoopsWithRuntime(defaultDockerSubsystemRuntime)
}

func StopDockerHealthLoopsWithRuntime(runtime *dockerSubsystemRuntime) {
	runtime = normalizeDockerSubsystemRuntime(runtime)
	runtime.health.mu.Lock()
	defer runtime.health.mu.Unlock()
	if !runtime.health.running || runtime.health.stopFn == nil {
		return
	}
	runtime.health.stopFn()
	runtime.health.running = false
	runtime.health.stopFn = nil
}

func (runtime *dockerSubsystemRuntime) isHealthLoopRunning() bool {
	if runtime == nil {
		return false
	}
	runtime.health.mu.Lock()
	defer runtime.health.mu.Unlock()
	return runtime.health.running
}

// dockerListenerWithContext runs until ctx is canceled.
func dockerListenerWithRuntime(runtime *dockerSubsystemRuntime, ctx context.Context) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runtime = normalizeDockerSubsystemRuntime(runtime)
	cli, err := newDockerEventStreamClient()
	if err != nil {
		return fmt.Errorf("initialize Docker client: %w", err)
	}
	defer closeutil.MergeCloseError(cli, "docker event listener", &err)
	messages, errs := cli.Events(ctx, eventtypes.ListOptions{})
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Docker event listener")
			return nil
		case event, ok := <-messages:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("docker event stream closed")
			}
			if ctx.Err() != nil {
				return nil
			}
			// Convert the Docker event to our custom event and send it to the EventBus.
			select {
			case runtime.events.channel <- structs.Event{Type: string(event.Action), Data: event}:
				continue
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("docker event bus saturated: %w", errEventBusFull)
			}
		case eventErr, ok := <-errs:
			if !ok {
				if ctx.Err() != nil {
					return nil
				}
				return fmt.Errorf("docker event error stream closed")
			}
			if eventErr == nil {
				if ctx.Err() != nil {
					return nil
				}
				continue
			}
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("docker event error: %w", eventErr)
		}
	}
}

func dockerSubscriptionHandlerWithRuntime(runtime *dockerSubsystemRuntime, ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runtime = normalizeDockerSubsystemRuntime(runtime)
	rt := newDockerRoutineRuntime()
	failureCount := 0
	var lastFailure time.Time
	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping Docker subscription handler")
			return nil
		case event := <-runtime.events.channel:
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
