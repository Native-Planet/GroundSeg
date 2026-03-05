package shipworkflow

import (
	"strings"
	"testing"

	"groundseg/structs"
)

func TestRunStartramUploadBackupWithRuntimeFailsWhenBackupKeyMissing(t *testing.T) {
	t.Chdir(t.TempDir())
	runtime := defaultStartramRuntime()
	runtime.PublishEventFn = func(structs.Event) {}

	err := runStartramUploadBackupWithRuntime(runtime, "~zod")
	if err == nil {
		t.Fatal("expected missing backup.key to fail upload backup flow")
	}
	if !strings.Contains(err.Error(), "failed to read private key file") {
		t.Fatalf("unexpected error for missing backup key: %v", err)
	}
}
