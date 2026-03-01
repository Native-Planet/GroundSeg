package storage

import (
	"errors"
	"strings"
	"testing"

	"groundseg/structs"
)

func resetStorageSeams() {
	executeClickCommandForStorage = executeStorageCommand
}

func TestUnlinkStorageBuildsClearPayload(t *testing.T) {
	t.Cleanup(resetStorageSeams)

	var gotFile, gotHoon, gotOp string
	executeClickCommandForStorage = func(_, file, hoon, _, _, operation string) (string, error) {
		gotFile = file
		gotHoon = hoon
		gotOp = operation
		return "ok", nil
	}

	if err := UnlinkStorage("~zod"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFile != "unlinkstorage" || gotOp != "Click unlink storage" {
		t.Fatalf("unexpected call values: file=%s op=%s", gotFile, gotOp)
	}
	for _, token := range []string{"%set-endpoint ''", "%set-access-key-id ''", "%set-secret-access-key ''", "%set-current-bucket ''"} {
		if !strings.Contains(gotHoon, token) {
			t.Fatalf("missing token %q in hoon: %s", token, gotHoon)
		}
	}
}

func TestLinkStorageBuildsAccountPayload(t *testing.T) {
	t.Cleanup(resetStorageSeams)

	var gotFile, gotHoon, gotOp string
	executeClickCommandForStorage = func(_, file, hoon, _, _, operation string) (string, error) {
		gotFile = file
		gotHoon = hoon
		gotOp = operation
		return "ok", nil
	}

	acc := structs.MinIOServiceAccount{AccessKey: "ak", SecretKey: "sk"}
	if err := LinkStorage("~bus", "https://s3.example", acc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotFile != "linkstorage" || gotOp != "Click link storage" {
		t.Fatalf("unexpected call values: file=%s op=%s", gotFile, gotOp)
	}
	for _, token := range []string{"%set-endpoint 'https://s3.example'", "%set-access-key-id 'ak'", "%set-secret-access-key 'sk'", "%set-current-bucket 'bucket'"} {
		if !strings.Contains(gotHoon, token) {
			t.Fatalf("missing token %q in hoon: %s", token, gotHoon)
		}
	}
}

func TestStorageCommandsBubbleErrors(t *testing.T) {
	t.Cleanup(resetStorageSeams)
	executeClickCommandForStorage = func(_, _, _, _, _, _ string) (string, error) {
		return "", errors.New("failed")
	}

	if err := UnlinkStorage("~nec"); err == nil {
		t.Fatalf("expected unlinkStorage error")
	}
	if err := LinkStorage("~nec", "endpoint", structs.MinIOServiceAccount{}); err == nil {
		t.Fatalf("expected linkStorage error")
	}
}

func executeStorageCommand(_, _, _, _, _, _ string) (string, error) {
	return "ok", nil
}
