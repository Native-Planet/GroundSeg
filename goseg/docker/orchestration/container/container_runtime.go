package container

import (
	"fmt"
	"os"

	"groundseg/docker/orchestration/internal/artifactwriter"
	"groundseg/structs"
)

type ContainerRuntimePlan struct {
	ContainerName         string
	ContainerImage        string
	ConfigPath            string
	OpenConfigFn          func(string) (*os.File, error)
	CreateDefaultConfigFn func() error
	WriteConfigFn         func() error
	StartContainerFn      func(string, string) (structs.ContainerState, error)
	UpdateContainerState  func(string, structs.ContainerState)
}

func RunContainerWithRuntime(plan ContainerRuntimePlan) error {
	if plan.ConfigPath != "" {
		if plan.OpenConfigFn == nil {
			return fmt.Errorf("missing file opener")
		}
		if _, err := plan.OpenConfigFn(plan.ConfigPath); err != nil {
			if plan.CreateDefaultConfigFn == nil {
				return err
			}
			if err := plan.CreateDefaultConfigFn(); err != nil {
				return err
			}
		}
	}

	if plan.WriteConfigFn != nil {
		if err := plan.WriteConfigFn(); err != nil {
			return err
		}
	}

	if plan.StartContainerFn == nil {
		return fmt.Errorf("missing container runtime")
	}
	info, err := plan.StartContainerFn(plan.ContainerName, plan.ContainerImage)
	if err != nil {
		return err
	}
	if plan.UpdateContainerState != nil {
		plan.UpdateContainerState(plan.ContainerName, info)
	}
	return nil
}

func writeContainerConfigArtifactWithRuntime(plan artifactwriter.WriteConfig) error {
	if plan.NormalizeTarget == nil {
		plan.NormalizeTarget = artifactwriter.NormalizeVolumeTargetPath
	}
	return artifactwriter.Write(plan)
}
