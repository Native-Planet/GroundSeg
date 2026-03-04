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
	"errors"
	"fmt"
	"groundseg/config"
	"groundseg/docker/orchestration/subsystem"
	"groundseg/internal/resource"
	"groundseg/logger"
	"groundseg/startuporchestrator"
	"groundseg/system"
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
)

var (
	//go:embed web/*
	//go:embed web/_app/*
	//go:embed web/_app/immutable/assets/_*
	//go:embed web/_app/immutable/chunks/_*
	//go:embed web/_app/immutable/entry/_*
	// we need to explicitly embed stuff starting with underscore
	content                      embed.FS
	webContent                   fs.FS
	fileServer                   http.Handler
	capContent                   embed.FS
	capFs                        = http.FS(capContent)
	capFileServer                = http.FileServer(capFs)
	DevMode                      = false
	shutdownChan                 = make(chan struct{})
	HttpPort                     = 80
	initWebErr                   error
	runtimeContextSnapshotFn     = config.RuntimeContextSnapshot
	connectivitySettingsSnapshot = config.ConnectivitySettingsSnapshot
	netCheckFn                   = config.NetCheck
	debugModeFn                  = config.DebugMode
)

type bootstrapRuntime struct {
	BootstrapFn   func(context.Context, startuporchestrator.StartupOptions) error `runtime:"bootstrap"`
	StartServerFn func(context.Context, int) error                                `runtime:"bootstrap"`
	RunC2cCheckFn func(context.Context) error                                     `runtime:"bootstrap"`
}

func defaultBootstrapRuntime() bootstrapRuntime {
	return bootstrapRuntime{
		BootstrapFn: startuporchestrator.Bootstrap,
		StartServerFn: func(ctx context.Context, httpPort int) error {
			return runServer(ctx, httpPort, defaultServerRuntime())
		},
		RunC2cCheckFn: func(ctx context.Context) error {
			return C2cCheckWith(ctx, defaultC2cRuntime())
		},
	}
}

func bootstrapRuntimeWith(
	runBootstrapFn func(context.Context, startuporchestrator.StartupOptions) error,
	runServerFn func(context.Context, int) error,
	RunC2cCheckFn func(context.Context) error,
) bootstrapRuntime {
	runtime := defaultBootstrapRuntime()
	if runBootstrapFn != nil {
		runtime.BootstrapFn = runBootstrapFn
	}
	if runServerFn != nil {
		runtime.StartServerFn = runServerFn
	}
	if RunC2cCheckFn != nil {
		runtime.RunC2cCheckFn = RunC2cCheckFn
	}
	return runtime
}

func (runtime bootstrapRuntime) validate() error {
	if runtime.BootstrapFn == nil {
		return fmt.Errorf("bootstrap callback is required")
	}
	if runtime.StartServerFn == nil {
		return fmt.Errorf("startServer callback is required")
	}
	if runtime.RunC2cCheckFn == nil {
		return fmt.Errorf("runC2cCheck callback is required")
	}
	return nil
}

func (runtime bootstrapRuntime) bootstrap(ctx context.Context, opts startuporchestrator.StartupOptions) error {
	if runtime.BootstrapFn == nil {
		return fmt.Errorf("bootstrap callback is not configured")
	}
	return runtime.BootstrapFn(ctx, opts)
}

func (runtime bootstrapRuntime) startServer(ctx context.Context, httpPort int) error {
	if runtime.StartServerFn == nil {
		return fmt.Errorf("startServer callback is not configured")
	}
	return runtime.StartServerFn(ctx, httpPort)
}

func (runtime bootstrapRuntime) runC2cCheck(ctx context.Context) error {
	if runtime.RunC2cCheckFn == nil {
		return fmt.Errorf("runC2cCheck callback is not configured")
	}
	return runtime.RunC2cCheckFn(ctx)
}

type serverRuntime struct {
	listenAndServe func(*http.Server) error
	shutdown       func(context.Context, *http.Server) error
}

func defaultServerRuntime() serverRuntime {
	return serverRuntime{
		listenAndServe: func(server *http.Server) error {
			return server.ListenAndServe()
		},
		shutdown: func(ctx context.Context, server *http.Server) error {
			return server.Shutdown(ctx)
		},
	}
}

const (
	cloudCheckHost   = "1.1.1.1:53"
	c2cCheckTimeout  = 30 * time.Second
	c2cCheckInterval = 5 * time.Second
)

type c2cRuntimeContext struct {
	basePath               string
	netCheck               func(string) bool
	connectivitySnapshotFn func() config.ConnectivitySettings
	isNPBoxFn              func(string) bool
	isDebugModeFn          func() bool
}

func defaultC2cRuntimeContext() c2cRuntimeContext {
	context := runtimeContextSnapshotFn()
	return c2cRuntimeContext{
		basePath:               context.BasePath,
		netCheck:               netCheckFn,
		connectivitySnapshotFn: connectivitySettingsSnapshot,
		isNPBoxFn:              system.IsNPBox,
		isDebugModeFn:          debugModeFn,
	}
}

type connectivityCheckRuntime struct {
	netCheck func(string) bool
	timeout  time.Duration
	interval time.Duration
}

type c2cRuntime struct {
	connectivity c2cConnectivityRuntime
	device       c2cDeviceRuntime
	mode         c2cModeRuntime
}

type c2cConnectivityRuntime struct {
	connCheck    func() bool
	settingsSnap func() config.ConnectivitySettings
}

type c2cDeviceRuntime struct {
	isNPBox   func() bool
	hasDevice func() bool
	wifiInfo  func() (bool, error)
}

type c2cModeRuntime struct {
	isC2cMode       func() error
	setC2cMode      func(bool) error
	startKillSwitch func(context.Context, func() config.ConnectivitySettings)
}

func defaultConnectivityCheckRuntime(netCheck func(string) bool) connectivityCheckRuntime {
	if netCheck == nil {
		netCheck = netCheckFn
	}
	return connectivityCheckRuntime{
		netCheck: netCheck,
		timeout:  c2cCheckTimeout,
		interval: c2cCheckInterval,
	}
}

func defaultC2cRuntime() c2cRuntime {
	runtimeContext := defaultC2cRuntimeContext()
	return c2cRuntime{
		connectivity: c2cConnectivityRuntime{
			connCheck: func() bool {
				return connCheckWith(defaultConnectivityCheckRuntime(runtimeContext.netCheck))
			},
			settingsSnap: func() config.ConnectivitySettings {
				if runtimeContext.connectivitySnapshotFn == nil {
					return config.ConnectivitySettingsSnapshot()
				}
				return runtimeContext.connectivitySnapshotFn()
			},
		},
		device: c2cDeviceRuntime{
			isNPBox: func() bool {
				if runtimeContext.isNPBoxFn == nil {
					return system.IsNPBox(runtimeContext.basePath)
				}
				return runtimeContext.isNPBoxFn(runtimeContext.basePath)
			},
			hasDevice: system.HasWifiDevice,
			wifiInfo:  system.NewWiFiRuntimeService().IsWifiEnabled,
		},
		mode: c2cModeRuntime{
			isC2cMode:  func() error { return system.NewC2CModeFlow().EnterC2CMode() },
			setC2cMode: system.SetC2CMode,
			startKillSwitch: func(ctx context.Context, connectivitySnapshot func() config.ConnectivitySettings) {
				killSwitchWithMode(ctx, connectivitySnapshot, runtimeContext.isDebugModeFn)
			},
		},
	}
}

func C2cCheckWith(ctx context.Context, runtime c2cRuntime) error {
	if ctx == nil {
		ctx = context.Background()
	}
	connectivity := runtime.connectivity.settingsSnap()
	isNPBox := runtime.device.isNPBox()
	wifiEnabled := true
	wifiErr := error(nil)
	internetAvailable := runtime.connectivity.connCheck()
	if !internetAvailable && runtime.device.hasDevice() && isNPBox {
		if runtime.device.wifiInfo != nil {
			wifiEnabled, wifiErr = runtime.device.wifiInfo()
			if wifiErr != nil {
				logger.Warnf("Failed to read wifi radio state: %v", wifiErr)
				return fmt.Errorf("unable to read wifi radio state for C2C mode check: %w", wifiErr)
			}
		}
		if !wifiEnabled {
			return nil
		}
		if err := runtime.mode.isC2cMode(); err != nil {
			return fmt.Errorf("unable to detect/enter C2C mode: %w", err)
		} else {
			logger.Info("GroundSeg is in C2C Mode")
			if err := runtime.mode.setC2cMode(true); err != nil {
				return fmt.Errorf("unable to set C2C mode: %w", err)
			}
			// start killswitch timer in another routine if c2cInterval in system.json is greater than 0
			if connectivity.C2cInterval > 0 {
				go runtime.mode.startKillSwitch(ctx, runtime.connectivity.settingsSnap)
			}
		}
	}
	return nil
}

func init() {
	var err error
	webContent, err = fs.Sub(content, "web")
	if err != nil {
		initWebErr = err
		return
	}
	fileServer = http.FileServer(http.FS(webContent))
}

// test for internet connectivity and interrupt ServerControl if we need to switch
func killSwitch(ctx context.Context, connectivitySnapshot func() config.ConnectivitySettings) {
	killSwitchWithMode(ctx, connectivitySnapshot, debugModeFn)
}

func killSwitchWithMode(ctx context.Context, connectivitySnapshot func() config.ConnectivitySettings, isDebugMode func() bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	if connectivitySnapshot == nil {
		connectivitySnapshot = config.ConnectivitySettingsSnapshot
	}
	if isDebugMode != nil && isDebugMode() {
		logger.Debug("Debug mode enabled; skipping killswitch reboot path")
		return
	}
	for {
		connectivity := connectivitySnapshot()
		delay := time.Duration(connectivity.C2cInterval) * time.Second
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
		logger.Info("Graceful reboot from C2C mode...")
		subsystem.GracefulShipExit()
		logger.Infof("Rebooting device..")
		cmd := exec.Command("reboot")
		if err := cmd.Run(); err != nil {
			logger.Errorf("Failed to reboot device in C2C kill switch: %v", err)
		}
	}
}

func connCheckWith(runtime connectivityCheckRuntime) bool {
	internetAvailable := runtime.netCheck(cloudCheckHost)
	if internetAvailable {
		return true
	}
	timeout := time.After(runtime.timeout)
	ticker := time.NewTicker(runtime.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if runtime.netCheck(cloudCheckHost) {
				return true
			}
		case <-timeout:
			return false
		}
	}
}

func connCheck() bool {
	return connCheckWith(defaultConnectivityCheckRuntime(netCheckFn))
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

type listenerError struct {
	name string
	err  error
}

func runServer(ctx context.Context, httpPort int, runtime serverRuntime) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if runtime.listenAndServe == nil {
		return fmt.Errorf("server runtime missing listenAndServe function")
	}
	if runtime.shutdown == nil {
		return fmt.Errorf("server runtime missing shutdown function")
	}
	r := buildMainHTTPHandler()
	httpAddress := fmt.Sprintf(":%d", httpPort)
	server := &http.Server{
		Addr:    httpAddress,
		Handler: r,
	}
	w := buildWebsocketRuntimeHandler()
	wsServer := &http.Server{
		Addr:    ":3000",
		Handler: w,
	}
	listenerErrCh := make(chan listenerError, 2)
	serverStartedCh := make(chan string, 2)

	startListener := func(name string, srv *http.Server, run func(*http.Server) error) {
		go func() {
			serverStartedCh <- name
			err := run(srv)
			listenerErrCh <- listenerError{name: name, err: err}
		}()
	}

	startListener("http", server, runtime.listenAndServe)
	startListener("websocket", wsServer, runtime.listenAndServe)

	for i := 0; i < 2; i++ {
		<-serverStartedCh
	}

	shutdownServers := func() {
		if err := runtime.shutdown(context.Background(), server); err != nil {
			logger.Errorf("Error shutting down HTTP server: %v", err)
		}
		if err := runtime.shutdown(context.Background(), wsServer); err != nil {
			logger.Errorf("Error shutting down websocket server: %v", err)
		}
	}

	httpDone := false
	wsDone := false
	var unexpectedErr error

	for {
		select {
		case <-ctx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			shutdownErrs := make([]error, 0, 2)
			if err := runtime.shutdown(shutdownCtx, server); err != nil {
				logger.Errorf("Error shutting down HTTP server: %v", err)
				shutdownErrs = append(shutdownErrs, fmt.Errorf("error shutting down HTTP server: %w", err))
			}
			if err := runtime.shutdown(shutdownCtx, wsServer); err != nil {
				logger.Errorf("Error shutting down websocket server: %v", err)
				shutdownErrs = append(shutdownErrs, fmt.Errorf("error shutting down websocket server: %w", err))
			}
			if len(shutdownErrs) == 0 {
				return nil
			}
			return errors.Join(shutdownErrs...)
		case listener := <-listenerErrCh:
			switch listener.name {
			case "http":
				httpDone = true
			case "websocket":
				wsDone = true
			}
			if listener.err != nil && listener.err != http.ErrServerClosed && unexpectedErr == nil {
				logger.Errorf("%s server exited unexpectedly: %v", listener.name, listener.err)
				unexpectedErr = fmt.Errorf("%s server exited unexpectedly: %w", listener.name, listener.err)
			}
			if httpDone && wsDone {
				return unexpectedErr
			}
			shutdownServers()
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
			defer func() {
				if closeErr := resource.JoinCloseError(nil, file, "close embedded static file"); closeErr != nil {
					logger.Errorf("failed to close embedded static file: %v", closeErr)
				}
			}()
		}
		http.FileServer(fs).ServeHTTP(w, r)
	}
}

type appStartupOptions struct {
	httpPort int
	devMode  bool
}

func parseStartupOptions(args []string) appStartupOptions {
	opts := appStartupOptions{
		httpPort: HttpPort,
	}
	for _, arg := range args {
		switch {
		case arg == "dev":
			opts.devMode = true
		case strings.HasPrefix(arg, "--http-port="):
			portStr := strings.TrimPrefix(arg, "--http-port=")
			port, err := strconv.Atoi(portStr)
			if err != nil {
				logger.Errorf("Invalid port number: %s -- defaulting to %d", portStr, HttpPort)
				continue
			}
			opts.httpPort = port
		}
	}
	if opts.httpPort <= 0 {
		opts.httpPort = HttpPort
	}
	DevMode = opts.devMode
	HttpPort = opts.httpPort
	return opts
}

func startDevMode(opts appStartupOptions) {
	if !opts.devMode {
		return
	}
	logger.Info("Starting pprof (:6060)")
	go func() {
		if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil && err != http.ErrServerClosed {
			logger.Errorf("pprof server failed: %v", err)
		}
	}()
}

func runBootstrapSubsystems(ctx context.Context, opts appStartupOptions, runtime bootstrapRuntime) error {
	runtime = bootstrapRuntimeWith(
		runtime.BootstrapFn,
		runtime.StartServerFn,
		runtime.RunC2cCheckFn,
	)
	if err := runtime.validate(); err != nil {
		return err
	}
	return runtime.bootstrap(ctx, startuporchestrator.StartupOptions{
		HTTPPort: opts.httpPort,
		ValidateConfig: func() error {
			return initWebErr
		},
		StartServer:    runtime.startServer,
		StartC2cCheck:  runtime.runC2cCheck,
		StartupRuntime: startuporchestrator.StartupRuntime{},
	})
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	if err := logger.Initialize(); err != nil {
		fmt.Printf("Logger initialization warning: %v\n", err)
	}
	startupOptions := parseStartupOptions(os.Args[1:])
	startDevMode(startupOptions)
	if err := runBootstrapSubsystems(ctx, startupOptions, bootstrapRuntimeWith(nil, nil, nil)); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
