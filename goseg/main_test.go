package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestFallbackToIndexStaticAssets(t *testing.T) {
	webFS := http.FS(fstest.MapFS{
		"index.html": {Data: []byte("index")},
		"hermes.png": {Data: []byte("png")},
	})

	tests := []struct {
		name     string
		path     string
		wantCode int
		wantBody string
	}{
		{name: "static asset", path: "/hermes.png", wantCode: http.StatusOK, wantBody: "png"},
		{name: "missing static asset", path: "/missing.png", wantCode: http.StatusNotFound, wantBody: "404 page not found\n"},
		{name: "app route", path: "/profile", wantCode: http.StatusOK, wantBody: "index"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			fallbackToIndex(webFS)(rec, req)

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if got := rec.Body.String(); got != tt.wantBody {
				t.Fatalf("body = %q, want %q", got, tt.wantBody)
			}
			if location := rec.Header().Get("Location"); location != "" {
				t.Fatalf("unexpected redirect to %q", location)
			}
		})
	}
}

var _ fs.FS = fstest.MapFS{}
