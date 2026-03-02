package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"

	"groundseg/exporter"
	"groundseg/importer"
	"groundseg/ws"
)

func buildMainHTTPHandler() http.Handler {
	mainRouter := http.NewServeMux()
	mainRouter.Handle("/", ContentTypeSetter(http.HandlerFunc(fallbackToIndex(http.FS(webContent)))))
	return mainRouter
}

func buildWebsocketRuntimeHandler() http.Handler {
	wsRouter := http.NewServeMux()
	wsRouter.HandleFunc("/ws", ws.WsHandler)
	wsRouter.HandleFunc("/logs", ws.LogsHandler)
	wsRouter.HandleFunc("/export/", exportRouteHandler)
	wsRouter.HandleFunc("/import/", importRouteHandler)
	return wsRouter
}

func exportRouteHandler(w http.ResponseWriter, r *http.Request) {
	container, ok := parseSinglePathSegment(r.URL.Path, "/export/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	request := mux.SetURLVars(r, map[string]string{
		"container": container,
	})
	exporter.ExportHandler(w, request)
}

func importRouteHandler(w http.ResponseWriter, r *http.Request) {
	session, patp, ok := parseTwoPathSegments(r.URL.Path, "/import/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	importer.HTTPUploadHandler(w, r, session, patp)
}

func parseSinglePathSegment(path, prefix string) (string, bool) {
	parts := splitPath(path, prefix)
	if len(parts) != 1 {
		return "", false
	}
	segment, err := url.PathUnescape(parts[0])
	if err != nil || segment == "" {
		return "", false
	}
	return segment, true
}

func parseTwoPathSegments(path, prefix string) (string, string, bool) {
	parts := splitPath(path, prefix)
	if len(parts) != 2 {
		return "", "", false
	}
	first, firstErr := url.PathUnescape(parts[0])
	second, secondErr := url.PathUnescape(parts[1])
	if firstErr != nil || secondErr != nil || first == "" || second == "" {
		return "", "", false
	}
	return first, second, true
}

func splitPath(path, prefix string) []string {
	trimmedPath := strings.TrimPrefix(path, prefix)
	if trimmedPath == path {
		return []string{}
	}
	return strings.Split(strings.Trim(trimmedPath, "/"), "/")
}
