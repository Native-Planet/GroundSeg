package lifecycle

// Phase is a typed lifecycle label emitted during orchestration flows.
type Phase string

type Step struct {
	Phase Phase
	Run   func() error
}

type Runner struct {
	Emit      func(phase Phase)
	OnError   func(phase Phase, err error)
	OnSuccess func()
	OnCleanup func()
}

func (runner Runner) Run(steps ...Step) error {
	if runner.OnCleanup != nil {
		defer runner.OnCleanup()
	}
	for _, step := range steps {
		if runner.Emit != nil && step.Phase != "" {
			runner.Emit(step.Phase)
		}
		if step.Run == nil {
			continue
		}
		if err := step.Run(); err != nil {
			if runner.OnError != nil {
				runner.OnError(step.Phase, err)
			}
			return err
		}
	}
	if runner.OnSuccess != nil {
		runner.OnSuccess()
	}
	return nil
}
