package registry

import (
	"testing"

	"groundseg/config"
	"groundseg/structs"
)

func TestGetLatestContainerInfoUsesStaticLlamaConfig(t *testing.T) {
	originalArch := config.Architecture()
	originalChannel := config.GetVersionChannel()
	config.SetArchitecture("amd64")
	config.SetVersionChannel(structs.Channel{})
	t.Cleanup(func() {
		config.SetArchitecture(originalArch)
		config.SetVersionChannel(originalChannel)
	})

	info, err := GetLatestContainerInfo("llama-api")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := info["repo"], "nativeplanet/llama-gpt"; got != want {
		t.Fatalf("repo: got %q want %q", got, want)
	}
	if got, want := info["tag"], "dev"; got != want {
		t.Fatalf("tag: got %q want %q", got, want)
	}
	hash := "ac2dcfac72bc3d8ee51ee255edecc10072ef9c0f958120971c00be5f4944a6fa"
	if got := info["hash"]; got != hash {
		t.Fatalf("hash: got %q want %q", got, hash)
	}
}

func TestGetLatestContainerInfoReadsFromVersionChannel(t *testing.T) {
	originalArch := config.Architecture()
	originalChannel := config.GetVersionChannel()
	config.SetArchitecture("arm64")
	config.SetVersionChannel(structs.Channel{Wireguard: structs.VersionDetails{
		Repo:        "example/wg",
		Tag:         "v1.2.3",
		Arm64Sha256: "deadbeef",
	}})
	t.Cleanup(func() {
		config.SetArchitecture(originalArch)
		config.SetVersionChannel(originalChannel)
	})

	info, err := GetLatestContainerInfo("wireguard")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := info["repo"], "example/wg"; got != want {
		t.Fatalf("repo: got %q want %q", got, want)
	}
	if got, want := info["hash"], "deadbeef"; got != want {
		t.Fatalf("hash: got %q want %q", got, want)
	}
}

func TestGetLatestContainerInfoRejectsUnsupportedType(t *testing.T) {
	if _, err := GetLatestContainerInfo("does-not-exist"); err == nil {
		t.Fatal("expected error for unsupported container type")
	}
}
