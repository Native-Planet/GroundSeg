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
	"groundseg/config"
	"groundseg/docker/orchestration/subsystem"
	"groundseg/logger"
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
	content       embed.FS
	webContent    fs.FS
	fileServer    http.Handler
	capContent    embed.FS
	capFs         = http.FS(capContent)
	capFileServer = http.FileServer(capFs)
	DevMode       = false
	shutdownChan  = make(chan struct{})
	HttpPort      = 80
	initWebErr    error
)

type bootstrapRuntime struct {
	runBootstrapFn func(context.Context, StartupOptions) error
	startServerFn  func(context.Context, int) error
	runC2cCheckFn  func(context.Context)
}

type serverRuntime struct {
	listenAndServe func(*http.Server) error
	shutdown       func(context.Context, *http.Server) error
}

func defaultBootstrapRuntime() bootstrapRuntime {
	return bootstrapRuntime{
		runBootstrapFn: Bootstrap,
		startServerFn: func(ctx context.Context, httpPort int) error {
			return runServer(ctx, httpPort, defaultServerRuntime())
		},
		runC2cCheckFn: func(ctx context.Context) {
			go C2cCheckWith(ctx, defaultC2cRuntime())
		},
	}
}

func bootstrapRuntimeWith(
	runBootstrapFn func(context.Context, StartupOptions) error,
	startServerFn func(context.Context, int) error,
	runC2cCheckFn func(context.Context),
) bootstrapRuntime {
	return bootstrapRuntime{
		runBootstrapFn: runBootstrapFn,
		startServerFn:  startServerFn,
		runC2cCheckFn:  runC2cCheckFn,
	}
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

type connectivityCheckRuntime struct {
	netCheck func(string) bool
	timeout  time.Duration
	interval time.Duration
}

type c2cRuntime struct {
	isNPBox         func() bool
	connCheck       func() bool
	isC2cMode       func() error
	setC2CMode      func(bool) error
	startKillSwitch func(context.Context, func() config.ConnectivitySettings)
	connectivity    func() config.ConnectivitySettings
	hasDevice       func() bool
}

func defaultConnectivityCheckRuntime() connectivityCheckRuntime {
	return connectivityCheckRuntime{
		netCheck: config.NetCheck,
		timeout:  c2cCheckTimeout,
		interval: c2cCheckInterval,
	}
}

func defaultC2cRuntime() c2cRuntime {
	return c2cRuntime{
		isNPBox: func() bool {
			return system.IsNPBox(config.BasePath())
		},
		connCheck:  func() bool { return connCheckWith(defaultConnectivityCheckRuntime()) },
		isC2cMode:  system.C2CMode,
		setC2CMode: system.SetC2CMode,
		startKillSwitch: func(ctx context.Context, connectivitySnapshot func() config.ConnectivitySettings) {
			killSwitch(ctx, connectivitySnapshot)
		},
		connectivity: func() config.ConnectivitySettings {
			return config.ConnectivitySettingsSnapshot()
		},
		hasDevice: system.HasWifiDevice,
	}
}

func C2cCheckWith(ctx context.Context, runtime c2cRuntime) {
	if ctx == nil {
		ctx = context.Background()
	}
	connectivity := runtime.connectivity()
	isNPBox := runtime.isNPBox()
	internetAvailable := runtime.connCheck()
	if !internetAvailable && runtime.hasDevice() && isNPBox {
		if err := runtime.isC2cMode(); err != nil {
			logger.Errorf("Error activating C2C mode: %v", err)
		} else {
			logger.Info("GroundSeg is in C2C Mode")
			_ = runtime.setC2CMode(true)
			// start killswitch timer in another routine if c2cInterval in system.json is greater than 0
			if connectivity.C2cInterval > 0 {
				go runtime.startKillSwitch(ctx, runtime.connectivity)
			}
		}
	}
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
	if ctx == nil {
		ctx = context.Background()
	}
	if connectivitySnapshot == nil {
		connectivitySnapshot = config.ConnectivitySettingsSnapshot
	}
	if config.DebugMode() {
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
	return connCheckWith(defaultConnectivityCheckRuntime())
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
			if err := runtime.shutdown(shutdownCtx, server); err != nil {
				logger.Errorf("Error shutting down HTTP server: %v", err)
			}
			if err := runtime.shutdown(shutdownCtx, wsServer); err != nil {
				logger.Errorf("Error shutting down websocket server: %v", err)
			}
			return nil
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
			defer file.Close()
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
	go http.ListenAndServe("0.0.0.0:6060", nil)
}

func startupRuntimeFromAppOptions(_ appStartupOptions) startupRuntime {
	return startupRuntime{}
}

func runBootstrapSubsystems(ctx context.Context, opts appStartupOptions, runtime bootstrapRuntime) error {
	if runtime.runBootstrapFn == nil {
		return fmt.Errorf("bootstrap runtime is not configured")
	}
	if runtime.startServerFn == nil {
		return fmt.Errorf("bootstrap runtime is not configured")
	}
	return runtime.runBootstrapFn(ctx, StartupOptions{
		HTTPPort: opts.httpPort,
		ValidateConfig: func() error {
			return initWebErr
		},
		StartServer: func(ctx context.Context, httpPort int) error {
			return runtime.startServerFn(ctx, httpPort)
		},
		StartC2cCheck: func(ctx context.Context) {
			if runtime.runC2cCheckFn != nil {
				runtime.runC2cCheckFn(ctx)
			}
		},
		StartupRuntime: startupRuntimeFromAppOptions(opts),
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
	if err := runBootstrapSubsystems(ctx, startupOptions, defaultBootstrapRuntime()); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
