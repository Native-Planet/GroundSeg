package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const errorBodyLimit = 512

// HTTPStatusError describes non-success status codes returned by HTTP boundaries.
type HTTPStatusError struct {
	URL        string
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("unexpected HTTP status %d from %s", e.StatusCode, e.URL)
	}
	return fmt.Sprintf("unexpected HTTP status %d from %s: %s", e.StatusCode, e.URL, e.Body)
}

// ReadBody reads a response body and validates that the HTTP status is 2xx.
func ReadBody(resp *http.Response, requestURL string) ([]byte, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read HTTP response body from %s: %w", requestURL, err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPStatusError{
			URL:        requestURL,
			StatusCode: resp.StatusCode,
			Body:       trimBody(body),
		}
	}

	return body, nil
}

// ReadJSON reads a response body and decodes JSON only after a success status check.
func ReadJSON(resp *http.Response, requestURL string, out interface{}) error {
	body, err := ReadBody(resp, requestURL)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode JSON response from %s: %w", requestURL, err)
	}

	return nil
}

func trimBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) <= errorBodyLimit {
		return trimmed
	}
	return trimmed[:errorBodyLimit] + "...(truncated)"
}
