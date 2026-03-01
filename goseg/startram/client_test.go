package startram

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetAPIClientIgnoresNilAndDelegatesCalls(t *testing.T) {
	original := defaultAPIClient
	t.Cleanup(func() {
		defaultAPIClient = original
	})

	getCalls := 0
	postCalls := 0
	stub := stubStartramAPIClient{
		getFn: func(url string) (*http.Response, error) {
			getCalls++
			if url != "https://example.test/get" {
				t.Fatalf("unexpected get url: %s", url)
			}
			return newStartramHTTPResponse(http.StatusAccepted, `{"ok":true}`), nil
		},
		postFn: func(url, contentType string, body io.Reader) (*http.Response, error) {
			postCalls++
			if url != "https://example.test/post" {
				t.Fatalf("unexpected post url: %s", url)
			}
			if contentType != "application/json" {
				t.Fatalf("unexpected content type: %s", contentType)
			}
			payload, _ := io.ReadAll(body)
			if string(payload) != "{}" {
				t.Fatalf("unexpected post body: %q", payload)
			}
			return newStartramHTTPResponse(http.StatusCreated, `{"ok":true}`), nil
		},
	}

	SetAPIClient(stub)
	SetAPIClient(nil)

	getResp, err := apiGet("https://example.test/get")
	if err != nil {
		t.Fatalf("apiGet returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusAccepted {
		t.Fatalf("unexpected get status: %d", getResp.StatusCode)
	}

	postResp, err := apiPost("https://example.test/post", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("apiPost returned error: %v", err)
	}
	if postResp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected post status: %d", postResp.StatusCode)
	}

	if getCalls != 1 || postCalls != 1 {
		t.Fatalf("unexpected transport call counts: get=%d post=%d", getCalls, postCalls)
	}
}

func TestHTTPAPIClientUsesProvidedHTTPClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNoContent)
		case http.MethodPost:
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := httpAPIClient{client: server.Client()}

	getResp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("httpAPIClient.Get returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected get status: %d", getResp.StatusCode)
	}

	postResp, err := client.Post(server.URL, "text/plain", strings.NewReader("body"))
	if err != nil {
		t.Fatalf("httpAPIClient.Post returned error: %v", err)
	}
	if postResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected post status: %d", postResp.StatusCode)
	}
}
