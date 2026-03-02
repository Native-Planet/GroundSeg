package shipworkflow

import (
	"fmt"

	"groundseg/internal/workflow"
)

type shipContainerRebuildRuntime struct {
	DeleteContainerFn func(string) error
	LoadUrbitsFn      func() error
	LoadMCFn          func() error
	LoadMinIOsFn      func() error
}

type shipContainerRebuildOptions struct {
	piers             []string
	deletePiers       bool
	deleteMinioClient bool
	loadUrbits        bool
	loadMinIOClient   bool
	loadMinIOs        bool
}

func appendShipContainerRebuildSteps(
	steps []workflow.Step,
	runtime shipContainerRebuildRuntime,
	options shipContainerRebuildOptions,
) []workflow.Step {
	if options.loadUrbits && runtime.LoadUrbitsFn != nil {
		steps = append(steps, workflow.Step{
			Name: "load urbit containers",
			Run:  runtime.LoadUrbitsFn,
		})
	}
	if options.deleteMinioClient {
		steps = append(steps, workflow.Step{
			Name: "delete minio client container",
			Run:  func() error { return runtime.DeleteContainerFn("mc") },
		})
	}
	for _, patp := range options.piers {
		if options.deletePiers {
			ship := patp
			steps = append(steps, workflow.Step{
				Name: fmt.Sprintf("delete %s container", ship),
				Run:  func() error { return runtime.DeleteContainerFn(ship) },
			})
		}
		minio := fmt.Sprintf("minio_%s", patp)
		mini := minio
		steps = append(steps, workflow.Step{
			Name: fmt.Sprintf("delete %s container", mini),
			Run:  func() error { return runtime.DeleteContainerFn(mini) },
		})
	}
	if options.loadMinIOClient && runtime.LoadMCFn != nil {
		steps = append(steps, workflow.Step{
			Name: "load minio client container",
			Run:  runtime.LoadMCFn,
		})
	}
	if options.loadMinIOs && runtime.LoadMinIOsFn != nil {
		steps = append(steps, workflow.Step{
			Name: "load minio containers",
			Run:  runtime.LoadMinIOsFn,
		})
	}
	return steps
}
