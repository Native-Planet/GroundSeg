package main

// NativePlanet GroundSeg: Go Edition (goseg)
// 🄯 2023 ~nallux-dozryl & ~sitful-hatred
// This is a Golang rewrite of GroundSeg that serves the v2 json
// object via websocket.
// The v2 rewrite decouples the frontend and backend, which makes it
// straightforward to implement alternative backends.
//
// Under development: reimplementing all pyseg functionality.
// Advantages:
// - Really, really fast
// - Event-driven
// - First-class support for concurrent operations
// - Very good golang Docker libraries

import (
	"context"
	"embed"
	"fmt"
	"groundseg/auth"
	"groundseg/broadcast"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/exporter"
	"groundseg/handler/api"
	"groundseg/handler/router"
	"groundseg/importer"
	"groundseg/leak"
	"groundseg/logger"
	"groundseg/rectify"
	"groundseg/routines"
	"groundseg/startram"
	"groundseg/system"
	"groundseg/ws"
	"io/fs"
	"mime"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

var (
	//go:embed web/*
	//go:embed web/_app/*
	//go:embed web/_app/immutable/assets/_*
	//go:embed web/_app/immutable/chunks/_*
	//go:embed web/_app/immutable/entry/_*
	// we need to explicitly embed stuff starting with underscore
	content       embed.FS
	webContent    fs.FS
	fileServer    http.Handler
	capContent    embed.FS
	capFs         = http.FS(capContent)
	capFileServer = http.FileServer(capFs)
	DevMode       = false
	shutdownChan  = make(chan struct{})
	HttpPort      = 80
	netCheckFn    = config.NetCheck
	connCheckFn   = connCheck
	connTimeout   = 30 * time.Second
	connInterval  = 5 * time.Second
	isNPBoxFn     = system.IsNPBox
	c2cModeFn     = system.C2CMode
	setC2CModeFn  = system.SetC2CMode
	killSwitchFn  = killSwitch
)

func init() {
	var err error
	webContent, err = fs.Sub(content, "web")
	if err != nil {
		panic(err)
	}
	fileServer = http.FileServer(http.FS(webContent))
}

// test for internet connectivity and interrupt ServerControl if we need to switch
func C2cCheck() {
	connectivity := config.ConnectivitySettingsSnapshot()
	isNPBox := isNPBoxFn(config.BasePath)
	internetAvailable := connCheckFn()
	if !internetAvailable && system.Device != "" && isNPBox {
		if err := c2cModeFn(); err != nil {
			zap.L().Error(fmt.Sprintf("Error activating C2C mode: %v", err))
		} else {
			zap.L().Info("GroundSeg is in C2C Mode")
			setC2CModeFn(true)
			// start killswitch timer in another routine if c2cInterval in system.json is greater than 0
			if connectivity.C2cInterval > 0 {
				go killSwitchFn()
			}
		}
	}
}

func killSwitch() {
	for {
		connectivity := config.ConnectivitySettingsSnapshot()
		time.Sleep(time.Duration(connectivity.C2cInterval) * time.Second)
		zap.L().Info("Graceful reboot from C2C mode...")
		routines.GracefulShipExit()
		if config.DebugMode {
			zap.L().Debug(fmt.Sprintf("DebugMode detected, skipping shutdown. Exiting program."))
			os.Exit(0)
		} else {
			zap.L().Info(fmt.Sprintf("Rebooting device.."))
			cmd := exec.Command("reboot")
			if err := cmd.Run(); err != nil {
				zap.L().Error(fmt.Sprintf("Failed to reboot device in C2C kill switch: %v", err))
			}
		}
	}
}

func connCheck() bool {
	internetAvailable := netCheckFn("1.1.1.1:53")
	if internetAvailable {
		return true
	}
	timeout := time.After(connTimeout)
	ticker := time.NewTicker(connInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if netCheckFn("1.1.1.1:53") {
				return true
			}
		case <-timeout:
			return false
		}
	}
}

func loadService(loadFunc func() error, errMsg string) {
	go func() {
		if err := loadFunc(); err != nil {
			zap.L().Error(fmt.Sprintf("%s %v", errMsg, err))
		}
	}()
}

// autodetect mime type
func ContentTypeSetter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, "/") {
			next.ServeHTTP(w, r)
			return
		}
		ext := filepath.Ext(r.URL.Path)
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}
		next.ServeHTTP(w, r)
	})
}

func startServer(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	r := mux.NewRouter()
	// r.PathPrefix("/").Handler(ContentTypeSetter(fileServer))
	r.PathPrefix("/").Handler(ContentTypeSetter(http.HandlerFunc(fallbackToIndex(http.FS(webContent)))))
	httpPort := fmt.Sprintf(":%d", HttpPort)
	server := &http.Server{
		Addr:    httpPort,
		Handler: r,
	}
	w := mux.NewRouter()
	w.HandleFunc("/ws", ws.WsHandler)
	w.HandleFunc("/logs", ws.LogsHandler)
	w.HandleFunc("/export/{container}", exporter.ExportHandler)
	w.HandleFunc("/import/{uploadSession}/{patp}", importer.HTTPUploadHandler)
	wsServer := &http.Server{
		Addr:    ":3000",
		Handler: w,
	}
	serverShutdownCh := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zap.L().Error(fmt.Sprintf("HTTP server failed: %v", err))
		}
		serverShutdownCh <- err
	}()
	go func() {
		err := wsServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			zap.L().Error(fmt.Sprintf("Websocket server failed: %v", err))
		}
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			zap.L().Error(fmt.Sprintf("Error shutting down HTTP server: %v", err))
		}
		if err := wsServer.Shutdown(shutdownCtx); err != nil {
			zap.L().Error(fmt.Sprintf("Error shutting down websocket server: %v", err))
		}
	case err := <-serverShutdownCh:
		if err != nil && err != http.ErrServerClosed {
			zap.L().Error(fmt.Sprintf("HTTP server exited unexpectedly: %v", err))
		}
	}
	//go wsServer.ListenAndServe()
	// http.ListenAndServe(":80", r)
	//zap.L().Info("GroundSeg web UI serving")
	//return server
}

func fallbackToIndex(fs http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := fs.Open(r.URL.Path)
		if err != nil {
			r.URL.Path = "/index.html"
		} else {
			defer file.Close()
		}
		http.FileServer(fs).ServeHTTP(w, r)
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger.Initialize()
	if err := config.Initialize(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize config subsystem: %v", err))
		return
	}
	auth.Initialize()
	router.Initialize()
	api.InitializeSupport()
	if err := exporter.Initialize(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize exporter subsystem: %v", err))
	}
	if err := importer.Initialize(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize importer subsystem: %v", err))
	}
	if err := system.InitializeWiFi(); err != nil {
		zap.L().Warn(fmt.Sprintf("Unable to initialize wifi subsystem: %v", err))
	}
	routines.StartMDNSServer()
	routines.StartDockerHealthLoops()
	go routines.PrimeRekorKey()

	// make sure resolved was reenabled
	if err := system.EnableResolved(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to enabled systemd-resolved: %v", err))
	}
	updateSettings := config.UpdateSettingsSnapshot()
	startramSettings := config.StartramSettingsSnapshot()
	internetAvailable := config.NetCheck("1.1.1.1:53")
	zap.L().Info(fmt.Sprintf("Internet available: %t", internetAvailable))
	if err := broadcast.Initialize(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize broadcast subsystem: %v", err))
		return
	}
	if err := docker.Initialize(); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to initialize docker subsystem: %v", err))
		return
	}
	// ongoing connectivity check
	go C2cCheck()
	// async operation to retrieve version info if updates are on
	versionUpdateChannel := make(chan bool)
	remoteVersion := false
	// debug mode
	for _, arg := range os.Args[1:] {
		// trigger dev mode with `./groundseg dev`
		if arg == "dev" {
			zap.L().Info("Starting pprof (:6060)")
			go http.ListenAndServe("0.0.0.0:6060", nil)
		}
		// set non-default port like `--http-port=8060`
		if strings.HasPrefix(arg, "--http-port=") {
			portStr := strings.TrimPrefix(arg, "--http-port=")
			port, err := strconv.Atoi(portStr)
			if err != nil {
				zap.L().Error(fmt.Sprintf("Invalid port number: %s -- defaulting to 80", portStr))
			} else {
				HttpPort = port
			}
			zap.L().Info(fmt.Sprintf("Running HTTP server on port %d", HttpPort))
		}
	}
	// setup swap
	swapSettings := config.SwapSettingsSnapshot()
	zap.L().Info(fmt.Sprintf("Setting up swap %v for %vG", swapSettings.SwapFile, swapSettings.SwapVal))
	if err := system.ConfigureSwap(swapSettings.SwapFile, swapSettings.SwapVal); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to set swap: %v", err))
	}

	// setup /tmp
	zap.L().Info("Setting up /tmp directory")
	if err := system.SetupTmpDir(); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to setup /tmp: %v", err))
	}

	// update mode
	if updateSettings.UpdateMode == "auto" {
		remoteVersion = true
		// get version info from remote server
		go func() {
			_, versionUpdate := config.SyncVersionInfo()
			versionUpdateChannel <- versionUpdate
		}()
		// otherwise use cached if possible, or save hardcoded defaults and use that
	} else {
		versionStruct := config.LocalVersion()
		releaseChannel := updateSettings.UpdateBranch
		targetChan := versionStruct.Groundseg[releaseChannel]
		config.SetVersionChannel(targetChan)
	}
	// routines/version.go
	go routines.CheckVersionLoop() // infinite version check loop
	go routines.AptUpdateLoop()    // check for base OS updates
	// routines/docker.go
	go routines.DockerListenerWithContext(ctx)            // listen to docker daemon
	go routines.DockerSubscriptionHandlerWithContext(ctx) // digest docker events from eventbus

	// digest urbit transition events
	go rectify.UrbitTransitionHandler()
	// digest system transition events
	go rectify.SystemTransitionHandler()
	// digest new ship transition events
	go rectify.NewShipTransitionHandler()
	// digest imported ship transition events
	go rectify.ImportShipTransitionHandler()
	// digest retrieve data
	go rectify.RectifyUrbit()
	// get the startram config from server
	if startramSettings.WgRegistered == true {
		_, err := startram.SyncRetrieve()
		if err != nil {
			zap.L().Warn(fmt.Sprintf("Could not retrieve StarTram/Anchor config: %v", err))
		}
	}
	// gallseg
	go leak.StartLeak() // vere 3.0
	go func() {         // vere 3.0
		for { // vere 3.0
			broadcast.BroadcastToClients() // vere 3.0
			time.Sleep(2 * time.Second)    // vere 3.0
		} // vere 3.0
	}() // vere 3.0
	// fake log temp
	// log stream to frontend
	go routines.SysLogStreamer()
	go routines.DockerLogStreamer()
	go routines.DockerLogConnRemover()
	go routines.OldLogsCleaner()
	// disk usage warning
	go routines.DiskUsageWarning()
	// S.M.A.R.T check
	go routines.SmartDiskCheck()
	// startram reminder
	go routines.StartramRenewalReminder()
	// pack scheduler
	go routines.PackScheduleLoop()
	// chop limiter
	go routines.StartChopRoutines() // vere 3.0
	// backups
	go routines.StartBackupRoutines()
	// block until version info returns
	if remoteVersion == true {
		select {
		case <-versionUpdateChannel:
			zap.L().Info("Version info retrieved")
		case <-time.After(10 * time.Second):
			zap.L().Warn("Could not retrieve version info after 10 seconds!")
			versionStruct := config.LocalVersion()
			releaseChannel := updateSettings.UpdateBranch
			targetChan := versionStruct.Groundseg[releaseChannel]
			config.SetVersionChannel(targetChan)
		}
	}

	// grab wg now cause its huge
	if wgConf, err := docker.GetLatestContainerInfo("wireguard"); err != nil {
		zap.L().Warn(fmt.Sprintf("Error getting WG container: %v", err))
	} else {
		zap.L().Info("Downloading Wireguard image")
		if _, err := docker.PullImageIfNotExist("wireguard", wgConf); err != nil {
			zap.L().Warn(fmt.Sprintf("Error getting WG container: %v", err))
		}
	}
	if startramSettings.WgRegistered == true {
		// Load Wireguard
		if err := docker.LoadWireguard(); err != nil {
			zap.L().Error(fmt.Sprintf("Unable to load Wireguard: %v", err))
		} else {
			// Load MC
			loadService(docker.LoadMC, "Unable to load MinIO Client!")
			// Load MinIOs
			loadService(docker.LoadMinIOs, "Unable to load MinIO containers!")
		}
	}
	// Load Netdata
	loadService(docker.LoadNetdata, "Unable to load Netdata!")
	// Load Urbits
	loadService(docker.LoadUrbits, "Unable to load Urbit ships!")
	// Load Penpai
	loadService(docker.LoadLlama, "Unable to load Llama GPT!")
	startServer(ctx)
}
