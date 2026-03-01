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
	originalClient := versionHTTPClient
	originalRetryCount := versionFetchRetryCount
	originalRetryDelay := versionFetchRetryDelay
	originalSleep := versionFetchSleep
	originalConfig := globalConfig
	t.Cleanup(func() {
		versionHTTPClient = originalClient
		versionFetchRetryCount = originalRetryCount
		versionFetchRetryDelay = originalRetryDelay
		versionFetchSleep = originalSleep
		globalConfig = originalConfig
	})

	sleepCalls := 0
	versionFetchSleep = func(_ time.Duration) {
		sleepCalls++
	}
	versionFetchRetryCount = 3
	versionFetchRetryDelay = time.Millisecond
	globalConfig.UpdateUrl = "https://updates.example/version"

	client := &stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{err: errors.New("temporary network failure")},
			{resp: newHTTPResponse(http.StatusOK, `{"groundseg":{"latest":{}}}`)},
		},
	}
	versionHTTPClient = client

	version, err := fetchVersionFromServer(structs.SysConfig{GsVersion: "1.0.0"})
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
	originalClient := versionHTTPClient
	originalRetryCount := versionFetchRetryCount
	originalRetryDelay := versionFetchRetryDelay
	originalSleep := versionFetchSleep
	originalConfig := globalConfig
	t.Cleanup(func() {
		versionHTTPClient = originalClient
		versionFetchRetryCount = originalRetryCount
		versionFetchRetryDelay = originalRetryDelay
		versionFetchSleep = originalSleep
		globalConfig = originalConfig
	})

	versionFetchSleep = func(_ time.Duration) {}
	versionFetchRetryCount = 2
	versionFetchRetryDelay = time.Millisecond
	globalConfig.UpdateUrl = "https://updates.example/version"
	versionHTTPClient = &stubVersionHTTPClient{
		results: []stubVersionHTTPResult{
			{err: errors.New("network down")},
			{err: errors.New("network still down")},
		},
	}

	_, err := fetchVersionFromServer(structs.SysConfig{GsVersion: "1.0.0"})
	if err == nil {
		t.Fatal("expected error after retry exhaustion")
	}
	if !strings.Contains(err.Error(), "request version metadata") {
		t.Fatalf("expected wrapped request error, got %v", err)
	}
}
