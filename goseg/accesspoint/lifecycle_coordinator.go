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

func accessPointStartRouterWithRuntime(rt AccessPointRuntime) error {
	if rt.UseDefaultStartFn {
		return startRouterWithRuntime(rt)
	}
	if rt.StartRouterFn != nil {
		return rt.StartRouterFn(rt)
	}
	return nil
}

func accessPointStopRouterWithRuntime(rt AccessPointRuntime) error {
	if rt.UseDefaultStopFn {
		return stopRouterWithRuntime(rt)
	}
	if rt.StopRouterFn != nil {
		return rt.StopRouterFn(rt)
	}
	return nil
}

func (accessPointLifecycleCoordinator) Start(rt AccessPointRuntime) error {
	zapLogger := zap.L()
	zapLogger.Info(fmt.Sprintf("Starting router on %v", rt.Wlan))

	if rt.EnsureRootDirFn == nil {
		return errors.New("missing root directory init runtime")
	}
	if err := rt.EnsureRootDirFn(rt.RootDir); err != nil {
		return err
	}
	// make sure dependencies are met
	if rt.CheckDependenciesFn == nil {
		return errors.New("missing dependency runtime")
	}
	if err := rt.CheckDependenciesFn(); err != nil {
		return err
	}
	// make sure params are set (maybe not needed)
	if rt.CheckParametersFn == nil {
		return errors.New("missing parameter validation runtime")
	}
	if err := rt.CheckParametersFn(rt); err != nil {
		return err
	}
	// check if AP already running
	if rt.IsRunningFn == nil {
		return errors.New("missing status runtime")
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
		return errors.New("missing hostapd config runtime")
	}
	if err := rt.WriteHostapdConfigFn(rt.HostapdConfigPath, rt.Wlan, rt.SSID, rt.Password); err != nil {
		return err
	}
	// start the router
	if rt.StartRouterFn == nil {
		return errors.New("missing router start runtime")
	}
	if err := accessPointStartRouterWithRuntime(rt); err != nil {
		return fmt.Errorf("start router: %w", err)
	}
	return nil
}

func (accessPointLifecycleCoordinator) Stop(rt AccessPointRuntime) error {
	zapLogger := zap.L()
	zapLogger.Info(fmt.Sprintf("Stopping router on %v", rt.Wlan))

	if rt.CheckParametersFn == nil {
		return errors.New("missing parameter validation runtime")
	}
	if err := rt.CheckParametersFn(rt); err != nil {
		return err
	}
	// check if AP is running
	if rt.IsRunningFn == nil {
		return errors.New("missing status runtime")
	}
	running, err := rt.IsRunningFn(rt)
	if err != nil {
		return err
	}
	// stop the router
	if running {
		if rt.StopRouterFn == nil {
			return errors.New("missing router stop runtime")
		}
		if err := accessPointStopRouterWithRuntime(rt); err != nil {
			return fmt.Errorf("stop router: %w", err)
		}
	} else {
		zapLogger.Info("Accesspoint already stopped")
	}
	return nil
}
