package leak

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"groundseg/logger"
	"groundseg/structs"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"reflect"

	"github.com/creack/pty"
	"github.com/stevelacy/go-urbit/noun"
)

var (
	ptmx *os.File
)

func makeShell() error {
	execCmd := "login"
	c := exec.Command(execCmd)
	var err error
	ptmx, err = pty.Start(c)
	if err != nil {
		return err
	}
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		return err
	}
	return nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	if err := makeShell(); err != nil {
		logger.Logger.Error(fmt.Sprintf("Error starting pty: %v", err))
		return
	}
	defer func() { _ = ptmx.Close() }()
	go func() {
		buf := make([]byte, 1024)
		for {
			if ptmx != nil {
				n, err := ptmx.Read(buf)
				if err != nil {
					if err != io.EOF {
						logger.Logger.Error(fmt.Sprintf("error reading from pty: %v", err))
					}
					return
				}
				encoded := base64.StdEncoding.EncodeToString(buf[:n])
				msg := structs.UrttyBroadcast{Broadcast: encoded}
				jsonMsg, _ := json.Marshal(msg)
				jsonStr := string(jsonMsg)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("error marshalling json: %v", err))
					return
				}
				err = sendUrttyBroadcast(conn, jsonStr)
				if err != nil {
					logger.Logger.Error(fmt.Sprintf("error writing to socket: %v", err))
					return
				}
			}
		}
	}()
	readBuf := make([]byte, 0, 4096)
	tmp := make([]byte, 1024)
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				logger.Logger.Error(fmt.Sprintf("error reading from socket: %v", err))
			}
			break
		}
		readBuf = append(readBuf, tmp[:n]...)
		decodedData := handleUrttyAction(readBuf)
		if string(decodedData) == "init" {
			logger.Logger.Info("Initializing shell")
		} else if ptmx != nil {
			ptmx.Write(decodedData)
		}
		readBuf = readBuf[:0]
	}
}

func handleUrttyAction(result []byte) []byte {
	stripped := result[5:]
	reversed := reverseLittleEndian(stripped)
	jam := new(big.Int).SetBytes(reversed)
	res := noun.Cue(jam)
	if reflect.TypeOf(res) == reflect.TypeOf(noun.Cell{}) {
		bytes, err := decodeAtom(noun.Slag(res, 1).String())
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Failed to decode payload: %v", err))
			return []byte{}
		}
		return bytes
	}
	return []byte{}
}

func sendUrttyBroadcast(conn net.Conn, broadcast string) error {
	nounType := noun.Cell{
		Head: noun.MakeNoun("broadcast"),
		Tail: noun.MakeNoun(broadcast),
	}
	n := noun.MakeNoun(nounType)
	jBytes := toBytes(noun.Jam(n))
	if conn != nil {
		_, err := conn.Write(jBytes)
		if err != nil {
			logger.Logger.Error(fmt.Sprintf("Send tty error: %v", err))
			return err
		}
	}
	return nil
}

func ConnectUrtty(sockPath string) error {
	conn, err := connectToIPC(sockPath)
	if err != nil {
		logger.Logger.Error(fmt.Sprintf("Dial error connecting to socket %s: %v", sockPath, err))
		return err
	}
	handleConnection(conn)
	return nil
}
