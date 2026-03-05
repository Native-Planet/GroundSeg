package events

import (
	"context"
	"errors"
)

type Runtime struct {
	Channel func() <-chan string
	Process func(string)
}

func Run(ctx context.Context, configEvents <-chan string, processFn func(string)) error {
	if configEvents == nil {
		return errors.New("config event channel is nil")
	}
	if processFn == nil {
		return errors.New("config event processor is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-configEvents:
				if !ok {
					return
				}
				processFn(event)
			}
		}
	}()
	return nil
}

func Start(ctx context.Context, runtime Runtime) error {
	if runtime.Channel == nil {
		return errors.New("config event runtime channel callback is nil")
	}
	return Run(ctx, runtime.Channel(), runtime.Process)
}
