package leak

import (
	"context"
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	LeakChan     = make(chan structs.AuthBroadcast)
	BytesChan    = make(map[string]chan string)
	lickStatuses = make(map[string]LickStatus)
	lickMu       sync.RWMutex

	leakNow             = time.Now
	leakSleep           = time.Sleep
	leakConf            = config.Conf
	leakUrbitConf       = config.UrbitConf
	leakLookupEnv       = os.LookupEnv
	leakStat            = os.Stat
	leakMakeSymlink     = makeSymlink
	leakMakeConnection  = makeConnection
	leakListener        = listener
	leakUpdateBroadcast = updateBroadcast
)

func GetLickStatuses() map[string]LickStatus {
	lickMu.RLock()
	defer lickMu.RUnlock()
	cloned := make(map[string]LickStatus, len(lickStatuses))
	for patp, status := range lickStatuses {
		cloned[patp] = status
	}
	return cloned
}

func GetLickChannels() map[string]chan string {
	lickMu.RLock()
	defer lickMu.RUnlock()
	cloned := make(map[string]chan string, len(BytesChan))
	for patp, channel := range BytesChan {
		cloned[patp] = channel
	}
	return cloned
}

func StartLeak() {
	_ = StartLeakWithContext(context.Background())
}

func StartLeakWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	go LookForPortsWithContext(ctx)
	oldBroadcast := structs.AuthBroadcast{}
	lastRcv := leakNow()
	for {
		var newBroadcast structs.AuthBroadcast
		select {
		case <-ctx.Done():
			return nil
		case newBroadcast = <-LeakChan:
		}
		updated, updatedLastRcv, err := processLeakBroadcast(oldBroadcast, lastRcv, newBroadcast)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to update leak broadcast: %v", err))
			continue
		}
		oldBroadcast = updated
		lastRcv = updatedLastRcv
	}
}

func LookForPortsWithContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	// symlink path
	symlinkPath := "/np/d/gs"
	// socket file name
	sockFile := "groundseg.sock"
	// always checking
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			conf := leakConf()
			dockerDir := config.DockerDir()
			// get statuses
			statuses := GetLickStatuses()
			// start dev
			if _, exists := statuses["dev"]; !exists {
				if devSocketPath, exists := leakLookupEnv("SHIP"); exists {
					go connectDevSocket(devSocketPath)
				}
			}
			// check for every ship that exists in groundseg
		for _, patp := range conf.Connectivity.Piers {
				// decide based on existence of info
				info, exists := statuses[patp]
				if exists {
					continue
				}
				// check if sock exists
				sockLocation := filepath.Join(dockerDir, patp, "_data", patp, ".urb", "dev", "groundseg")
				shipConf := leakUrbitConf(patp)
				if shipConf.CustomPierLocation != "" {
					sockLocation = filepath.Join(shipConf.CustomPierLocation, patp, ".urb", "dev", "groundseg")
				}
				sock := filepath.Join(sockLocation, sockFile)
				_, err := leakStat(sock)
				if err != nil {
					// socket doesn't exist
					continue
				}
				// socket exists, shorten path with symlink
				info.Symlink, err = leakMakeSymlink(patp, sockLocation, symlinkPath)
				if err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
					continue
				}
				// Auth is false by default
				info.Auth = false

				// attempt connection to socket
				conn := leakMakeConnection(filepath.Join(info.Symlink, sockFile))
				if conn == nil {
					continue
				}
				zap.L().Info(fmt.Sprintf("Opening lick channel for %s", patp))
				go leakListener(patp, conn, info)
			}
		}
	}
}

func processLeakBroadcast(oldBroadcast structs.AuthBroadcast, lastRcv time.Time, newBroadcast structs.AuthBroadcast) (structs.AuthBroadcast, time.Time, error) {
	currentTime := leakNow()
	if currentTime.Sub(lastRcv) < time.Second {
		return oldBroadcast, lastRcv, nil
	}
	lastRcv = currentTime
	newBroadcast.Type = "structure"
	newBroadcast.AuthLevel = "authorized"
	updatedBroadcast, err := leakUpdateBroadcast(oldBroadcast, newBroadcast)
	if err != nil {
		return oldBroadcast, lastRcv, err
	}
	return updatedBroadcast, lastRcv, nil
}

func LookForPorts() bool {
	LookForPortsWithContext(context.Background())
	return true
}

func connectDevSocket(socketPath string) {
	var info LickStatus
	info.Auth = true
	info.Symlink = socketPath
	conn := leakMakeConnection(filepath.Join(info.Symlink, "groundseg.sock"))
	if conn == nil {
		return
	}
	go leakListener("dev", conn, info)
}
