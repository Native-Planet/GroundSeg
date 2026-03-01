package logger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	LogPath             string
	SysLogChannel       = make(chan []byte, 100)
	SysLogSessions      []*websocket.Conn
	DockerLogSessions   = make(map[string]map[*websocket.Conn]bool)
	SysSessionsToRemove []*websocket.Conn
	loggerInitOnce      sync.Once
)

// File Writer
type FileWriter struct{}

func (fw FileWriter) Write(p []byte) (n int, err error) {
	fileName := SysLogfile()
	// Open the file in append mode, create it if it doesn't exist
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Write the byte slice to the file
	n, err = f.Write(p)
	return n, err
}

// Sync implements the zapcore.WriteSyncer interface for ConsoleWriter.
func (fw FileWriter) Sync() error {
	return nil
}

// Channel Writer
type ChanWriter struct{}

func (cw ChanWriter) Write(p []byte) (n int, err error) {
	SysLogChannel <- p
	return len(p), nil
}

func (cw ChanWriter) Sync() error {
	return nil
}

func Initialize() {
	loggerInitOnce.Do(func() {
		// write logs to file
		fw := FileWriter{}
		fileWriteSyncer := zapcore.AddSync(fw)

		// stdout
		consoleWriteSyncer := zapcore.AddSync(os.Stdout)

		// channel
		cw := ChanWriter{}
		wsWriteSyncer := zapcore.AddSync(cw)

		// encoder config
		encoderConfig := zap.NewDevelopmentEncoderConfig()

		// encoder
		encoder := zapcore.NewJSONEncoder(encoderConfig)

		// trigger dev mode with `./groundseg dev`
		logLevel := zap.InfoLevel
		for _, arg := range os.Args[1:] {
			if arg == "dev" {
				logLevel = zap.DebugLevel
			}
		}

		// zap core
		core := zapcore.NewTee(
			zapcore.NewCore(encoder, fileWriteSyncer, logLevel),
			zapcore.NewCore(encoder, consoleWriteSyncer, logLevel),
			zapcore.NewCore(encoder, wsWriteSyncer, logLevel),
		)

		// instantiate global logger
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

		LogPath = makeLogPath()
		err := os.MkdirAll(LogPath, 0755)
		if err != nil {
			fmt.Printf("Failed to create log directory: %v\n", err)
			fmt.Print("\n\n.・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n")
			fmt.Print("Please run GroundSeg as root!  \n    /) /)\n   ( . . )" +
				"\n   (  >< )\n Love, Native Planet\n")
			fmt.Print(".・。.・゜✭・.・✫・゜・。..・。.・゜✭・.・✫・゜・。.\n\n")
			LogPath = "/tmp/groundseg-logs/"
			if mkErr := os.MkdirAll(LogPath, 0755); mkErr != nil {
				fmt.Printf("Failed to create fallback log directory: %v\n", mkErr)
				return
			}
		}
		zap.L().Info("Starting GroundSeg")
		zap.L().Info("Urbit is love <3")
	})
}

func SysLogfile() string {
	currentTime := time.Now()
	curMonthYear := fmt.Sprintf("%d-%02d", currentTime.Year(), currentTime.Month())
	count := 0
	for {
		// check if y-m-part-n.log exists
		fn := fmt.Sprintf("%s%s-part-%v.log", LogPath, curMonthYear, count)
		// file doesn't exist, use this
		file, err := os.Stat(fn)
		if err != nil {
			return fn
		}
		// check if already 10MB
		const maxSize int64 = 10 * 1024 * 1024
		if file.Size() >= maxSize {
			count = count + 1
			continue
		}
		return fn
	}
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
	return fmt.Sprintf("%s%d-%02d.log", LogPath, year, month)
}

func makeLogPath() string {
	return config.GetStoragePath("logs")
}

func RemoveSysSessions() {
	result := []*websocket.Conn{}
	itemsToRemove := make(map[*websocket.Conn]struct{})

	// Create a set of items to remove for quick lookup
	for _, item := range SysSessionsToRemove {
		itemsToRemove[item] = struct{}{}
	}

	// Iterate over slice1 and add to result if not in itemsToRemove
	for _, item := range SysLogSessions {
		if _, found := itemsToRemove[item]; !found {
			result = append(result, item)
		}
	}
	SysLogSessions = result
	// always clear remove list after running function
	SysSessionsToRemove = []*websocket.Conn{}
}

func getDockerLogs(name string) ([]byte, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return []byte{}, err
	}

	options := container.LogsOptions{ShowStdout: true, ShowStderr: true, Timestamps: true}
	logs, err := cli.ContainerLogs(context.Background(), name, options)
	if err != nil {
		return []byte{}, err
	}
	defer logs.Close()

	var logEntries []string
	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 8 {
			line = line[8:]
		}
		jsonLine, err := json.Marshal(line)
		if err != nil {
			return []byte{}, err
		}
		logEntries = append(logEntries, string(jsonLine))
	}
	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return []byte{}, fmt.Errorf("Error reading docker logs: %v", err)
	}
	jsArray := fmt.Sprintf("[%s]", strings.Join(logEntries, ", "))

	// Print the JavaScript array string
	return []byte(fmt.Sprintf(`{"type":"%s","history":true,"log":%s}`, name, jsArray)), nil
}

func RetrieveSysLogHistory() ([]byte, error) {
	filePath := SysLogfile()
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return []byte{}, fmt.Errorf("Error opening file: %v", err)
	}
	defer file.Close()

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)
	var lines []string

	// Read each line and append it to the lines slice
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return []byte{}, fmt.Errorf("Error reading file: %v", err)
	}

	// Join the lines slice into a single string resembling a JavaScript array
	jsArray := fmt.Sprintf("[%s]", strings.Join(lines, ", "))
	fmt.Println(jsArray)

	// Print the JavaScript array string
	return []byte(fmt.Sprintf(`{"type":"system","history":true,"log":%s}`, jsArray)), nil
}
