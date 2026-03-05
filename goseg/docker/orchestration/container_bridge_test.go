package orchestration

import "testing"

func TestDockerContainerBridgeBuildsLlamaRuntime(t *testing.T) {
	rt := llamaRuntimeFromDocker(newDockerRuntime())
	if rt.StartContainerFn == nil {
		t.Fatal("expected llama runtime start container callback")
	}
	if rt.StopContainerByNameFn == nil {
		t.Fatal("expected llama runtime stop container callback")
	}
	if rt.PenpaiSettingsSnapshotFn == nil {
		t.Fatal("expected llama runtime penpai settings snapshot callback")
	}
	if rt.ShipSettingsSnapshotFn == nil {
		t.Fatal("expected llama runtime ship settings snapshot callback")
	}
	if rt.DockerDirFn == nil {
		t.Fatal("expected llama runtime docker dir callback")
	}
	if rt.VolumeDirFn == nil {
		t.Fatal("expected llama runtime volume dir callback")
	}
}

func TestDockerContainerBridgeBuildsNetdataRuntime(t *testing.T) {
	rt := netdataRuntimeFromDocker(newDockerRuntime())
	if rt.StartContainerFn == nil {
		t.Fatal("expected netdata runtime start container callback")
	}
	if rt.UpdateContainerState == nil {
		t.Fatal("expected netdata runtime update container state callback")
	}
	if rt.GetLatestContainerInfoFn == nil {
		t.Fatal("expected netdata runtime image info callback")
	}
	if rt.CopyFileToVolumeFn == nil {
		t.Fatal("expected netdata runtime copy-to-volume callback")
	}
	if rt.CreateDefaultFn == nil {
		t.Fatal("expected netdata runtime default config callback")
	}
}

func TestDockerContainerBridgeBuildsMinioRuntime(t *testing.T) {
	rt := minioRuntimeFromDocker(newDockerRuntime())
	if rt.StartContainerFn == nil {
		t.Fatal("expected minio runtime start container callback")
	}
	if rt.UpdateContainerStateFn == nil {
		t.Fatal("expected minio runtime update container state callback")
	}
	if rt.GetLatestContainerInfoFn == nil {
		t.Fatal("expected minio runtime image info callback")
	}
	if rt.CopyFileToVolumeFn == nil {
		t.Fatal("expected minio runtime copy-to-volume callback")
	}
	if rt.LoadUrbitConfigFn == nil {
		t.Fatal("expected minio runtime load urbit config callback")
	}
	if rt.SetMinIOPasswordFn == nil || rt.GetMinIOPasswordFn == nil {
		t.Fatal("expected minio runtime password callbacks")
	}
}
