package luscode

import (
	"errors"
	"groundseg/structs"
	"strings"
	"testing"
	"time"
)

func resetCodeState() {
	codeMutex.Lock()
	lusCodes = make(map[string]structs.ClickLusCode)
	codeMutex.Unlock()
	executeClickCommandForCode = executeLusCode
}

func TestAllowLusCodeRequestFlowControl(t *testing.T) {
	t.Cleanup(resetCodeState)

	patp := "~zod"
	if !allowLusCodeRequest(patp) {
		t.Fatalf("expected request to be allowed for unknown patp")
	}

	codeMutex.Lock()
	lusCodes[patp] = structs.ClickLusCode{LastError: time.Now()}
	codeMutex.Unlock()
	if allowLusCodeRequest(patp) {
		t.Fatalf("expected request to be denied after recent error")
	}

	codeMutex.Lock()
	lusCodes[patp] = structs.ClickLusCode{LastFetch: time.Now(), LusCode: "short"}
	codeMutex.Unlock()
	if !allowLusCodeRequest(patp) {
		t.Fatalf("expected request to be allowed for invalid cached code")
	}

	codeMutex.Lock()
	lusCodes[patp] = structs.ClickLusCode{
		LastFetch: time.Now().Add(-16 * time.Minute),
		LusCode:   strings.Repeat("a", 27),
	}
	codeMutex.Unlock()
	if !allowLusCodeRequest(patp) {
		t.Fatalf("expected request to be allowed for stale cache")
	}

	codeMutex.Lock()
	lusCodes[patp] = structs.ClickLusCode{
		LastFetch: time.Now(),
		LusCode:   strings.Repeat("b", 27),
	}
	codeMutex.Unlock()
	if allowLusCodeRequest(patp) {
		t.Fatalf("expected request to use fresh cached code")
	}
}

func TestGetLusCodeReturnsCachedValue(t *testing.T) {
	t.Cleanup(resetCodeState)

	patp := "~nec"
	expected := strings.Repeat("c", 27)
	codeMutex.Lock()
	lusCodes[patp] = structs.ClickLusCode{LastFetch: time.Now(), LusCode: expected}
	codeMutex.Unlock()

	got, err := GetLusCode(patp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("unexpected code: got %s want %s", got, expected)
	}
}

func TestGetLusCodeCreateFailure(t *testing.T) {
	t.Cleanup(resetCodeState)
	executeClickCommandForCode = func(
		_, _, _, _, _, _ string,
		_ func(string),
	) (string, string, bool, error) {
		return "", "", false, errors.New("write failed")
	}

	_, err := GetLusCode("~bus")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failed to get exec") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetLusCodeExecFailureStoresError(t *testing.T) {
	t.Cleanup(resetCodeState)
	executeClickCommandForCode = func(
		_, _, _, _, _, _ string,
		_ func(string),
	) (string, string, bool, error) {
		return "", "", false, errors.New("exec failed")
	}

	_, err := GetLusCode("~mar")
	if err == nil {
		t.Fatalf("expected error")
	}

	codeMutex.Lock()
	record, ok := lusCodes["~mar"]
	codeMutex.Unlock()
	if !ok || record.LastError.IsZero() {
		t.Fatalf("expected LastError to be recorded")
	}
}

func TestGetLusCodeSuccessStoresFetchedCode(t *testing.T) {
	t.Cleanup(resetCodeState)
	expected := strings.Repeat("z", 27)
	executeClickCommandForCode = func(
		_, _, _, _, _, _ string,
		_ func(string),
	) (string, string, bool, error) {
		return "", expected, true, nil
	}

	got, err := GetLusCode("~pal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != strings.Repeat("z", 27) {
		t.Fatalf("unexpected code: %s", got)
	}

	codeMutex.Lock()
	record := lusCodes["~pal"]
	codeMutex.Unlock()
	if record.LusCode != got || record.LastFetch.IsZero() {
		t.Fatalf("expected code and fetch timestamp to be stored, got %+v", record)
	}
}

func executeLusCode(
	_, _, _, _, _, _ string,
	_ func(string),
) (string, string, bool, error) {
	return "", "", false, nil
}
