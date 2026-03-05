package workflow

import (
	"context"
	"sync"
)

// ContextWorker executes work until completion, cancellation, or failure.
type ContextWorker func(context.Context) error

// RunUntilDoneOrWorkerResult runs workers with a shared cancellation scope.
// It returns nil when parent context is canceled, or the first worker result
// (including nil) when any worker exits.
func RunUntilDoneOrWorkerResult(ctx context.Context, workers ...ContextWorker) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errs := make(chan error, len(workers))
	var wg sync.WaitGroup
	started := 0
	for _, worker := range workers {
		if worker == nil {
			continue
		}
		started++
		wg.Add(1)
		go func(worker ContextWorker) {
			defer wg.Done()
			errs <- worker(runCtx)
		}(worker)
	}
	if started == 0 {
		return nil
	}
	for {
		select {
		case <-runCtx.Done():
			wg.Wait()
			return nil
		case err := <-errs:
			cancel()
			wg.Wait()
			return err
		}
	}
}
