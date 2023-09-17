package exporter

import (
	"encoding/json"
	"fmt"
	"goseg/docker"
	"goseg/logger"
	"goseg/structs"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
)

var (
	whitelist = make(map[string]structs.WsTokenStruct)
	exportMu  sync.Mutex
)

func WhitelistContainer(container string, token structs.WsTokenStruct) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	whitelist[container] = token
	logger.Logger.Info(fmt.Sprintf("Whitelisted %v for export", container))
	return nil
}

func RemoveContainerFromWhitelist(container string) error {
	exportMu.Lock()
	defer exportMu.Unlock()
	delete(whitelist, container)
	logger.Logger.Info(fmt.Sprintf("Removed %v from export whitelist", container))
	return nil
}

func ExportHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

	// Handle pre-flight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// if invalid, reject
	invalidSession := func(err error) {
		logger.Logger.Error(fmt.Sprintf("%v", err))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// check if container is whitelisted
	vars := mux.Vars(r)
	container := vars["container"]
	exportMu.Lock()
	whitelistToken, exists := whitelist[container]
	exportMu.Unlock()
	if !exists {
		err := fmt.Errorf("Container %v is not in whitelist!", container)
		invalidSession(err)
	}
	// check if token match
	var tokenData structs.WsTokenStruct
	err := json.NewDecoder(r.Body).Decode(&tokenData)
	if err != nil {
		err := fmt.Errorf("Export failed to decode token: %v", err)
		invalidSession(err)
	}
	if whitelistToken != tokenData {
		err := fmt.Errorf("Token for exporting %v is not valid", container)
		invalidSession(err)
	}
	// compress volume transition compressed %
	// send file
	docker.UTransBus <- structs.UrbitTransition{Patp: container, Type: "exportShip", Event: ""}
}
