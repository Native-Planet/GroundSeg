package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/disk"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// zap and lumberjack
	logChannel = make(chan string, 100)

	// legacy
	logPath        string
	logFile        *os.File
	multiWriter    io.Writer
	Logger         *slog.Logger
	dynamicHandler *DynamicLevelHandler
	ErrBus         = make(chan string, 100)
	filePointers   = make(map[string]struct {
		file   *os.File
		offset int64
	})
)

const (
	LevelInfo  = slog.LevelInfo
	LevelDebug = slog.LevelDebug
)

type MuMultiWriter struct {
	Writers []io.Writer
	Mu      sync.Mutex
}

type DynamicLevelHandler struct {
	currentLevel slog.Leveler
	handler      slog.Handler
}

func NewDynamicLevelHandler(initialLevel slog.Leveler, h slog.Handler) *DynamicLevelHandler {
	return &DynamicLevelHandler{currentLevel: initialLevel, handler: h}
}

func (d *DynamicLevelHandler) SetLevel(newLevel slog.Leveler) {
	d.currentLevel = newLevel
}

func (d *DynamicLevelHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= d.currentLevel.Level()
}

func (d *DynamicLevelHandler) Handle(ctx context.Context, r slog.Record) error {
	return d.handler.Handle(ctx, r)
}

func (d *DynamicLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewDynamicLevelHandler(d.currentLevel, d.handler.WithAttrs(attrs))
}

func (d *DynamicLevelHandler) WithGroup(name string) slog.Handler {
	return NewDynamicLevelHandler(d.currentLevel, d.handler.WithGroup(name))
}

func (d *DynamicLevelHandler) Level() slog.Level {
	return d.currentLevel.Level()
}

type ErrorChannelHandler struct {
	underlyingHandler slog.Handler
}

func NewErrorChannelHandler(handler slog.Handler) *ErrorChannelHandler {
	return &ErrorChannelHandler{underlyingHandler: handler}
}

func (e *ErrorChannelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return e.underlyingHandler.Enabled(ctx, level)
}

func (e *ErrorChannelHandler) Handle(ctx context.Context, r slog.Record) error {
	// If the level is Error, send the message to ErrBus channel
	if r.Level == slog.LevelError {
		ErrBus <- r.Message
	}
	return e.underlyingHandler.Handle(ctx, r)
}

func (e *ErrorChannelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewErrorChannelHandler(e.underlyingHandler.WithAttrs(attrs))
}

func (e *ErrorChannelHandler) WithGroup(name string) slog.Handler {
	return NewErrorChannelHandler(e.underlyingHandler.WithGroup(name))
}

func init() {

	// lumberjack logger
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(makeLogPath(), "zaplog.log"),
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     28,   // days
		Compress:   true, // disabled by default
	}

	// encoder config
	fileWriteSyncer := zapcore.AddSync(lumberjackLogger)
	consoleWriteSyncer := zapcore.AddSync(os.Stdout)
	encoderConfig := zap.NewDevelopmentEncoderConfig()

	// encoder
	fmt.Println(fmt.Sprintf("%+v", &encoderConfig))
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	/*
		for _, arg := range os.Args[1:] {
			// trigger dev mode with `./groundseg dev`
			if arg == "dev" {
				cfg = zap.NewDevelopmentConfig()
				cfg.Encoding = "json"
			}
		}
	*/

	// zap core
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, fileWriteSyncer, zap.InfoLevel),
		zapcore.NewCore(encoder, consoleWriteSyncer, zap.InfoLevel),
		//zapcore.NewCore(encoder, zapcore.AddSync(&ChannelWriter{Channel: logChannel}), zap.InfoLevel),
	)

	// zap
	zap.ReplaceGlobals(zap.Must(zap.New(core, zap.AddCaller()), nil))

	fmt.Println("                                       !G#:\n                                   " +
		" .7G@@@^\n          .                       :J#@@@@P.\n     .75GB#BG57.                ~5&@@" +
		"@&Y^  \n    ?&@@@@@@@@@&J             !G@@@@B?. .^ \n   Y@@@@@@@@@@@@@J         :?B@@@@G!  :" +
		"Y&&:\n   @@@@@@@@@@@@@@B       ^Y&@@@&5^  ~P&@@@:\n   Y@@@@@@@@@@@@@J     !P&@@@#J.  7B@@@@G" +
		"! \n    ?#@@@@@@@@@&?   .7B@@@@G7  :J#@@@&5~   \n     .!YGBBBGY7.  :J#@@@&P~  ^5&@@@#J:  !?." +
		"\n                ^5&@@@#J:  !G@@@@G7. .?B@@:\n             .7G@@@@G7. :J#@@@&P~  ^Y#@@@&:\n" +
		"            .P&&&&G!   ~B&&&#5^   ~#&&&&&P. \n\nＮａｔｉｖｅ Ｐｌａｎｅｔ")
	fmt.Println(" ▄▄ • ▄▄▄        ▄• ▄▌ ▐ ▄ ·▄▄▄▄  .▄▄ · ▄▄▄ . ▄▄ • 𝐯𝟐!\n▐█ ▀ ▪▀▄ █·▪     █▪██▌•" +
		"█▌▐███▪ ██ ▐█ ▀. ▀▄.▀·▐█ ▀ ▪\n▄█ ▀█▄▐▀▀▄  ▄█▀▄ █▌▐█▌▐█▐▐▌▐█· ▐█▌▄▀▀▀█▄▐▀▀▪▄▄█ ▀█▄ 🪐\n▐█▄▪▐█▐" +
		"█•█▌▐█▌.▐▌▐█▄█▌██▐█▌██. ██ ▐█▄▪▐█▐█▄▄▌▐█▄▪▐█\n·▀▀▀▀ .▀  ▀ ▀█▄▀▪ ▀▀▀ ▀▀ █▪▀▀▀▀▀•  ▀▀▀▀  ▀▀▀" +
		" ·▀▀▀▀ (~)")
	logPath = makeLogPath()
	err := os.MkdirAll(logPath, 0755)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to create log directory: %v", err))
		fmt.Println("\n\n.・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.")
		fmt.Println("Please run GroundSeg as root!  \n    /) /)\n   ( . . )" +
			"\n   (  >< )\n Love, Native Planet")
		fmt.Println(".・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n\n")
		panic("")
	}
	logFile, err := os.OpenFile(SysLogfile(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("Failed to open log file: %v", err))
	}
	multiWriter = muMultiWriter(os.Stdout, logFile)
	jsonHandler := slog.NewJSONHandler(multiWriter, nil)
	var level slog.Level
	for _, arg := range os.Args[1:] {
		if arg == "dev" {
			level = LevelDebug
		} else {
			level = LevelInfo
		}
	}
	dynamicHandler = NewDynamicLevelHandler(level, jsonHandler)
	customHandler := NewErrorChannelHandler(dynamicHandler)
	Logger = slog.New(customHandler)
}

func ToggleDebugLogging(enable bool) {
	if enable {
		dynamicHandler.SetLevel(LevelDebug)
	} else {
		dynamicHandler.SetLevel(LevelInfo)
	}
}

func SysLogfile() string {
	currentTime := time.Now()
	return fmt.Sprintf("%s%d-%02d.log", logPath, currentTime.Year(), currentTime.Month())
}

func PrevSysLogfile() string {
	currentTime := time.Now()
	year := currentTime.Year()
	month := currentTime.Month()
	if month == time.January {
		year = year - 1
		month = time.December
	} else {
		month = month - 1
	}
	return fmt.Sprintf("%s%d-%02d.log", logPath, year, month)
}

func muMultiWriter(writers ...io.Writer) *MuMultiWriter {
	return &MuMultiWriter{
		Writers: writers,
	}
}

func (m *MuMultiWriter) Write(p []byte) (n int, err error) {
	m.Mu.Lock()
	defer m.Mu.Unlock()
	var firstError error
	for _, w := range m.Writers {
		n, err := w.Write(p)
		if err != nil && firstError == nil {
			firstError = err
		}
		if n != len(p) && firstError == nil {
			firstError = io.ErrShortWrite
		}
	}
	return len(p), firstError
}

func TailLogs(filename string, n int) ([]string, error) {
	fp, exists := filePointers[filename]
	if !exists || fp.file == nil {
		var err error
		fp.file, err = os.Open(filename)
		if err != nil {
			return nil, err
		}
		fp.offset = 0
		filePointers[filename] = fp
	}
	_, err := fp.file.Seek(fp.offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(fp.file)
	bufSize := 1024
	scanner.Buffer(make([]byte, bufSize), bufSize)
	lineQueue := make([]string, 0, n)
	for scanner.Scan() {
		if len(lineQueue) >= n {
			lineQueue = lineQueue[1:]
		}
		lineQueue = append(lineQueue, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	newOffset, err := fp.file.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	fp.offset = newOffset
	filePointers[filename] = fp
	return lineQueue, nil
}

func makeLogPath() string {
	basePath := os.Getenv("GS_BASE_PATH")
	if basePath == "" {
		basePath = "/opt/nativeplanet/groundseg"
	}
	// check if basePath is an absolute path, if it isn't exit
	if !strings.HasPrefix(basePath, "/") {
		fmt.Println("base path is not absolute! Exiting...")
		os.Exit(1)
	}
	// check if the basePath (or its parents) is a mountpoint with gopsutil
	bpCopy := basePath

	partitions, err := disk.Partitions(true)
	if err != nil {
		fmt.Println("failed to get list of partitions! Exiting...")
		os.Exit(1)
	}

	/*
		the outer loop loops from child up the unix path
		until a mountpoint is found
	*/
	//var mountpoint string
OuterLoop:
	for {
		for _, p := range partitions {
			if p.Mountpoint == bpCopy {
				devType := "mmc"
				if strings.Contains(p.Device, devType) {
					return "/media/data/logs/"
				} else {
					break OuterLoop
				}
			}
		}
		if bpCopy == "/" {
			break
		}
		bpCopy = path.Dir(bpCopy) // Reduce the path by one level
	}
	return basePath + "/logs/"
}
