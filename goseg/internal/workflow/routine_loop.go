package workflow

import "time"

// RunForever executes fn repeatedly, sleeps for interval between passes, and
// reports pass errors through onError.
func RunForever(interval time.Duration, fn func() error, sleep func(time.Duration), onError func(error)) {
	if fn == nil {
		return
	}
	if sleep == nil {
		sleep = time.Sleep
	}
	if onError == nil {
		onError = func(error) {}
	}
	for {
		if err := fn(); err != nil {
			onError(err)
		}
		sleep(interval)
	}
}
