package accesspoint

import (
	"errors"
	"fmt"

	"go.uber.org/zap"
)

type AccessPointRuntimeCoordinator interface {
	Start(AccessPointRuntime) error
	Stop(AccessPointRuntime) error
}

type accessPointLifecycleCoordinator struct{}

func accessPointStartRouterWithRuntime(resolved resolvedAccessPointRuntime) error {
	if resolved.runtime.StartRouterFn != nil {
		return resolved.runtime.StartRouterFn(resolved.runtime)
	}
	return nil
}

func accessPointStopRouterWithRuntime(resolved resolvedAccessPointRuntime) error {
	if resolved.runtime.StopRouterFn != nil {
		return resolved.runtime.StopRouterFn(resolved.runtime)
	}
	return nil
}

func (accessPointLifecycleCoordinator) Start(rt AccessPointRuntime) error {
	return accessPointLifecycleCoordinator{}.StartResolved(resolveAccessPointRuntime(rt))
}

func (accessPointLifecycleCoordinator) StartResolved(resolved resolvedAccessPointRuntime) error {
	rt := resolved.runtime
	zapLogger := zap.L()
	zapLogger.Info(fmt.Sprintf("Starting router on %v", rt.Wlan))

	if rt.EnsureRootDirFn == nil {
		return errors.New("missing root directory strategy")
	}
	if err := rt.EnsureRootDirFn(rt.RootDir); err != nil {
		return err
	}
	// make sure dependencies are met
	if rt.CheckDependenciesFn == nil {
		return errors.New("missing dependency strategy")
	}
	if err := rt.CheckDependenciesFn(); err != nil {
		return err
	}
	// make sure params are set (maybe not needed)
	if rt.CheckParametersFn == nil {
		return errors.New("missing parameter validation strategy")
	}
	if err := rt.CheckParametersFn(rt); err != nil {
		return err
	}
	// check if AP already running
	if rt.IsRunningFn == nil {
		return errors.New("missing status strategy")
	}
	running, err := rt.IsRunningFn(rt)
	if err != nil {
		return err
	}
	if running {
		if rt.ForceRestart {
			zapLogger.Info("Accesspoint already started; force restart requested")
		} else {
			zapLogger.Info("Accesspoint already started")
			return nil
		}
	}
	// dump config to file
	if rt.WriteHostapdConfigFn == nil {
		return errors.New("missing hostapd config strategy")
	}
	if err := rt.WriteHostapdConfigFn(rt.HostapdConfigPath, rt.Wlan, rt.SSID, rt.Password); err != nil {
		return err
	}
	// start the router
	if rt.StartRouterFn == nil {
		return errors.New("missing router start strategy")
	}
	if err := accessPointStartRouterWithRuntime(resolved); err != nil {
		return fmt.Errorf("start router: %w", err)
	}
	return nil
}

func (accessPointLifecycleCoordinator) Stop(rt AccessPointRuntime) error {
	return accessPointLifecycleCoordinator{}.StopResolved(resolveAccessPointRuntime(rt))
}

func (accessPointLifecycleCoordinator) StopResolved(resolved resolvedAccessPointRuntime) error {
	rt := resolved.runtime
	zapLogger := zap.L()
	zapLogger.Info(fmt.Sprintf("Stopping router on %v", rt.Wlan))

	if rt.CheckParametersFn == nil {
		return errors.New("missing parameter validation strategy")
	}
	if err := rt.CheckParametersFn(rt); err != nil {
		return err
	}
	// check if AP is running
	if rt.IsRunningFn == nil {
		return errors.New("missing status strategy")
	}
	running, err := rt.IsRunningFn(rt)
	if err != nil {
		return err
	}
	// stop the router
	if running {
		if rt.StopRouterFn == nil {
			return errors.New("missing router stop strategy")
		}
		if err := accessPointStopRouterWithRuntime(resolved); err != nil {
			return fmt.Errorf("stop router: %w", err)
		}
	} else {
		zapLogger.Info("Accesspoint already stopped")
	}
	return nil
}
