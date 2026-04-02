package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"groundseg/auth"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/dockerclient"
	"groundseg/structs"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type ShellInitPayload struct {
	Patp  string                `json:"patp"`
	Cols  uint                  `json:"cols"`
	Rows  uint                  `json:"rows"`
	Token structs.WsTokenStruct `json:"token"`
}

type ShellClientMessage struct {
	Type  string `json:"type"`
	Input string `json:"input,omitempty"`
	Cols  uint   `json:"cols,omitempty"`
	Rows  uint   `json:"rows,omitempty"`
}

type ShellServerMessage struct {
	Type    string `json:"type"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code,omitempty"`
}

type shellConnWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *shellConnWriter) writeJSON(message ShellServerMessage) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(message)
}

func (w *shellConnWriter) writeBinary(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteMessage(websocket.BinaryMessage, data)
}

func ShellHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not upgrade to websocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	writer := &shellConnWriter{conn: conn}

	_, payload, err := conn.ReadMessage()
	if err != nil {
		zap.L().Error(fmt.Sprintf("shell socket init read failed: %v", err))
		return
	}

	var init ShellInitPayload
	if err := json.Unmarshal(payload, &init); err != nil {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Invalid shell init payload"})
		return
	}

	if !auth.LogTokenCheck(init.Token, r) {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Unauthorized"})
		return
	}

	patp := strings.TrimSpace(init.Patp)
	if patp == "" {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Missing ship name"})
		return
	}

	shipConf := config.UrbitConf(patp)
	if shipConf.PierName == "" {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Ship not found"})
		return
	}
	if !shipConf.DevMode {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Developer mode must be enabled"})
		return
	}
	if _, err := docker.GetContainerRunningStatus(patp); err != nil {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Ship container is not running"})
		return
	}

	cli, err := dockerclient.New()
	if err != nil {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Failed to connect to Docker"})
		return
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	containerID, err := docker.GetContainerIDByName(ctx, cli, patp)
	if err != nil {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Could not find running ship container"})
		return
	}

	execResp, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"tmux", "a"},
	})
	if err != nil {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Failed to start shell session"})
		return
	}

	hijackedResp, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		_ = writer.writeJSON(ShellServerMessage{Type: "error", Message: "Failed to attach shell session"})
		return
	}
	defer hijackedResp.Close()

	if init.Cols > 0 && init.Rows > 0 {
		if err := cli.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
			Width:  init.Cols,
			Height: init.Rows,
		}); err != nil {
			zap.L().Warn(fmt.Sprintf("initial shell resize failed for %s: %v", patp, err))
		}
	}

	if err := writer.writeJSON(ShellServerMessage{Type: "ready"}); err != nil {
		return
	}

	outputDone := make(chan error, 1)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := hijackedResp.Reader.Read(buf)
			if n > 0 {
				chunk := append([]byte(nil), buf[:n]...)
				if writeErr := writer.writeBinary(chunk); writeErr != nil {
					outputDone <- writeErr
					return
				}
			}
			if err != nil {
				if err == io.EOF {
					outputDone <- nil
				} else {
					outputDone <- err
				}
				return
			}
		}
	}()

	inputDone := make(chan error, 1)
	go func() {
		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				inputDone <- err
				return
			}
			if messageType != websocket.TextMessage {
				continue
			}
			var message ShellClientMessage
			if err := json.Unmarshal(payload, &message); err != nil {
				inputDone <- err
				return
			}
			switch message.Type {
			case "input":
				if message.Input == "" {
					continue
				}
				if _, err := hijackedResp.Conn.Write([]byte(message.Input)); err != nil {
					inputDone <- err
					return
				}
			case "resize":
				if message.Cols == 0 || message.Rows == 0 {
					continue
				}
				if err := cli.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
					Width:  message.Cols,
					Height: message.Rows,
				}); err != nil {
					zap.L().Warn(fmt.Sprintf("shell resize failed for %s: %v", patp, err))
				}
			case "close":
				inputDone <- nil
				return
			}
		}
	}()

	select {
	case err := <-inputDone:
		cancel()
		hijackedResp.Close()
		if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived) {
			zap.L().Debug(fmt.Sprintf("shell input closed for %s: %v", patp, err))
		}
	case err := <-outputDone:
		cancel()
		if err != nil {
			zap.L().Debug(fmt.Sprintf("shell output closed for %s: %v", patp, err))
		}
		inspect, inspectErr := cli.ContainerExecInspect(context.Background(), execResp.ID)
		if inspectErr != nil {
			_ = writer.writeJSON(ShellServerMessage{Type: "exit", Message: "Shell session ended"})
			return
		}
		_ = writer.writeJSON(ShellServerMessage{Type: "exit", Code: inspect.ExitCode})
	}
}
