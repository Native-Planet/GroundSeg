package startram

import (
	"io"
	"net/http"
)

// APIClient abstracts StarTram HTTP transport for injection/testing.
type APIClient interface {
	Get(url string) (*http.Response, error)
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}

type httpAPIClient struct {
	client *http.Client
}

func (c httpAPIClient) Get(url string) (*http.Response, error) {
	return c.client.Get(url)
}

func (c httpAPIClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	return c.client.Post(url, contentType, body)
}

var defaultAPIClient APIClient = httpAPIClient{client: http.DefaultClient}

func SetAPIClient(client APIClient) {
	if client != nil {
		defaultAPIClient = client
	}
}

func apiGet(url string) (*http.Response, error) {
	return defaultAPIClient.Get(url)
}

func apiPost(url, contentType string, body io.Reader) (*http.Response, error) {
	return defaultAPIClient.Post(url, contentType, body)
}
