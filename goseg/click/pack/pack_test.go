package pack

import (
	"errors"
	"strings"
	"testing"
)

func resetPackSeams() {
	executeClickCommandForPack = executePackCommand
}

func TestSendPackBuildsExpectedCommand(t *testing.T) {
	t.Cleanup(resetPackSeams)

	var gotPatp, gotFile, gotHoon, gotSource, gotToken, gotOp string
	executeClickCommandForPack = func(patp, file, hoon, sourcePath, successToken, operation string) (string, error) {
		gotPatp = patp
		gotFile = file
		gotHoon = hoon
		gotSource = sourcePath
		gotToken = successToken
		gotOp = operation
		return "ok", nil
	}

	if err := SendPack("~zod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPatp != "~zod" || gotFile != "pack" || gotSource != "" || gotToken != "success" || gotOp != "Click |pack" {
		t.Fatalf("unexpected call args: %q %q %q %q %q", gotPatp, gotFile, gotSource, gotToken, gotOp)
	}
	if !strings.Contains(gotHoon, "%pack") {
		t.Fatalf("expected hoon to include pack action: %s", gotHoon)
	}
}

func TestSendPackBubblesError(t *testing.T) {
	t.Cleanup(resetPackSeams)
	executeClickCommandForPack = func(string, string, string, string, string, string) (string, error) {
		return "", errors.New("failed")
	}

	if err := SendPack("~bus"); err == nil {
		t.Fatalf("expected error")
	}
}

func executePackCommand(_, _, _, _, _, _ string) (string, error) {
	return "ok", nil
}
