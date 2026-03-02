package leak

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
	"time"

	"groundseg/leakchannel"
	"groundseg/structs"

	"github.com/stevelacy/go-urbit/noun"
)

func mustReceiveLeakAction(t *testing.T, name string) leakchannel.ActionChannel {
	t.Helper()
	select {
	case action := <-leakchannel.LeakAction:
		return action
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for leak action: %s", name)
		return leakchannel.ActionChannel{}
	}
}

func waitForAuthStatus(t *testing.T, patp string, want bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, exists := GetLickStatuses()[patp]
		if exists && status.Auth == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for auth state %v for %s", want, patp)
}

func makeLickPacket(payload []byte) []byte {
	cell := noun.Cell{
		Head: noun.MakeNoun("event"),
		Tail: noun.MakeNoun(string(payload)),
	}
	return toBytes(noun.Jam(noun.MakeNoun(cell)))
}

func makeAtomPacket(value int64) []byte {
	return toBytes(noun.Jam(noun.MakeNoun(big.NewInt(value))))
}

func TestReverseLittleEndian(t *testing.T) {
	input := []byte{1, 2, 3, 4}
	got := reverseLittleEndian(input)
	want := []byte{4, 3, 2, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected reverse result: got %v want %v", got, want)
	}
}

func TestDecodeAtom(t *testing.T) {
	original := []byte("decode-me")
	reversed := append([]byte(nil), original...)
	reverseLittleEndian(reversed)
	atom := new(big.Int).SetBytes(reversed).String()

	got, err := decodeAtom(atom)
	if err != nil {
		t.Fatalf("decodeAtom returned error: %v", err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("unexpected decoded bytes: got %q want %q", got, original)
	}
}

func TestDecodeAtomRejectsInvalidInput(t *testing.T) {
	if _, err := decodeAtom("not-a-number"); err == nil {
		t.Fatal("expected decodeAtom to fail for invalid decimal input")
	}
}

func TestProcessActionReturnsErrorForInvalidJSON(t *testing.T) {
	if err := processAction("~zod", []byte("{")); err == nil {
		t.Fatal("expected processAction to fail for invalid payload json")
	}
}

func TestProcessActionNoopsForUnknownPatp(t *testing.T) {
	resetLeakStateForTest(t)
	payload := []byte(`{"payload":{"type":"poke"}}`)

	if err := processAction("~zod", payload); err != nil {
		t.Fatalf("expected nil error for unknown patp, got %v", err)
	}
}

func TestProcessActionLogoutMarksShipUnauthenticated(t *testing.T) {
	resetLeakStateForTest(t)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: true}})

	payload := []byte(`{"payload":{"type":"logout"}}`)
	if err := processAction("~zod", payload); err != nil {
		t.Fatalf("processAction(logout) returned error: %v", err)
	}

	status := GetLickStatuses()["~zod"]
	if status.Auth {
		t.Fatal("expected logout to clear auth state")
	}
}

func TestProcessActionForwardsDefaultActions(t *testing.T) {
	resetLeakStateForTest(t)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: false}})

	payload := []byte(`{"payload":{"type":"poke"}}`)
	errCh := make(chan error, 1)
	go func() {
		errCh <- processAction("~zod", payload)
	}()

	action := mustReceiveLeakAction(t, "default action")
	if action.Patp != "~zod" || action.Type != "poke" || action.Auth {
		t.Fatalf("unexpected leak action: %+v", action)
	}
	if !reflect.DeepEqual(action.Content, payload) {
		t.Fatalf("unexpected forwarded payload: got %s want %s", action.Content, payload)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("processAction(default) returned error: %v", err)
	}
}

func TestProcessActionPasswordForwardsAndHandlesLogoutSignal(t *testing.T) {
	resetLeakStateForTest(t)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: true}})

	payload := []byte(`{"payload":{"type":"password"}}`)
	errCh := make(chan error, 1)
	go func() {
		errCh <- processAction("~zod", payload)
	}()

	action := mustReceiveLeakAction(t, "password action")
	if action.Patp != "~zod" || action.Type != "password" || !action.Auth {
		t.Fatalf("unexpected leak action: %+v", action)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("processAction(password) returned error: %v", err)
	}

	go func() {
		leakchannel.Logout <- struct{}{}
	}()
	waitForAuthStatus(t, "~zod", false)
}

func TestUrbitLogoutClearsAuth(t *testing.T) {
	resetLeakStateForTest(t)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: true}})

	urbitLogout("~zod")

	if GetLickStatuses()["~zod"].Auth {
		t.Fatal("expected urbitLogout to clear auth")
	}
}

func TestUrbitLoginSendsResponseEvent(t *testing.T) {
	resetLeakStateForTest(t)
	BytesChan["~zod"] = make(chan string, 1)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: false}})

	payload := structs.WsLoginPayload{}
	payload.Payload.Type = "login"
	payload.Payload.Password = "definitely-not-the-password"
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal login payload: %v", err)
	}

	urbitLogin(false, "~zod", payloadBytes)

	message := mustReceiveString(t, BytesChan["~zod"], "login response")
	var response AuthEvent
	if err := json.Unmarshal([]byte(message), &response); err != nil {
		t.Fatalf("failed to unmarshal login response: %v", err)
	}
	if response.Type != "urbit-activity" || response.PayloadType != "login" {
		t.Fatalf("unexpected login response: %+v", response)
	}
}

func TestSendToLeakChannelForwardsAction(t *testing.T) {
	payload := []byte(`{"payload":{"type":"poke"}}`)
	go sendToLeakChannel("~zod", true, "poke", payload)

	action := mustReceiveLeakAction(t, "sendToLeakChannel")
	if action.Patp != "~zod" || action.Type != "poke" || !action.Auth {
		t.Fatalf("unexpected action forwarded to leak channel: %+v", action)
	}
	if !reflect.DeepEqual(action.Content, payload) {
		t.Fatalf("unexpected action content: got %s want %s", action.Content, payload)
	}
}

func TestHandleActionDecodesAndDispatchesPayload(t *testing.T) {
	resetLeakStateForTest(t)
	setLickStatusesForTest(map[string]LickStatus{"~zod": {Auth: false}})
	payload := []byte(`{"payload":{"type":"poke"}}`)

	done := make(chan struct{})
	go func() {
		handleAction("~zod", makeLickPacket(payload))
		close(done)
	}()

	action := mustReceiveLeakAction(t, "handleAction")
	if action.Patp != "~zod" || action.Type != "poke" {
		t.Fatalf("unexpected action from handleAction: %+v", action)
	}
	if !reflect.DeepEqual(action.Content, payload) {
		t.Fatalf("unexpected decoded payload: got %s want %s", action.Content, payload)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handleAction to return")
	}
}

func TestHandleActionReportsMalformedPayloads(t *testing.T) {
	resetLeakStateForTest(t)
	tests := []struct {
		name   string
		packet []byte
	}{
		{
			name:   "short packet",
			packet: []byte{0, 1, 2, 3},
		},
		{
			name:   "atom payload",
			packet: makeAtomPacket(123),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			done := make(chan struct{})
			go func() {
				handleAction("~zod", tc.packet)
				close(done)
			}()

			action := mustReceiveLeakAction(t, tc.name)
			if action.Patp != "~zod" {
				t.Fatalf("expected action for ~zod, got %q", action.Patp)
			}
			if action.Type != string(leakPayloadError) {
				t.Fatalf("expected %q action type, got %q", leakPayloadError, action.Type)
			}
			var payload leakProtocolErrorPayload
			if err := json.Unmarshal(action.Content, &payload); err != nil {
				t.Fatalf("expected JSON error payload, got %v", err)
			}
			if payload.Error == "" || payload.Reason == "" {
				t.Fatalf("expected protocol error payload fields, got %+v", payload)
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for handleAction to return")
			}
		})
	}
}
