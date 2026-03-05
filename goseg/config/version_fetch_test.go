package config

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"groundseg/structs"
)

func preserveVersionFetchRuntimeState(t *testing.T) {
	t.Helper()
	state := versionRuntimeSnapshot()
	state.stateMu.Lock()
	originalStore := state.versionStore
	originalClient := state.versionHTTPClient
	originalRetryCount := state.versionFetchRetryCount
	originalRetryDelay := state.versionFetchRetryDelay
	originalSleep := state.versionFetchSleep
	state.stateMu.Unlock()
	t.Cleanup(func() {
		setVersionStore(originalStore)
		setVersionHTTPClient(originalClient)
		setVersionFetchPolicy(originalRetryCount, originalRetryDelay)
		setVersionFetchSleep(originalSleep)
	})
}

type stubVersionHTTPResult struct {
	resp *http.Response
	err  error
}

type stubVersionHTTPClient struct {
	results []stubVersionHTTPResult
	calls   int
}

func (client *stubVersionHTTPClient) Do(_ *http.Request) (*http.Response, error) {
	if client.calls >= len(client.results) {
		client.calls++
		return nil, errors.New("unexpected HTTP call")
	}
	result := client.results[client.calls]
	client.calls++
	return result.resp, result.err
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestFetchVersionFromServerRetriesThenSucceeds(t *testing.T) {
	preserveVersionFetchRuntimeState(t)
	originalConfig := globalConfig
	t.Cleanup(func() {
		globalConfig = originalConfig
	})

	sleepCalls := 0
	setVersionFetchSleep(func(_ time.Duration) {
		sleepCalls++
	})
	setVersionFetchPolicy(3, time.Millisecond)
	globalConfig.Connectivity.UpdateURL = "https://updates.example/version"

	client := &stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{err: errors.New("temporary network failure")},
			{resp: newHTTPResponse(http.StatusOK, `{"groundseg":{"latest":{}}}`)},
		},
	}
	setVersionHTTPClient(client)

	conf := structs.SysConfig{}
	conf.Runtime.GsVersion = "1.0.0"
	version, err := fetchVersionFromServer(conf)
	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if _, ok := version.Groundseg["latest"]; !ok {
		t.Fatalf("expected latest channel to be decoded, got %+v", version)
	}
	if client.calls != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", client.calls)
	}
	if sleepCalls != 1 {
		t.Fatalf("expected 1 backoff sleep call, got %d", sleepCalls)
	}
}

func TestFetchVersionFromServerFailsAfterRetryExhaustion(t *testing.T) {
	preserveVersionFetchRuntimeState(t)
	originalConfig := globalConfig
	t.Cleanup(func() {
		globalConfig = originalConfig
	})

	setVersionFetchSleep(func(_ time.Duration) {})
	setVersionFetchPolicy(2, time.Millisecond)
	globalConfig.Connectivity.UpdateURL = "https://updates.example/version"
	setVersionHTTPClient(&stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{err: errors.New("network down")},
			{err: errors.New("network still down")},
		},
	})

	conf := structs.SysConfig{}
	conf.Runtime.GsVersion = "1.0.0"
	_, err := fetchVersionFromServer(conf)
	if err == nil {
		t.Fatal("expected error after retry exhaustion")
	}
	if !strings.Contains(err.Error(), "request version metadata") {
		t.Fatalf("expected wrapped request error, got %v", err)
	}
}
