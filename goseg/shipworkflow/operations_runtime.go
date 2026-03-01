package shipworkflow

import (
	"context"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/lifecycle"
	"groundseg/orchestration"
	"groundseg/structs"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	publishUrbitTransitionForWorkflow = docker.PublishUrbitTransition
	sleepForWorkflow                  = time.Sleep
	lookupAliasCNAME                  = net.LookupCNAME
	cleanDeleteGetStatus              = docker.GetShipStatus
	cleanDeleteBarExit                = click.BarExit
	cleanDeleteStopContainer          = docker.StopContainerByName
	cleanDeleteDeleteContainer        = docker.DeleteContainer
)

func SetAliasCNAMELookup(fn func(string) (string, error)) {
	lookupAliasCNAME = fn
}

func SetUrbitCleanupSeams(
	getStatus func([]string) (map[string]string, error),
	barExit func(string) error,
	stopContainer func(string) error,
	deleteContainer func(string) error,
) {
	cleanDeleteGetStatus = getStatus
	cleanDeleteBarExit = barExit
	cleanDeleteStopContainer = stopContainer
	cleanDeleteDeleteContainer = deleteContainer
}

func emitUrbitTransition(patp, transitionType, event string) {
	publishUrbitTransitionForWorkflow(structs.UrbitTransition{Patp: patp, Type: transitionType, Event: event})
}

func PublishTransitionWithPolicy[T any](publish func(T), event T, clear T, clearDelay time.Duration) {
	publish(event)
	if clearDelay > 0 {
		sleepForWorkflow(clearDelay)
	}
	publish(clear)
}

func RunTransitionedOperation(patp, transitionType, startEvent, successEvent string, clearDelay time.Duration, operation func() error) error {
	policy := orchestration.NewTransitionPolicy(clearDelay, sleepForWorkflow)
	return orchestration.RunSinglePhase(
		lifecycle.Phase(startEvent),
		operation,
		func(phase lifecycle.Phase) {
			emitUrbitTransition(patp, transitionType, string(phase))
		},
		func(_ lifecycle.Phase, _ error) {
			emitUrbitTransition(patp, transitionType, "error")
		},
		func() {
			if successEvent != "" {
				emitUrbitTransition(patp, transitionType, successEvent)
			}
		},
		func() {
			policy.Cleanup(func() {
				emitUrbitTransition(patp, transitionType, "")
			})
		},
	)
}

func waitForDeskState(patp, desk, expectedState string, shouldMatch bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return PollWithTimeout(ctx, 5*time.Second, func() (bool, error) {
		status, err := click.GetDesk(patp, desk, true)
		if err != nil {
			return false, fmt.Errorf("get %s desk status for %s: %w", desk, patp, err)
		}
		if shouldMatch {
			return status == expectedState, nil
		}
		return status != expectedState, nil
	})
}

func runDeskTransition(patp, transitionType string, operation func() error) error {
	return RunTransitionedOperation(patp, transitionType, "loading", "success", 3*time.Second, operation)
}

func WaitComplete(patp string) error {
	return WaitForUrbitStop(patp, docker.GetShipStatus, PollWithTimeout)
}

func persistShipConf(patp string, mutate func(*structs.UrbitDocker) error) error {
	return PersistUrbitConfig(patp, mutate, config.UpdateUrbit)
}

func PersistUrbitConfigValue(patp string, mutate func(*structs.UrbitDocker) error) error {
	return persistShipConf(patp, mutate)
}

func areSubdomainsAliases(domain1, domain2 string) (bool, error) {
	firstDot := strings.Index(domain1, ".")
	if firstDot == -1 {
		return false, fmt.Errorf("Invalid subdomain")
	}
	if config.GetStartramConfig().Cname != "" && domain1[firstDot+1:] == config.GetStartramConfig().Cname {
		return true, nil
	}
	cname1, err := lookupAliasCNAME(domain1)
	if err != nil {
		return false, err
	}
	cname2, err := lookupAliasCNAME(domain2)
	if err != nil {
		return false, err
	}
	return cname1 == cname2, nil
}

func AreSubdomainsAliases(domain1, domain2 string) (bool, error) {
	return areSubdomainsAliases(domain1, domain2)
}

func urbitCleanDelete(patp string) error {
	getShipRunningStatus := func(patp string) (string, error) {
		statuses, err := cleanDeleteGetStatus([]string{patp})
		if err != nil {
			return "", fmt.Errorf("Failed to get statuses for %s: %w", patp, err)
		}
		status, exists := statuses[patp]
		if !exists {
			return "", fmt.Errorf("%s status doesn't exist", patp)
		}
		return status, nil
	}
	status, err := getShipRunningStatus(patp)
	if err == nil {
		if strings.Contains(status, "Up") {
			if err := cleanDeleteBarExit(patp); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to stop %s with |exit: %v", patp, err))
				if err = cleanDeleteStopContainer(patp); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to stop %s: %v", patp, err))
				}
			}
		}
		for {
			status, err := getShipRunningStatus(patp)
			if err != nil {
				break
			}
			zap.L().Debug(fmt.Sprintf("%s", status))
			if !strings.Contains(status, "Up") {
				break
			}
			sleepForWorkflow(1 * time.Second)
		}
	}
	return cleanDeleteDeleteContainer(patp)
}

func UrbitCleanDelete(patp string) error {
	return urbitCleanDelete(patp)
}
