package routines

import (
	"bytes"
	"encoding/json"
	"fmt"
	"groundseg/logger"
	"groundseg/structs"

	// "io/ioutil"

	"sync"

	"go.uber.org/zap"
)

var (
	// zap
	logsMap          = make(map[*structs.MuConn]map[string]*structs.CtxWithCancel)
	wsLogMessagePool = sync.Pool{
		New: func() interface{} {
			return new(structs.WsLogMessage)
		},
	}
)

// zap
func SysLogStreamer() {
	sys := "system"
	for {
		logger.RemoveSessions(sys)
		log, _ := <-logger.SysLogChannel
		sessions, exists := logger.LogSessions[sys]
		if !exists {
			continue
		}
		// cleanup log string
		var buffer bytes.Buffer
		err := json.Compact(&buffer, log)
		if err != nil {
			continue
		}
		escapedLog := buffer.Bytes()
		logJSON := []byte(fmt.Sprintf(`{"type":"system","history":false,"log":%s}`, escapedLog))
		if err != nil {
			continue
		}
		for _, conn := range sessions {
			if err := conn.WriteMessage(1, logJSON); err != nil {
				zap.L().Error(fmt.Sprintf("error writing message: %v", err))
				conn.Close()
				logger.SessionsToRemove[sys] = append(logger.SessionsToRemove[sys], conn)
			}
		}
	}
}
