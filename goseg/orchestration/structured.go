package orchestration

import "groundseg/lifecycle"

// WorkflowPhases defines the structured phases of a coordinated operation.
type WorkflowPhases struct {
	Prepare  []lifecycle.Step
	Execute  []lifecycle.Step
	Post     []lifecycle.Step
	Rollback []lifecycle.Step
}

// WorkflowCallbacks defines optional hooks for operation orchestration.
type WorkflowCallbacks struct {
	Emit       func(lifecycle.Phase)
	OnError    func(error)
	OnSuccess  func()
	OnCleanup  func()
	OnRollback func(error)
}

// RunStructuredWorkflow runs phases in prepare -> execute -> post order with rollback on failure.
func RunStructuredWorkflow(phases WorkflowPhases, callbacks WorkflowCallbacks) error {
	if callbacks.OnCleanup != nil {
		defer callbacks.OnCleanup()
	}

	runPhases := func(runner []lifecycle.Step) error {
		return RunPhases(runner, callbacks.Emit, func(_ lifecycle.Phase, err error) {
			if callbacks.OnError != nil {
				callbacks.OnError(err)
			}
		}, nil, nil)
	}

	if err := runPhases(phases.Prepare); err != nil {
		if callbacks.OnRollback != nil {
			callbacks.OnRollback(err)
		}
		if err := runPhases(phases.Rollback); err != nil && callbacks.OnError != nil {
			callbacks.OnError(err)
		}
		return err
	}
	if err := runPhases(phases.Execute); err != nil {
		if callbacks.OnRollback != nil {
			callbacks.OnRollback(err)
		}
		if err := runPhases(phases.Rollback); err != nil && callbacks.OnError != nil {
			callbacks.OnError(err)
		}
		return err
	}
	if err := runPhases(phases.Post); err != nil {
		if callbacks.OnRollback != nil {
			callbacks.OnRollback(err)
		}
		if err := runPhases(phases.Rollback); err != nil && callbacks.OnError != nil {
			callbacks.OnError(err)
		}
		return err
	}
	if callbacks.OnSuccess != nil {
		callbacks.OnSuccess()
	}
	return nil
}
