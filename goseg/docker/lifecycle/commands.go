package lifecycle

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
)

// ExecDockerCommand executes a command inside a container and returns stdout/stderr plus the exit code.
func (runtime *Runtime) ExecDockerCommand(containerName string, cmd []string) (output string, exitCode int, err error) {
	cli, err := runtime.dockerClientNew()
	if err != nil {
		return "", -1, fmt.Errorf("unable to create docker client: %w", err)
	}
	defer closeRuntimeDockerClient(cli, "docker command exec", &err)

	execConfig := container.ExecOptions{AttachStdout: true, AttachStderr: true, Cmd: cmd}
	ctx := context.Background()

	containerID, err := GetContainerIDByName(ctx, cli, containerName)
	if err != nil {
		return "", -1, fmt.Errorf("lookup container %s: %w", containerName, err)
	}

	resp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", -1, fmt.Errorf("failed to create exec for container %s: %w", containerName, err)
	}
	hijackedResp, err := cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", -1, fmt.Errorf("failed to attach to exec for container %s: %w", containerName, err)
	}
	defer hijackedResp.Close()

	rawOutput, err := ioutil.ReadAll(hijackedResp.Reader)
	if err != nil {
		return output, -1, fmt.Errorf("read exec output from %s: %w", containerName, err)
	}
	output = string(rawOutput)
	execID := resp.ID
	deadline := time.Now().Add(30 * time.Second)
	for {
		execState, inspectErr := cli.ContainerExecInspect(ctx, execID)
		if inspectErr != nil {
			return string(output), -1, fmt.Errorf("failed to inspect exec command %s in %s: %w", execID, containerName, inspectErr)
		}
		if !execState.Running && execState.ExitCode != 0 {
			return string(output), execState.ExitCode, fmt.Errorf("command failed with exit code %d: %s", execState.ExitCode, strings.TrimSpace(string(output)))
		}
		if !execState.Running && execState.ExitCode == 0 {
			return string(output), 0, nil
		}
		if time.Now().After(deadline) {
			return string(output), -1, fmt.Errorf("timed out waiting for command completion in %s", containerName)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func GetContainerIDByName(ctx context.Context, cli *client.Client, name string) (string, error) {
	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}
	for _, cont := range containers {
		for _, n := range cont.Names {
			if n == "/"+name {
				return cont.ID, nil
			}
		}
	}
	return "", errdefs.NotFound(fmt.Errorf("container %s not found", name))
}
