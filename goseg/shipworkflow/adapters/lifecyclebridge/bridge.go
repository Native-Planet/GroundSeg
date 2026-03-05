package lifecyclebridge

import (
	"context"
	"groundseg/lifecycle"
	"groundseg/structs"
	"time"
)

var (
	ErrMutateFunctionRequired  = lifecycle.ErrMutateFunctionRequired
	ErrPersistFunctionRequired = lifecycle.ErrPersistFunctionRequired
	ErrPollIntervalNonPositive = lifecycle.ErrPollIntervalNonPositive
)

type shipConfigPersist func(string, func(*structs.UrbitDocker) error) error

func PersistUrbitConfig(patp string, mutate func(*structs.UrbitDocker) error, persistFn shipConfigPersist) error {
	return lifecycle.PersistWithMutator[structs.UrbitDocker](patp, mutate, persistFn)
}

func PollWithTimeout(ctx context.Context, interval time.Duration, condition func() (bool, error)) error {
	return lifecycle.PollWithTimeout(ctx, interval, condition)
}
