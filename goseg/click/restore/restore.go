package restore

import (
	"fmt"
	"strings"

	"groundseg/click/internal/runtime"

	"go.uber.org/zap"
)

var (
	executeClickCommandForRestore = executeClickCommandForRestoreImpl
	filterResponseForRestore      = runtime.FilterResponse
	restoreAgentFn                = restoreAgent
)

func executeClickCommandForRestoreImpl(patp, file, hoon, sourcePath, successToken, operation string) (string, error) {
	if err := runtime.CreateHoon(patp, file, hoon); err != nil {
		return "", fmt.Errorf("%s failed to create hoon: %v", operation, err)
	}
	defer runtime.DeleteHoon(patp, file)

	response, err := runtime.ClickExec(patp, file, sourcePath)
	if err != nil {
		return "", fmt.Errorf("%s failed to execute hoon: %v", operation, err)
	}
	if successToken != "" {
		_, success, err := runtime.FilterResponse(successToken, response)
		if err != nil {
			return "", fmt.Errorf("%s failed to parse response: %v", operation, err)
		}
		if !success {
			return "", fmt.Errorf("%s failed poke", operation)
		}
	}
	return response, nil
}

func restoreAgent(patp, agent string) error {
	file := fmt.Sprintf("restore-%s", agent)
	hoon := strings.Join([]string{
		"=/", "m", "(strand ,vase)",
		"^-", "form:m",
		";<", "egg=@", "bind:m", fmt.Sprintf("(scry @ /cx/base/bak/backup-%s/jam)", agent),
		";<", "our=@p", "bind:m", "get-our",
		"=/", "dk=dock", fmt.Sprintf("[our %%%s]", agent),
		"=/", "cg=cage", "[%noun !>((cue egg))]",
		"=/", "=card:agent:gall", "[%pass /pokeas %agent dk %poke %egg-any -:!>(*egg-any:gall) (cue egg)]",
		";<", "~", "bind:m", "(send-raw-card card)",
		";<", "~", "bind:m", "(take-poke-ack /pokeas)",
		"(pure:m !>('success'))",
	}, " ")
	response, err := executeClickCommandForRestore(patp, file, hoon, "", "", fmt.Sprintf("Click %s", file))
	if err != nil {
		return err
	}
	_, succeeded, err := filterResponseForRestore("success", response)
	if err != nil {
		return fmt.Errorf("Click %s failed to get exec: %v", file, err)
	}
	if !succeeded {
		return fmt.Errorf("Click %s failed poke: %s", file, patp)
	}
	zap.L().Info(fmt.Sprintf("Click %s restored agent %s on %s", file, agent, patp))
	return nil
}

func RestoreAgent(patp, agent string) error {
	return restoreAgent(patp, agent)
}

func RestoreTlon(patp string) error {
	var errors []string
	components := []struct {
		name string
		err  error
	}{
		{"activity", restoreAgentFn(patp, "activity")},
		{"channels", restoreAgentFn(patp, "channels")},
		{"channels-server", restoreAgentFn(patp, "channels-server")},
		{"groups", restoreAgentFn(patp, "groups")},
		{"profile", restoreAgentFn(patp, "profile")},
		{"chat", restoreAgentFn(patp, "chat")},
	}

	for _, component := range components {
		if component.err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", component.name, component.err))
		}
	}

	if len(errors) == 0 {
		return nil
	}
	return fmt.Errorf("restore errors: %s", strings.Join(errors, ", "))
}
