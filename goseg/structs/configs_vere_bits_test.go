package structs

import (
	"encoding/json"
	"testing"
)

func TestEffectiveVereBitsDefaultsTo32(t *testing.T) {
	cases := map[int]int{0: 32, 32: 32, 64: 64, 16: 32, 128: 32}
	for input, want := range cases {
		if got := (UrbitDocker{VereBits: input}).EffectiveVereBits(); got != want {
			t.Fatalf("VereBits %d: got %d, want %d", input, got, want)
		}
	}
}

func TestUrbitDockerUnmarshalsVereBits(t *testing.T) {
	var conf UrbitDocker
	if err := json.Unmarshal([]byte(`{"pier_name":"zod","vere_bits":64}`), &conf); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if conf.VereBits != 64 {
		t.Fatalf("expected vere_bits 64, got %d", conf.VereBits)
	}
	var legacy UrbitDocker
	if err := json.Unmarshal([]byte(`{"pier_name":"zod"}`), &legacy); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if legacy.EffectiveVereBits() != 32 {
		t.Fatalf("legacy config should default to 32-bit, got %d", legacy.EffectiveVereBits())
	}
}
