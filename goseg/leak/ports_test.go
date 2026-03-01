package leak

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMakeSymlinkCreatesAndReusesMatchingLink(t *testing.T) {
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	linkRoot := filepath.Join(t.TempDir(), "links")

	linkPath, err := makeSymlink("~zod", targetDir, linkRoot)
	if err != nil {
		t.Fatalf("makeSymlink returned error: %v", err)
	}
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("failed to read created symlink: %v", err)
	}
	if target != targetDir {
		t.Fatalf("unexpected symlink target: got %q want %q", target, targetDir)
	}

	reusedPath, err := makeSymlink("~zod", targetDir, linkRoot)
	if err != nil {
		t.Fatalf("makeSymlink(reuse) returned error: %v", err)
	}
	if reusedPath != linkPath {
		t.Fatalf("expected symlink path reuse: got %q want %q", reusedPath, linkPath)
	}
}

func TestMakeSymlinkReplacesMismatchedLink(t *testing.T) {
	baseDir := t.TempDir()
	targetA := filepath.Join(baseDir, "target-a")
	targetB := filepath.Join(baseDir, "target-b")
	if err := os.MkdirAll(targetA, 0o755); err != nil {
		t.Fatalf("failed to create target-a: %v", err)
	}
	if err := os.MkdirAll(targetB, 0o755); err != nil {
		t.Fatalf("failed to create target-b: %v", err)
	}
	linkRoot := filepath.Join(baseDir, "links")
	linkPath := filepath.Join(linkRoot, "~zod")
	if err := os.MkdirAll(linkRoot, 0o755); err != nil {
		t.Fatalf("failed to create link root: %v", err)
	}
	if err := os.Symlink(targetA, linkPath); err != nil {
		t.Fatalf("failed to seed existing symlink: %v", err)
	}

	resultPath, err := makeSymlink("~zod", targetB, linkRoot)
	if err != nil {
		t.Fatalf("makeSymlink returned error: %v", err)
	}
	if resultPath != linkPath {
		t.Fatalf("unexpected link path: got %q want %q", resultPath, linkPath)
	}
	newTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("failed to read replaced symlink: %v", err)
	}
	if newTarget != targetB {
		t.Fatalf("expected symlink target replacement to %q, got %q", targetB, newTarget)
	}
}

func TestMakeSymlinkReplacesExistingFile(t *testing.T) {
	baseDir := t.TempDir()
	targetDir := filepath.Join(baseDir, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	linkRoot := filepath.Join(baseDir, "links")
	if err := os.MkdirAll(linkRoot, 0o755); err != nil {
		t.Fatalf("failed to create link root: %v", err)
	}
	linkPath := filepath.Join(linkRoot, "~zod")
	if err := os.WriteFile(linkPath, []byte("not-a-symlink"), 0o644); err != nil {
		t.Fatalf("failed to seed regular file: %v", err)
	}

	if _, err := makeSymlink("~zod", targetDir, linkRoot); err != nil {
		t.Fatalf("makeSymlink returned error: %v", err)
	}
	if info, err := os.Lstat(linkPath); err != nil {
		t.Fatalf("failed to stat resulting link: %v", err)
	} else if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected resulting path to be a symlink, mode=%v", info.Mode())
	}
}

func TestMakeConnectionHandlesMissingAndAvailableSockets(t *testing.T) {
	missing := shortUnixSocketPath(t, "ports-missing")
	if conn := makeConnection(missing); conn != nil {
		t.Fatal("expected nil connection for missing socket")
	}

	socketPath := shortUnixSocketPath(t, "ports-live")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen on unix socket: %v", err)
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

	conn := makeConnection(socketPath)
	if conn == nil {
		t.Fatal("expected connection for live socket")
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close client connection: %v", err)
	}

	select {
	case <-accepted:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not accept unix socket connection")
	}
}
