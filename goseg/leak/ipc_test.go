package leak

import (
	"encoding/binary"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func shortUnixSocketPath(t *testing.T, suffix string) string {
	t.Helper()
	path := filepath.Join("/tmp", "gs-"+suffix+"-"+strconv.FormatInt(time.Now().UnixNano(), 10)+".sock")
	t.Cleanup(func() {
		_ = os.Remove(path)
	})
	return path
}

func TestConnectToIPCRejectsMissingSocket(t *testing.T) {
	socketPath := shortUnixSocketPath(t, "missing")
	if _, err := connectToIPC(socketPath); err == nil {
		t.Fatal("expected connectToIPC to fail for missing socket")
	}
}

func TestConnectToIPCConnectsToUnixSocket(t *testing.T) {
	socketPath := shortUnixSocketPath(t, "listen")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create unix socket listener: %v", err)
	}
	defer listener.Close()

	accepted := make(chan struct{}, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close()
			accepted <- struct{}{}
		}
	}()

	conn, err := connectToIPC(socketPath)
	if err != nil {
		t.Fatalf("connectToIPC returned error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected non-nil connection")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close client connection: %v", err)
	}

	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not observe client connection")
	}
}

func TestInt64ToLittleEndianBytes(t *testing.T) {
	encoded, err := int64ToLittleEndianBytes(0x01020304)
	if err != nil {
		t.Fatalf("int64ToLittleEndianBytes returned error: %v", err)
	}
	want := []byte{0x04, 0x03, 0x02, 0x01}
	if !reflect.DeepEqual(encoded, want) {
		t.Fatalf("unexpected little-endian encoding: got %v want %v", encoded, want)
	}
}

func TestMakeBytesReversesBigIntBytes(t *testing.T) {
	num := new(big.Int).SetBytes([]byte{1, 2, 3, 4})
	got := makeBytes(num)
	want := []byte{4, 3, 2, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected byte order: got %v want %v", got, want)
	}
}

func TestToBytesEncodesVersionLengthAndPayload(t *testing.T) {
	num := new(big.Int).SetBytes([]byte{0x01, 0x02, 0x03})
	encoded := toBytes(num)
	if len(encoded) < 8 {
		t.Fatalf("expected at least 8 bytes in encoded output, got %d", len(encoded))
	}
	if encoded[0] != 0 {
		t.Fatalf("expected version byte 0, got %d", encoded[0])
	}

	length := binary.LittleEndian.Uint32(encoded[1:5])
	if length != 3 {
		t.Fatalf("expected byte length 3, got %d", length)
	}

	wantPayload := []byte{0x03, 0x02, 0x01}
	if !reflect.DeepEqual(encoded[5:], wantPayload) {
		t.Fatalf("unexpected payload encoding: got %v want %v", encoded[5:], wantPayload)
	}
}
