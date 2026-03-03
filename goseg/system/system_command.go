package system

import (
	"bytes"
	"fmt"
	"os/exec"
)

func runCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("run command %q: %w", command, err)
	}
	return out.String(), nil
}
