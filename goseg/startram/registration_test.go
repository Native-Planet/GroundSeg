package startram

import (
	"context"
	"errors"
	"groundseg/config"
	"groundseg/structs"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWaitForRegistration(t *testing.T) {
	t.Run("pending becomes ready", func(t *testing.T) {
		calls := 0
		err := waitForRegistration(context.Background(), time.Millisecond, func(context.Context) (structs.StartramRetrieve, error) {
			calls++
			if calls < 3 {
				return structs.StartramRetrieve{}, errConfigurationPending
			}
			return structs.StartramRetrieve{}, nil
		})
		if err != nil || calls != 3 {
			t.Fatalf("calls=%d error=%v", calls, err)
		}
	})
	t.Run("permanent error", func(t *testing.T) {
		want := errors.New("invalid configuration")
		calls := 0
		err := waitForRegistration(context.Background(), time.Millisecond, func(context.Context) (structs.StartramRetrieve, error) {
			calls++
			return structs.StartramRetrieve{}, want
		})
		if !errors.Is(err, want) || calls != 1 {
			t.Fatalf("calls=%d error=%v", calls, err)
		}
	})
	t.Run("deadline while pending", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := waitForRegistration(ctx, time.Hour, func(context.Context) (structs.StartramRetrieve, error) {
			return structs.StartramRetrieve{}, errConfigurationPending
		})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error=%v", err)
		}
	})
	t.Run("in flight cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err := waitForRegistration(ctx, time.Hour, func(ctx context.Context) (structs.StartramRetrieve, error) {
			cancel()
			<-ctx.Done()
			return structs.StartramRetrieve{}, ctx.Err()
		})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error=%v", err)
		}
	})
}

type registrationTransport func(*http.Request) (*http.Response, error)

func (f registrationTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRetrievePendingConfiguration(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	cached := config.StartramConfig.Conf
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		pending bool
	}{
		{"null", 200, `{"error":0,"status":"registered","conf":null}`, true},
		{"empty", 200, `{"error":0,"status":"registered","conf":""}`, true},
		{"invalid", 200, `{"error":0,"status":"ready","conf":"invalid!"}`, false},
		{"API error", 200, `{"error":1,"conf":null}`, false},
		{"unknown key", 400, `{"error":1,"status":"No record"}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = registrationTransport(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})
			_, err := Retrieve()
			if err == nil || errors.Is(err, errConfigurationPending) != tc.pending {
				t.Fatalf("error=%v want pending=%v", err, tc.pending)
			}
			if config.StartramConfig.Conf != cached {
				t.Fatal("incomplete response replaced cached configuration")
			}
		})
	}
}
