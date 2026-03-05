package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"groundseg/shipworkflow"

	"github.com/docker/docker/errdefs"
)

func TestParseUploadChunkMetadata(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/upload?dzchunkindex=2&dztotalchunkcount=5", nil)
	index, total, err := parseUploadChunkMetadata(req, "ship.tar.gz")
	if err != nil {
		t.Fatalf("parseUploadChunkMetadata returned error: %v", err)
	}
	if index != 2 || total != 5 {
		t.Fatalf("unexpected parsed values: index=%d total=%d", index, total)
	}

	badIndexReq := httptest.NewRequest(http.MethodPost, "/upload?dzchunkindex=bad&dztotalchunkcount=5", nil)
	if _, _, err := parseUploadChunkMetadata(badIndexReq, "ship.tar.gz"); err == nil {
		t.Fatal("expected invalid chunk index error")
	}
}

func TestPersistChunkAndCombineChunks(t *testing.T) {
	setUploadDirsForTest(t)

	filename := "ship.tar"
	parts := []string{"abc", "123", "xyz"}
	for index, part := range parts {
		if err := persistChunkToTemp(strings.NewReader(part), filename, index); err != nil {
			t.Fatalf("persistChunkToTemp(%d) returned error: %v", index, err)
		}
	}
	allChunks, err := allChunksReceived(filename, len(parts))
	if err != nil {
		t.Fatalf("allChunksReceived returned error: %v", err)
	}
	if !allChunks {
		t.Fatal("expected allChunksReceived to be true after writing all parts")
	}

	if err := combineChunks(filename, len(parts)); err != nil {
		t.Fatalf("combineChunks returned error: %v", err)
	}
	combinedPath := filepath.Join(uploadDir, filename)
	data, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatalf("read combined file: %v", err)
	}
	if string(data) != "abc123xyz" {
		t.Fatalf("unexpected combined output: %q", string(data))
	}
	for i := range parts {
		partPath := filepath.Join(tempDir, filename+"-part-"+strconv.Itoa(i))
		if _, err := os.Stat(partPath); !os.IsNotExist(err) {
			t.Fatalf("expected chunk file %s to be removed, stat err=%v", partPath, err)
		}
	}
}

func TestPersistChunkToTempPropagatesCloseError(t *testing.T) {
	setUploadDirsForTest(t)
	closeErr := errors.New("close failed")
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		uploadOps := uploadRuntime(t, runtime)
		uploadOps.CloseTempFileFn = func(*os.File) error {
			return closeErr
		}
	})

	err := persistChunkToTemp(strings.NewReader("payload"), "ship.tgz", 0, runtime)
	if err == nil || !errors.Is(err, closeErr) {
		t.Fatalf("expected close error wrapped, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(tempDir, "ship.tgz-part-0")); !os.IsNotExist(statErr) {
		t.Fatalf("expected temp chunk file to be removed when close fails, got %v", statErr)
	}
}

func TestAllChunksReceivedPropagatesFilesystemError(t *testing.T) {
	setUploadDirsForTest(t)
	statErr := errors.New("stat access denied")
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		uploadOps := uploadRuntime(t, runtime)
		uploadOps.StatFn = func(string) (os.FileInfo, error) {
			return nil, statErr
		}
	})

	if _, err := allChunksReceived("ship.tgz", 1, runtime); err == nil || !errors.Is(err, statErr) {
		t.Fatalf("expected stat error to propagate, got %v", err)
	}
}

func TestCombineChunksReturnsErrorWhenChunkMissing(t *testing.T) {
	setUploadDirsForTest(t)

	filename := "ship.tar"
	if err := persistChunkToTemp(strings.NewReader("only-first"), filename, 0); err != nil {
		t.Fatalf("persistChunkToTemp returned error: %v", err)
	}

	err := combineChunks(filename, 2)
	if err == nil {
		t.Fatal("expected combineChunks to fail when a chunk is missing")
	}
	if !strings.Contains(err.Error(), "failed to open chunk file 1") {
		t.Fatalf("unexpected combine error: %v", err)
	}
}

func TestConfigureUploadedPierRunsPostImportWorkflowAndPropagatesErrors(t *testing.T) {
	setUploadDirsForTest(t)
	runtime := defaultImporterRuntime()
	initOps := initRuntime(t, &runtime)
	uploadPath := uploadDir
	tempPath := tempDir
	initOps.StoragePathForFn = func(pathType string) (string, error) {
		switch pathType {
		case "uploads":
			return uploadPath, nil
		case "temp":
			return tempPath, nil
		default:
			return "", fmt.Errorf("unexpected path type: %s", pathType)
		}
	}

	configureErr := errors.New("configure failed")
	postErr := errors.New("post-process failed")

	workflowCalled := false
	postCalled := false
	provisionOps := provisionRuntime(t, &runtime)
	provisionOps.RunImportedPierWorkflowFn = func(_ importedPierContext, _ importerWorkflowRuntime) error {
		workflowCalled = true
		t.Logf("configure workflow called")
		return nil
	}
	provisionOps.RunImportedPierPostImportWorkflowFn = func(_ importedPierContext, _ importerWorkflowRuntime) error {
		postCalled = true
		t.Logf("post-import workflow called")
		return nil
	}
	provisionOps.CleanupMultipartFn = func(string) error { return nil }

	if err := configureUploadedPier(context.Background(), shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, "ship.tgz"),
		Filename:    "ship.tgz",
		Patp:        "~zod",
	}, runtime); err != nil {
		t.Fatalf("configureUploadedPier returned unexpected error: %v", err)
	}
	if !workflowCalled || !postCalled {
		t.Fatalf("expected both imported-pier workflows to be called, got workflow=%v post=%v", workflowCalled, postCalled)
	}

	workflowCalled = false
	postCalled = false
	provisionOps.RunImportedPierWorkflowFn = func(_ importedPierContext, _ importerWorkflowRuntime) error {
		workflowCalled = true
		return configureErr
	}
	provisionOps.RunImportedPierPostImportWorkflowFn = func(_ importedPierContext, _ importerWorkflowRuntime) error {
		postCalled = false
		return nil
	}
	if err := configureUploadedPier(context.Background(), shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, "ship.tgz"),
		Filename:    "ship.tgz",
		Patp:        "~zod",
	}, runtime); err == nil || !errors.Is(err, configureErr) {
		t.Fatalf("expected configure phase failure to be returned, got: %v", err)
	}
	if !workflowCalled || postCalled {
		t.Fatalf("expected only configure workflow to run on configure failure, got workflow=%v post=%v", workflowCalled, postCalled)
	}

	workflowCalled = false
	postCalled = false
	provisionOps.RunImportedPierWorkflowFn = func(_ importedPierContext, _ importerWorkflowRuntime) error {
		workflowCalled = true
		return nil
	}
	provisionOps.RunImportedPierPostImportWorkflowFn = func(_ importedPierContext, _ importerWorkflowRuntime) error {
		postCalled = true
		return postErr
	}
	if err := configureUploadedPier(context.Background(), shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, "ship.tgz"),
		Filename:    "ship.tgz",
		Patp:        "~zod",
	}, runtime); err == nil || !errors.Is(err, postErr) {
		t.Fatalf("expected post-import workflow failure to be returned, got: %v", err)
	}
	if !workflowCalled || !postCalled {
		t.Fatalf("expected both workflows to run, got workflow=%v post=%v", workflowCalled, postCalled)
	}
}

func TestFinalizeUploadOnCompletionDispatchesUploadImportCommand(t *testing.T) {
	setUploadDirsForTest(t)

	runtime := defaultImporterRuntime()
	provisionOps := provisionRuntime(t, &runtime)

	requestCtx := context.WithValue(context.Background(), "test-key", "test-value")
	var observedCmd shipworkflow.UploadImportCommand
	provisionOps.UploadCoordinatorFn = func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
		if ctx != requestCtx {
			t.Fatalf("unexpected dispatch context: %v", ctx)
		}
		observedCmd = cmd
		return nil
	}

	if err := persistChunkToTemp(strings.NewReader("payload"), "ship.tgz", 0); err != nil {
		t.Fatalf("persistChunkToTemp returned error: %v", err)
	}

	completed, err := finalizeUploadOnCompletion(
		uploadChunkProgress{
			Filename:  "ship.tgz",
			Total:     1,
			AllChunks: true,
		},
		validatedUploadRequest{
			Context: requestCtx,
			Patp:    "~zod",
			Session: uploadSession{
				Remote:      true,
				Fix:         true,
				CustomDrive: "/custom/drive",
			},
		},
		runtime,
	)
	if err != nil {
		t.Fatalf("finalizeUploadOnCompletion returned unexpected error: %v", err)
	}
	if !completed {
		t.Fatalf("expected completion to be true after final chunk")
	}
	if observedCmd.ArchivePath != filepath.Join(uploadDir, "ship.tgz") {
		t.Fatalf("unexpected archive path: %q", observedCmd.ArchivePath)
	}
	if observedCmd.Patp != "~zod" {
		t.Fatalf("unexpected patp: %q", observedCmd.Patp)
	}
	if !observedCmd.Remote || !observedCmd.Fix {
		t.Fatalf("expected flags to be propagated, got remote=%v fix=%v", observedCmd.Remote, observedCmd.Fix)
	}
	if observedCmd.CustomDrive != "/custom/drive" {
		t.Fatalf("unexpected custom drive: %q", observedCmd.CustomDrive)
	}
}

func TestFinalizeUploadOnCompletionWrapsDispatchFailureAndImportError(t *testing.T) {
	setUploadDirsForTest(t)

	dispatchErr := errors.New("dispatch failed")
	runtime := defaultImporterRuntime()
	provisionOps := provisionRuntime(t, &runtime)
	provisionOps.UploadCoordinatorFn = func(context.Context, shipworkflow.UploadImportCommand) error {
		return dispatchErr
	}

	if err := persistChunkToTemp(strings.NewReader("payload"), "ship.tgz", 0); err != nil {
		t.Fatalf("persistChunkToTemp returned error: %v", err)
	}
	completed, err := finalizeUploadOnCompletion(
		uploadChunkProgress{
			Filename:  "ship.tgz",
			Total:     1,
			AllChunks: true,
		},
		validatedUploadRequest{
			Patp: "~zod",
		},
		runtime,
	)
	if err == nil {
		t.Fatal("expected finalizeUploadOnCompletion to return an error")
	}
	if !completed {
		t.Fatalf("expected completion to be true after final chunk")
	}
	if !errors.Is(err, errImportPierConfig) {
		t.Fatalf("expected import config sentinel in error chain, got %v", err)
	}
	if !errors.Is(err, dispatchErr) {
		t.Fatalf("expected dispatch error in error chain, got %v", err)
	}
}

func TestFinalizeImportedPierReadinessReturnsAcmeFixErrorWhenFixIsEnabled(t *testing.T) {
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		provisionOps := provisionRuntime(t, runtime)
		provisionOps.ShipworkflowWaitForBootCodeFn = func(string, time.Duration) {}
		provisionOps.ShipworkflowWaitForRemoteReadyFn = func(string, time.Duration) {}
		provisionOps.ShipworkflowSwitchToWireguardFn = func(string, bool) error {
			return nil
		}
	})
	expectedErr := errors.New("acme fix failed")
	provisionOps := provisionRuntime(t, &runtime)
	provisionOps.AcmeFixFn = func(string) error {
		return expectedErr
	}

	err := finalizeImportedPierReadiness(importedPierContext{
		Patp: "~zod",
		Fix:  true,
	}, runtime)
	if err == nil {
		t.Fatal("expected finalizeImportedPierReadiness to return acme fix error")
	}
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error chain to include acme failure, got %v", err)
	}
}

func TestPrepareImportedPierEnvironmentPropagatesRealCleanupErrors(t *testing.T) {
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		provisionOps := provisionRuntime(t, runtime)
		provisionOps.ShipcreatorCreateUrbitConfigFn = func(string, string) error { return nil }
		provisionOps.ShipcreatorAppendSysConfigPierFn = func(string) error { return nil }

		provisionOps.DeleteContainerFn = func(string) error {
			return errors.New("failed to delete container: permission denied")
		}
		provisionOps.DeleteVolumeFn = func(string) error {
			return errors.New("volume cleanup should be ignored when container cleanup fails")
		}
		provisionOps.CreateVolumeFn = func(string) error { return nil }

		provisionOps.MkdirAllFn = func(string, os.FileMode) error { return nil }
	})

	err := prepareImportedPierEnvironment(importedPierContext{
		Patp:        "~zod",
		CustomDrive: "",
	}, runtime)
	if err == nil {
		t.Fatal("expected prepareImportedPierEnvironment to fail for non-ignorable container delete error")
	}
}

func TestPrepareImportedPierEnvironmentIgnoresNotFoundCleanupErrors(t *testing.T) {
	runtime := importerRuntimeWith(func(runtime *importerRuntime) {
		provisionOps := provisionRuntime(t, runtime)
		provisionOps.ShipcreatorCreateUrbitConfigFn = func(string, string) error { return nil }
		provisionOps.ShipcreatorAppendSysConfigPierFn = func(string) error { return nil }

		provisionOps.DeleteContainerFn = func(string) error {
			return errdefs.NotFound(errors.New("container not found"))
		}
		provisionOps.DeleteVolumeFn = func(string) error {
			return errors.New("no such volume")
		}
		provisionOps.CreateVolumeFn = func(string) error { return nil }

		provisionOps.MkdirAllFn = func(string, os.FileMode) error { return nil }
	})

	if err := prepareImportedPierEnvironment(importedPierContext{
		Patp:        "~zod",
		CustomDrive: "",
	}, runtime); err != nil {
		t.Fatalf("expected prepareImportedPierEnvironment to ignore cleanup not found errors: %v", err)
	}
}

func TestIgnorableCleanupDeleteError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "not found",
			err:  errdefs.NotFound(errors.New("container not found")),
			want: true,
		},
		{
			name: "permission denied",
			err:  errors.New("permission denied"),
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isIgnorableCleanupDeleteError(tc.err); got != tc.want {
				t.Fatalf("isIgnorableCleanupDeleteError(%q) = %v, want %v", tc.err.Error(), got, tc.want)
			}
		})
	}
}
