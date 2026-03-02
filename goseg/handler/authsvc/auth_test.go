package authsvc

import (
	"sync"
	"testing"
)

func TestRegisterFailedLoginStartsLockoutAtThreshold(t *testing.T) {
	resetFailedLoginState()

	for i := 0; i < MaxFailedLogins-1; i++ {
		if start := registerFailedLogin(); start {
			t.Fatalf("did not expect lockout to start before %d failed logins, got at index %d", MaxFailedLogins, i+1)
		}
	}

	if !registerFailedLogin() {
		t.Fatalf("expected lockout to start on %dth failed login", MaxFailedLogins)
	}

	state := getLockoutStateSnapshot()
	if state.FailedLogins != MaxFailedLogins {
		t.Fatalf("expected failed logins=%d, got %d", MaxFailedLogins, state.FailedLogins)
	}
}

func TestLockoutSnapshotReturnsCurrentRemainder(t *testing.T) {
	resetFailedLoginState()
	setLockoutRemainder(42)

	state := getLockoutStateSnapshot()
	if state.Remainder != 42 {
		t.Fatalf("expected lockout remainder %d, got %d", 42, state.Remainder)
	}
}

func TestDecrementAndGetRemainderClampsAtZero(t *testing.T) {
	resetFailedLoginState()
	setLockoutRemainder(2)

	if got := decrementAndGetRemainder(); got != 1 {
		t.Fatalf("expected first decrement to return 1, got %d", got)
	}
	if got := decrementAndGetRemainder(); got != 0 {
		t.Fatalf("expected second decrement to return 0, got %d", got)
	}
	if got := decrementAndGetRemainder(); got != 0 {
		t.Fatalf("expected third decrement to stay at 0, got %d", got)
	}
}

func TestLockoutStateAccessIsSafeUnderConcurrentUse(t *testing.T) {
	resetFailedLoginState()
	const workers = 10
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				registerFailedLogin()
				_ = getLockoutStateSnapshot()
				decrementAndGetRemainder()
			}
		}()
	}
	wg.Wait()

	if state := getLockoutStateSnapshot(); state.FailedLogins == 0 {
		t.Fatal("expected some failed logins to be recorded")
	}
}
