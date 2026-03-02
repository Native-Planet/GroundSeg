package importer

import (
	"context"
	"errors"
	"fmt"
	"groundseg/auth"
	"groundseg/click/acme"
	"groundseg/config"
	"groundseg/docker/events"
	dockerOrchestration "groundseg/docker/orchestration"
	"groundseg/driveresolver"
	"groundseg/lifecycle"
	"groundseg/logger"
	workflowOrchestration "groundseg/orchestration"
	"groundseg/shipcleanup"
	"groundseg/shipcreator"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"groundseg/system"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type uploadSession struct {
	Token           structs.WsTokenStruct
	Remote          bool
	Fix             bool
	SelectedDrive   string
	CustomDrive     string
	NeedsFormatting bool
	ExpiresAt       time.Time
}

type OpenUploadEndpointCmd struct {
	Endpoint      string
	Token         structs.WsTokenStruct
	Remote        bool
	Fix           bool
	SelectedDrive string
}

var (
	uploadSessions  = make(map[string]uploadSession) // todo: add checkbox data to struct
	uploadDir       string
	tempDir         string
	uploadMu        sync.RWMutex
	uploadTTL       = 20 * time.Minute
	uploadKeyRegex  = regexp.MustCompile(`^[a-f0-9]{32}$`)
	closeTempFileFn = func(file *os.File) error {
		return file.Close()
	}
	statFn                           = os.Stat
	shipworkflowWaitForBootCodeFn    = shipworkflow.WaitForBootCode
	shipworkflowWaitForRemoteReadyFn = shipworkflow.WaitForRemoteReady
	shipworkflowSwitchToWireguardFn  = shipworkflow.SwitchShipToWireguard
	acmeFixFn                        = acme.Fix
	shipcreatorCreateUrbitConfigFn   = shipcreator.CreateUrbitConfig
	shipcreatorAppendSysConfigPierFn = shipcreator.AppendSysConfigPier
	validateUploadSessionTokenFn     = auth.ValidateUploadSessionToken
	dockerDeleteContainerFn          = dockerOrchestration.DeleteContainer
	dockerDeleteVolumeFn             = dockerOrchestration.DeleteVolume
	dockerCreateVolumeFn             = dockerOrchestration.CreateVolume
	mkdirAllFn                       = os.MkdirAll
)

type importerRuntime struct {
	runImportedPierWorkflowFn           func(importedPierContext) error
	runImportedPierPostImportWorkflowFn func(importedPierContext) error
	resolveImportedPierVolumePathFn     func(context.Context, string) (string, error)
	cleanupMultipartFn                  func(string) error
	storagePathForFn                    func(string) (string, error)
	// uploadCoordinator should remain an interface seam because it represents a meaningful
	// cross-package handoff with its own lifecycle and test seam.
	uploadCoordinator shipworkflow.UploadImportCoordinator
}

func defaultImporterRuntime() importerRuntime {
	return importerRuntime{
		runImportedPierWorkflowFn:           runImportedPierWorkflowDefault,
		runImportedPierPostImportWorkflowFn: runImportedPierPostImportWorkflowDefault,
		resolveImportedPierVolumePathFn: func(_ context.Context, patp string) (string, error) {
			return resolveImportedPierVolumePath(patp)
		},
		cleanupMultipartFn: system.RemoveMultipartFiles,
		storagePathForFn:   config.GetStoragePath,
		uploadCoordinator: shipworkflow.UploadImportCoordinatorFunc(func(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
			return shipworkflow.DispatchUploadImportWithCoordinator(shipworkflow.UploadImportCoordinatorFunc(configureUploadedPier), ctx, cmd)
		}),
	}
}

func (runtime importerRuntime) ensureInitializedForInit() error {
	if runtime.storagePathForFn == nil {
		return errors.New("importer runtime storage path callback is not configured")
	}
	return nil
}

func (runtime importerRuntime) ensureForWorkflow() error {
	if runtime.resolveImportedPierVolumePathFn == nil {
		return errors.New("importer runtime volume resolution callback is not configured")
	}
	if runtime.cleanupMultipartFn == nil {
		return errors.New("importer runtime cleanup callback is not configured")
	}
	if runtime.runImportedPierWorkflowFn == nil {
		return errors.New("importer runtime workflow callback is not configured")
	}
	if runtime.runImportedPierPostImportWorkflowFn == nil {
		return errors.New("importer runtime post-import workflow callback is not configured")
	}
	if runtime.uploadCoordinator == nil {
		return errUploadImportCoordinatorUnconfigured
	}
	return nil
}

func resolveImportedPierVolumePath(patp string) (string, error) {
	if patp == "" {
		return "", fmt.Errorf("ship name is required for volume resolution")
	}
	return filepath.Join(dockerOrchestration.VolumeDir, patp, "_data"), nil
}

var (
	errUploadImportCoordinatorUnconfigured = errors.New("upload import coordinator is not configured")
	errChunkCombineTimeout                 = errors.New("import chunk combining timed out")
	errImportPierConfig                    = errors.New("failed to configure imported pier")
)

func Initialize() error {
 	return initializeWithRuntime(defaultImporterRuntime())
}

func initializeWithRuntime(runtime importerRuntime) error {
	if err := runtime.ensureInitializedForInit(); err != nil {
		return err
	}
	var err error
	uploadDir, err = runtime.storagePathForFn("uploads")
	if err != nil {
		return fmt.Errorf("initialize upload directory: %w", err)
	}
	tempDir, err = runtime.storagePathForFn("temp")
	if err != nil {
		return fmt.Errorf("initialize temp directory: %w", err)
	}
	swapSettings := config.SwapSettingsSnapshot()
	if !strings.HasPrefix(swapSettings.SwapFile, "/opt") {
		var tempPath string
		lastSlashIndex := strings.LastIndex(swapSettings.SwapFile, "/")
		if lastSlashIndex != -1 {
			tempPath = swapSettings.SwapFile[:lastSlashIndex]
			tempDir = filepath.Join(tempPath, "temp")
			uploadDir = filepath.Join(tempPath, "uploads")
		}
	}
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return err
	}
	return nil
}

func OpenUploadEndpoint(cmd OpenUploadEndpointCmd) error {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	// grab from payload
	endpoint := cmd.Endpoint
	token := cmd.Token
	remote := cmd.Remote
	fix := cmd.Fix
	if !uploadKeyRegex.MatchString(endpoint) {
		return errors.New("invalid upload session key format")
	}
	if token.ID == "" || token.Token == "" {
		return errors.New("missing upload auth token")
	}
	var expectedToken structs.WsTokenStruct
	if existingSession, exists := uploadSessions[endpoint]; exists {
		expectedToken = existingSession.Token
	}

	authz := validateUploadSessionTokenFn(expectedToken, token, nil, auth.UploadAuthPolicy())
	if !authz.IsAuthorized() {
		if authz.Status == auth.UploadValidationStatusTokenContract {
			return errors.New("token mismatch")
		}
		return errors.New("upload token id is not authorized")
	}

	driveResolution, err := driveresolver.Resolve(cmd.SelectedDrive)
	if err != nil {
		return fmt.Errorf("resolve selected drive: %w", err)
	}
	sel := driveResolution.SelectedDrive
	customDrive := driveResolution.Mountpoint

	// check if endpoint exists
	existingSession, exists := uploadSessions[endpoint]
	if !exists {
		//build new configuration
		sesh := uploadSession{
			Token:           token,
			Remote:          remote,
			Fix:             fix,
			SelectedDrive:   sel,
			CustomDrive:     customDrive,
			NeedsFormatting: driveResolution.NeedsFormatting,
			ExpiresAt:       time.Now().Add(uploadTTL),
		}
		uploadSessions[endpoint] = sesh
		return nil
	}
	// Modify checkboxes
	existingSession.Remote = remote
	existingSession.Fix = fix
	existingSession.SelectedDrive = sel
	existingSession.CustomDrive = customDrive
	existingSession.NeedsFormatting = driveResolution.NeedsFormatting
	existingSession.ExpiresAt = time.Now().Add(uploadTTL)

	uploadSessions[endpoint] = existingSession
	logger.Warnf("current upload configuration: %+v", cmd)
	return nil
}

func SetUploadSession(uploadPayload structs.WsUploadPayload) error {
	return OpenUploadEndpoint(OpenUploadEndpointCmd{
		Endpoint:      uploadPayload.Payload.Endpoint,
		Token:         uploadPayload.Token,
		Remote:        uploadPayload.Payload.Remote,
		Fix:           uploadPayload.Payload.Fix,
		SelectedDrive: uploadPayload.Payload.SelectedDrive,
	})
}

func ClearUploadSession(session string) {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	delete(uploadSessions, session)
}

func VerifySession(session string) bool {
	_, exists := loadUploadSession(session)
	return exists
}

func loadUploadSession(session string) (uploadSession, bool) {
	uploadMu.RLock()
	defer uploadMu.RUnlock()
	sesh, exists := uploadSessions[session]
	return sesh, exists
}

func storeUploadSession(session string, sesh uploadSession) {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	uploadSessions[session] = sesh
}

func Reset() error {
	publishImportStatus("")
	events.PublishImportShipTransition(structs.UploadTransition{Type: "patp", Event: ""})
	publishImportError("")
	events.PublishImportShipTransition(structs.UploadTransition{Type: "extracted", Value: 0})
	return nil
}

type validatedUploadRequest struct {
	SessionID string
	Patp      string
	Session   uploadSession
	Context   context.Context
}

type uploadChunkPayload struct {
	Filename string
	Index    int
	Total    int
	File     io.ReadCloser
}

type uploadChunkProgress struct {
	Filename  string
	Total     int
	AllChunks bool
}

func setUploadResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, X-Requested-With")
}

func sendUploadResponse(w http.ResponseWriter, code int, status, message string) error {
	w.WriteHeader(code)
	if _, err := w.Write([]byte(fmt.Sprintf(`{"status": "%s", "message": "%s"}`, status, message))); err != nil {
		return fmt.Errorf("failed to write upload response: %w", err)
	}
	return nil
}

func publishImportStatus(event string) {
	events.PublishImportShipTransition(structs.UploadTransition{Type: "status", Event: event})
}

func publishImportError(message string) {
	events.PublishImportShipTransition(structs.UploadTransition{Type: "error", Event: message})
}

func failUploadRequest(w http.ResponseWriter, code int, message string) error {
	logger.Errorf("Upload error: %v", message)
	publishImportError(message)
	publishImportStatus("aborted")
	return sendUploadResponse(w, code, "failure", message)
}

func validateUploadRequest(r *http.Request, uploadSession, patp string) (validatedUploadRequest, error) {
	var validated validatedUploadRequest
	session := uploadSession
	shipName := patp

	sesh, validSession := loadUploadSession(session)

	if !validSession {
		return validated, fmt.Errorf("invalid upload session")
	}
	if time.Now().After(sesh.ExpiresAt) {
		ClearUploadSession(session)
		return validated, fmt.Errorf("upload session expired")
	}

	tokenID := strings.TrimSpace(r.Header.Get("X-Upload-Token-Id"))
	tokenHash := strings.TrimSpace(r.Header.Get("X-Upload-Token"))
	if tokenID == "" || tokenHash == "" {
		return validated, fmt.Errorf("missing upload authorization headers")
	}
	authz := auth.ValidateUploadSessionToken(sesh.Token, structs.WsTokenStruct{
		ID:    tokenID,
		Token: tokenHash,
	}, r, auth.UploadAuthPolicy())
	if !authz.IsAuthorized() {
		if authz.Status == auth.UploadValidationStatusTokenContract {
			return validated, fmt.Errorf("upload token does not match upload session")
		}
		return validated, fmt.Errorf("upload token validation failed: %w", authz.AuthorizationErr)
	}

	if authz.IsRotated() {
		// keep session token in sync with any rotated, authorized token
		sesh.Token.Token = authz.AuthorizedToken
		storeUploadSession(session, sesh)
	}

	validated.SessionID = session
	validated.Patp = shipName
	validated.Session = sesh
	validated.Context = r.Context()
	return validated, nil
}

func ensureUploadDriveReady(sessionID string, sesh uploadSession) (uploadSession, error) {
	if sesh.SelectedDrive == "system-drive" || !sesh.NeedsFormatting {
		return sesh, nil
	}
	resolution, err := driveresolver.EnsureReady(driveresolver.Resolution{
		SelectedDrive:   sesh.SelectedDrive,
		Mountpoint:      sesh.CustomDrive,
		NeedsFormatting: sesh.NeedsFormatting,
	})
	if err != nil {
		return sesh, err
	}

	uploadMu.Lock()
	defer uploadMu.Unlock()
	sesh.NeedsFormatting = resolution.NeedsFormatting
	sesh.CustomDrive = resolution.Mountpoint
	uploadSessions[sessionID] = sesh
	return sesh, nil
}

func parseUploadChunkMetadata(r *http.Request, filename string) (int, int, error) {
	chunkIndex := r.FormValue("dzchunkindex")
	totalChunks := r.FormValue("dztotalchunkcount")
	logger.Debug(fmt.Sprintf("%v chunkIndex: %v, totalChunks: %v", filename, chunkIndex, totalChunks))

	index, err := strconv.Atoi(chunkIndex)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid chunk index")
	}
	total, err := strconv.Atoi(totalChunks)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid total chunk count")
	}
	return index, total, nil
}

func persistChunkToTemp(file io.Reader, filename string, index int) error {
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, index))
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	buffer := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(tempFile, file, buffer); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to save chunk: %w", err)
	}
	if err := closeTempFileFn(tempFile); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	return nil
}

func combineChunksWithTimeout(filename string, total int, timeout time.Duration) error {
	combineCtx, combineCancel := context.WithTimeout(context.Background(), timeout)
	defer combineCancel()
	resultCh := make(chan error, 1)
	go func() {
		resultCh <- combineChunks(filename, total)
	}()
	select {
	case err := <-resultCh:
		return err
	case <-combineCtx.Done():
		if errors.Is(combineCtx.Err(), context.DeadlineExceeded) {
			return errChunkCombineTimeout
		}
		return combineCtx.Err()
	}
}

func runImportPhases(filename, patp, customDrive string, phases workflowOrchestration.WorkflowPhases) error {
	return workflowOrchestration.RunStructuredWorkflow(
		phases,
		workflowOrchestration.WorkflowCallbacks{
			Emit: func(phase lifecycle.Phase) {
				if phase == "" {
					return
				}
				publishImportStatus(string(phase))
			},
			OnError: func(err error) {
				errorCleanup(filename, patp, customDrive, err)
			},
		},
	)
}

func runImportPhasesWithSteps(filename, patp, customDrive string, steps ...lifecycle.Step) error {
	return runImportPhases(
		filename,
		patp,
		customDrive,
		workflowOrchestration.WorkflowPhases{
			Execute: steps,
		},
	)
}

func HTTPUploadHandler(w http.ResponseWriter, r *http.Request, uploadSession, patp string) {
	setUploadResponseHeaders(w)

	// Handle pre-flight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Set timeouts on the request context
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	r = r.WithContext(ctx)

	validated, err := validateUploadRequest(r, uploadSession, patp)
	if err != nil {
		if err := failUploadRequest(w, http.StatusUnauthorized, err.Error()); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}

	chunk, err := readUploadChunk(r)
	if err != nil {
		if err := failUploadRequest(w, http.StatusBadRequest, err.Error()); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}
	defer chunk.File.Close()

	pipelineResult := runImportUploadPipeline(validated, chunk)
	if pipelineResult.err != nil {
		if err := failUploadRequest(w, pipelineResult.statusCode, pipelineResult.err.Error()); err != nil {
			logger.Errorf("failed to write upload failure response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		}
		return
	}
	if pipelineResult.completed {
		if err := sendUploadResponse(w, http.StatusOK, "success", "Upload successful"); err != nil {
			logger.Errorf("failed to write upload completion response: %v", err)
			http.Error(w, "failed to write upload response", http.StatusInternalServerError)
			return
		}
		ClearUploadSession(validated.SessionID)
		return
	}

	// If we get here, just acknowledge the chunk reception
	if err := sendUploadResponse(w, http.StatusOK, "success", "Chunk received successfully"); err != nil {
		logger.Errorf("failed to write upload progress response: %v", err)
		http.Error(w, "failed to write upload response", http.StatusInternalServerError)
		return
	}
}

type uploadPipelineResult struct {
	completed  bool
	statusCode int
	err        error
}

type importUploadPipeline struct {
	validated       validatedUploadRequest
	chunk           uploadChunkPayload
	session         uploadSession
	completionState importCompletionState
}

type importCompletionState struct {
	Error      error
	IsComplete bool
}

func runImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload) uploadPipelineResult {
	pipeline := newImportUploadPipeline(validated, chunk)
	return pipeline.run()
}

func newImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload) *importUploadPipeline {
	return &importUploadPipeline{
		validated: validated,
		chunk:     chunk,
	}
}

func (pipeline *importUploadPipeline) run() uploadPipelineResult {
	logger.Debug(fmt.Sprintf("Upload session information for %v: %+v", pipeline.validated.SessionID, pipeline.session))
	publishUploadStatus(pipeline.validated.Patp)
	if err := runImportPhases(
		pipeline.chunk.Filename,
		pipeline.validated.Patp,
		pipeline.validated.Session.CustomDrive,
		workflowOrchestration.WorkflowPhases{
			Prepare: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("preparing"),
					Run:   pipeline.prepareSession,
				},
			},
			Execute: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("persisting"),
					Run:   pipeline.persistChunk,
				},
			},
			Post: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("finalizing"),
					Run:   pipeline.setUploadFinalizationState,
				},
			},
		},
	); err != nil {
		if pipeline.completionState.Error != nil {
			switch {
			case errors.Is(pipeline.completionState.Error, errChunkCombineTimeout):
				return uploadPipelineResult{
					completed:  pipeline.completionState.IsComplete,
					statusCode: http.StatusRequestTimeout,
					err:        pipeline.completionState.Error,
				}
			case errors.Is(pipeline.completionState.Error, errImportPierConfig):
				return uploadPipelineResult{
					completed:  pipeline.completionState.IsComplete,
					statusCode: http.StatusInternalServerError,
					err:        pipeline.completionState.Error,
				}
			}
			return uploadPipelineResult{
				completed:  pipeline.completionState.IsComplete,
				statusCode: http.StatusBadRequest,
				err:        pipeline.completionState.Error,
			}
		}
		return uploadPipelineResult{
			completed:  false,
			statusCode: http.StatusBadRequest,
			err:        err,
		}
	}
	return uploadPipelineResult{
		completed: pipeline.completionState.IsComplete,
	}
}

func (pipeline *importUploadPipeline) prepareSession() error {
	session, err := prepareUploadSessionForChunk(pipeline.validated.SessionID, pipeline.validated.Session)
	if err != nil {
		return err
	}
	pipeline.session = session
	pipeline.validated.Session = session
	return nil
}

func (pipeline *importUploadPipeline) persistChunk() error {
	return persistUploadedChunk(pipeline.chunk.Filename, pipeline.chunk.Index, pipeline.chunk.File)
}

func (pipeline *importUploadPipeline) setUploadFinalizationState() error {
	allChunks, err := allChunksReceived(pipeline.chunk.Filename, pipeline.chunk.Total)
	if err != nil {
		return err
	}
	completed, err := finalizeUploadOnCompletion(
		uploadChunkProgress{
			Filename:  pipeline.chunk.Filename,
			Total:     pipeline.chunk.Total,
			AllChunks: allChunks,
		},
		pipeline.validated,
	)
	pipeline.completionState = importCompletionState{
		Error:      err,
		IsComplete: completed,
	}
	if err != nil {
		return err
	}
	return nil
}

func publishUploadStatus(patp string) {
	events.PublishImportShipTransition(structs.UploadTransition{Type: "patp", Event: patp})
	publishImportStatus("uploading")
}

func prepareUploadSessionForChunk(sessionID string, session uploadSession) (uploadSession, error) {
	return ensureUploadDriveReady(sessionID, session)
}

func readUploadChunk(r *http.Request) (uploadChunkPayload, error) {
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return uploadChunkPayload{}, fmt.Errorf("Unable to read uploaded file: %w", err)
	}

	index, total, err := parseUploadChunkMetadata(r, fileHeader.Filename)
	if err != nil {
		file.Close()
		return uploadChunkPayload{}, err
	}
	return uploadChunkPayload{
		Filename: fileHeader.Filename,
		Index:    index,
		Total:    total,
		File:     file,
	}, nil
}

func persistUploadedChunk(filename string, index int, data io.Reader) error {
	return persistChunkToTemp(data, filename, index)
}

func finalizeUploadOnCompletion(progress uploadChunkProgress, validated validatedUploadRequest) (bool, error) {
	return finalizeUploadOnCompletionWithRuntime(progress, validated, defaultImporterRuntime())
}

func finalizeUploadOnCompletionWithRuntime(
	progress uploadChunkProgress,
	validated validatedUploadRequest,
	runtime importerRuntime,
) (bool, error) {
	if !progress.AllChunks {
		return false, nil
	}
	if err := combineChunksWithTimeout(progress.Filename, progress.Total, 30*time.Minute); err != nil {
		if errors.Is(err, errChunkCombineTimeout) {
			return true, errChunkCombineTimeout
		}
		return true, fmt.Errorf("Failed to combine chunks: %w", err)
	}
	dispatchCmd := toUploadImportCommand(validated, progress)
	if err := runtime.uploadCoordinator.HandleUploadImport(validated.Context, dispatchCmd); err != nil {
		return true, fmt.Errorf("failed to finalize imported pier config: %w", errors.Join(errImportPierConfig, err))
	}
	return true, nil
}

func toUploadImportCommand(validated validatedUploadRequest, progress uploadChunkProgress) shipworkflow.UploadImportCommand {
	return shipworkflow.UploadImportCommand{
		ArchivePath: filepath.Join(uploadDir, progress.Filename),
		Filename:    progress.Filename,
		Patp:        validated.Patp,
		Remote:      validated.Session.Remote,
		Fix:         validated.Session.Fix,
		CustomDrive: validated.Session.CustomDrive,
	}
}

func allChunksReceived(filename string, total int) (bool, error) {
	for i := 0; i < total; i++ {
		partPath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))
		exists, err := fileExists(partPath)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

func fileExists(path string) (bool, error) {
	if _, err := statFn(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat path %s: %w", path, err)
	}
	return true, nil
}

func combineChunks(filename string, total int) error {
	destFilePath := filepath.Join(uploadDir, filename)
	destFile, err := os.Create(destFilePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Use a buffer to improve performance
	buffer := make([]byte, 32*1024) // 32KB buffer

	for i := 0; i < total; i++ {
		partFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))

		// Open each chunk file
		partFile, err := os.Open(partFilePath)
		if err != nil {
			return fmt.Errorf("failed to open chunk file %d: %w", i, err)
		}

		// Stream the chunk file to the destination using the buffer
		_, err = io.CopyBuffer(destFile, partFile, buffer)
		partFile.Close() // Close immediately after copying

		if err != nil {
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}

		// Remove the chunk file after successful copy
		if err := os.Remove(partFilePath); err != nil {
			logger.Warn(fmt.Sprintf("Failed to remove chunk file %s: %v", partFilePath, err))
			// Continue despite error removing the file
		}
	}

	return nil
}

func configureUploadedPier(ctx context.Context, cmd shipworkflow.UploadImportCommand) error {
	return configureUploadedPierWithRuntime(ctx, cmd, defaultImporterRuntime())
}

func configureUploadedPierWithRuntime(
	ctx context.Context,
	cmd shipworkflow.UploadImportCommand,
	runtime importerRuntime,
) error {
	if err := runtime.ensureForWorkflow(); err != nil {
		return err
	}
	defer runtime.cleanupMultipartFn(tempDir)
	pierCtx, err := newImportedPierContext(ctx, runtime, cmd)
	if err != nil {
		return err
	}
	if err := runtime.runImportedPierWorkflowFn(pierCtx); err != nil {
		return err
	}
	if err := runtime.runImportedPierPostImportWorkflowFn(pierCtx); err != nil {
		return err
	}
	return nil
}

type importedPierContext struct {
	Context     context.Context
	ArchivePath string
	Filename    string
	Patp        string
	Remote      bool
	Fix         bool
	CustomDrive string
	VolumePath  string
}

func newImportedPierContext(ctx context.Context, runtime importerRuntime, cmd shipworkflow.UploadImportCommand) (importedPierContext, error) {
	requestCtx := ctx
	if requestCtx == nil {
		requestCtx = context.Background()
	}
	if cmd.Patp == "" {
		return importedPierContext{}, fmt.Errorf("upload command missing ship name")
	}
	if _, err := runtime.storagePathForFn("uploads"); err != nil {
		return importedPierContext{}, err
	}
	if _, err := runtime.storagePathForFn("temp"); err != nil {
		return importedPierContext{}, err
	}
	volumePath, err := runtime.resolveImportedPierVolumePathFn(requestCtx, cmd.Patp)
	if err != nil {
		return importedPierContext{}, fmt.Errorf("failed to resolve import volume path for %s: %w", cmd.Patp, err)
	}
	if cmd.CustomDrive != "" {
		volumePath = filepath.Join(cmd.CustomDrive, cmd.Patp)
	}
	archivePath := cmd.ArchivePath
	if archivePath == "" {
		if cmd.Filename == "" {
			return importedPierContext{}, fmt.Errorf("archive path and filename missing for %s", cmd.Patp)
		}
		archivePath = filepath.Join(uploadDir, cmd.Filename)
	}
	return importedPierContext{
		Context:     requestCtx,
		ArchivePath: archivePath,
		Filename:    cmd.Filename,
		Patp:        cmd.Patp,
		Remote:      cmd.Remote,
		Fix:         cmd.Fix,
		CustomDrive: cmd.CustomDrive,
		VolumePath:  volumePath,
	}, nil
}

func runImportedPierWorkflow(ctx importedPierContext) error {
	return runImportedPierWorkflowWithRuntime(ctx, defaultImporterRuntime())
}

func runImportedPierWorkflowWithRuntime(ctx importedPierContext, runtime importerRuntime) error {
	if err := runtime.ensureForWorkflow(); err != nil {
		return err
	}
	return runtime.runImportedPierWorkflowFn(ctx)
}

func runImportedPierWorkflowDefault(ctx importedPierContext) error {
	return runImportPhases(
		ctx.Filename,
		ctx.Patp,
		ctx.CustomDrive,
		workflowOrchestration.WorkflowPhases{
			Prepare: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("creating"),
					Run: func() error {
						return prepareImportedPierEnvironment(ctx)
					},
				},
			},
			Execute: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("extracting"),
					Run: func() error {
						return extractUploadedPier(ctx)
					},
				},
				{
					Phase: lifecycle.Phase("booting"),
					Run: func() error {
						return bootImportedPier(ctx)
					},
				},
			},
		},
	)
}

func runImportedPierPostImportWorkflowWithRuntime(ctx importedPierContext, runtime importerRuntime) error {
	if err := runtime.ensureForWorkflow(); err != nil {
		return err
	}
	if err := runtime.runImportedPierPostImportWorkflowFn(ctx); err != nil {
		logger.Error(fmt.Sprintf("Imported pier post-processing failed for %s: %v", ctx.Patp, err))
		return err
	}
	return nil
}

func runImportedPierPostImportWorkflow(ctx importedPierContext) error {
	return runImportedPierPostImportWorkflowWithRuntime(ctx, defaultImporterRuntime())
}

func runImportedPierPostImportWorkflowDefault(ctx importedPierContext) error {
	return runImportedPierPostProcessWorkflowWithRuntime(ctx, defaultImporterRuntime())
}

func runImportedPierPostProcessWorkflowWithRuntime(ctx importedPierContext, _ importerRuntime) error {
	return runImportPhases(
		ctx.Filename,
		ctx.Patp,
		ctx.CustomDrive,
		workflowOrchestration.WorkflowPhases{
			Post: []lifecycle.Step{
				{
					Phase: lifecycle.Phase("registering"),
					Run: func() error {
						return startImportedPierRegistration(ctx.Patp)
					},
				},
				{
					Phase: lifecycle.Phase("finalizing"),
					Run: func() error {
						return finalizeImportedPierReadiness(ctx)
					},
				},
			},
		},
	)
}

func runImportedPierPostProcessWorkflow(ctx importedPierContext) error {
	return runImportedPierPostProcessWorkflowWithRuntime(ctx, defaultImporterRuntime())
}

func startImportedPierRegistration(patp string) error {
	startramSettings := config.StartramSettingsSnapshot()
	if startramSettings.WgRegistered {
		return registerServices(patp)
	}
	return nil
}

func finalizeImportedPierReadiness(ctx importedPierContext) error {
	logger.Info(fmt.Sprintf("Booting ship: %v", ctx.Patp))
	shipworkflowWaitForBootCodeFn(ctx.Patp, 1*time.Second)
	if ctx.Fix {
		if err := acmeFixFn(ctx.Patp); err != nil {
			wrappedErr := fmt.Errorf("failed to apply ACME fix for imported ship %s: %w", ctx.Patp, err)
			logger.Error(wrappedErr.Error())
			return wrappedErr
		}
	}
	startramSettings := config.StartramSettingsSnapshot()
	if startramSettings.WgRegistered && startramSettings.WgOn && ctx.Remote {
		publishImportStatus("remote")
		shipworkflowWaitForRemoteReadyFn(ctx.Patp, 1*time.Second)
		if err := shipworkflowSwitchToWireguardFn(ctx.Patp, true); err != nil {
			wrappedErr := fmt.Errorf("failed to switch imported ship %s to Wireguard: %w", ctx.Patp, err)
			logger.Error(wrappedErr.Error())
			errorCleanup(ctx.Filename, ctx.Patp, ctx.CustomDrive, wrappedErr)
			return wrappedErr
		}
	}
	publishImportStatus("completed")
	return nil
}

func prepareImportedPierEnvironment(ctx importedPierContext) error {
	if err := shipcreatorCreateUrbitConfigFn(ctx.Patp, ctx.CustomDrive); err != nil {
		return fmt.Errorf("failed to create urbit config: %w", err)
	}
	if err := shipcreatorAppendSysConfigPierFn(ctx.Patp); err != nil {
		return fmt.Errorf("failed to update system.json: %w", err)
	}
	logger.Info(fmt.Sprintf("Preparing environment for pier: %v", ctx.Patp))
	if err := dockerDeleteContainerFn(ctx.Patp); err != nil {
		if !isIgnorableCleanupDeleteError(err) {
			return fmt.Errorf("failed to clean up pre-existing container %s: %w", ctx.Patp, err)
		}
		logger.Info(fmt.Sprintf("ignoring pre-existing container cleanup error for %s: %v", ctx.Patp, err))
	}
	if err := dockerDeleteVolumeFn(ctx.Patp); err != nil {
		if !isIgnorableCleanupDeleteError(err) {
			return fmt.Errorf("failed to clean up pre-existing volume %s: %w", ctx.Patp, err)
		}
		logger.Info(fmt.Sprintf("ignoring pre-existing volume cleanup error for %s: %v", ctx.Patp, err))
	}
	if ctx.CustomDrive == "" {
		if err := dockerCreateVolumeFn(ctx.Patp); err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
		return nil
	}
	if err := mkdirAllFn(ctx.CustomDrive, os.ModePerm); err != nil {
		return fmt.Errorf("create custom pier directory error: %w", err)
	}
	return nil
}

func isIgnorableCleanupDeleteError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "no such")
}

func extractUploadedPier(ctx importedPierContext) error {
	extractionDone := make(chan struct{})
	defer close(extractionDone)
	go monitorExtractionProgress(extractionDone)

	compressedPath := ctx.ArchivePath
	if err := extractUploadedArchive(compressedPath, ctx.VolumePath, ctx.Filename); err != nil {
		return fmt.Errorf("failed to extract %v: %w", ctx.Filename, err)
	}

	logger.Debug(fmt.Sprintf("%v extracted to %v", ctx.Filename, ctx.VolumePath))
	if err := restructureDirectory(ctx); err != nil {
		return fmt.Errorf("failed to restructure directory: %w", err)
	}
	return nil
}

func monitorExtractionProgress(done <-chan struct{}) {
	extractionTimeout := time.NewTimer(4 * time.Hour)
	select {
	case <-extractionTimeout.C:
		events.PublishImportShipTransition(structs.UploadTransition{Type: "extracted", Value: 100})
		publishImportStatus("checking")
	case <-done:
		extractionTimeout.Stop()
	}
}

func bootImportedPier(ctx importedPierContext) error {
	logger.Info(fmt.Sprintf("Starting extracted pier: %v", ctx.Patp))
	info, err := dockerOrchestration.StartContainer(ctx.Patp, "vere")
	if err != nil {
		return fmt.Errorf("failed to start imported ship: %w", err)
	}
	config.UpdateContainerState(ctx.Patp, info)
	if err := os.Remove(ctx.ArchivePath); err != nil {
		logger.Warn(fmt.Sprintf("Failed to remove uploaded archive %s: %v", ctx.Filename, err))
	}
	return nil
}

func errorCleanup(filename, patp, customDrive string, err error) {
	// notify that we are cleaning up
	logger.Info(fmt.Sprintf("Pier import process failed: %s: %v", patp, err))
	logger.Info(fmt.Sprintf("Running cleanup routine"))

	customPierPath := ""
	if customDrive != "" {
		customPierPath = filepath.Join(customDrive, patp)
	}
	if err := shipcleanup.RollbackProvisioning(patp, shipcleanup.RollbackOptions{
		UploadArchivePath:    filepath.Join(uploadDir, filename),
		CustomPierPath:       customPierPath,
		RemoveContainer:      true,
		RemoveContainerState: true,
	}); err != nil {
		logger.Error(fmt.Sprintf("Import rollback encountered errors: %v", err))
	}
	publishImportError(err.Error())
	publishImportStatus("aborted")
	return
}

func restructureDirectory(ctx importedPierContext) error {
	patp := ctx.Patp
	volDir := ctx.VolumePath
	if volDir == "" {
		return fmt.Errorf("No docker volume for %s!", patp)
	}

	logger.Info("Checking pier directory")
	logger.Info(fmt.Sprintf("%v pier path: %v", patp, volDir))
	// find .urb
	var urbLoc []string
	if err := filepath.Walk(volDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && filepath.Base(path) == ".urb" && !strings.Contains(path, "__MACOSX") {
			urbLoc = append(urbLoc, filepath.Dir(path))
		}
		return nil
	}); err != nil {
		return err
	}
	// there can only be one
	if len(urbLoc) > 1 {
		return fmt.Errorf("%v ships detected in pier directory", len(urbLoc))
	}
	if len(urbLoc) < 1 {
		return fmt.Errorf("No ship found in pier directory")
	}
	logger.Debug(fmt.Sprintf(".urb subdirectory in %v", urbLoc[0]))
	pierDir := filepath.Join(volDir, patp)
	tempDir := filepath.Join(volDir, "temp_dir")
	unusedDir := filepath.Join(volDir, "unused")
	// move it into the right place
	if filepath.Join(pierDir, ".urb") != filepath.Join(urbLoc[0], ".urb") {
		logger.Info(".urb location incorrect! Restructuring directory structure")
		logger.Debug(fmt.Sprintf(".urb found in %v", urbLoc[0]))
		logger.Debug(fmt.Sprintf("Moving to %v", tempDir))
		if volDir == urbLoc[0] { // .urb in root
			if err := os.MkdirAll(tempDir, 0755); err != nil {
				return fmt.Errorf("create temp import directory %s: %w", tempDir, err)
			}
			items, err := ioutil.ReadDir(urbLoc[0])
			if err != nil {
				return fmt.Errorf("read pier directory %s: %w", urbLoc[0], err)
			}
			for _, item := range items {
				if item.Name() != patp {
					if err := os.Rename(filepath.Join(urbLoc[0], item.Name()), filepath.Join(tempDir, item.Name())); err != nil {
						return fmt.Errorf("move %s to temp directory: %w", item.Name(), err)
					}
				}
			}
		} else {
			if err := os.Rename(urbLoc[0], tempDir); err != nil {
				return fmt.Errorf("move %s to temp directory: %w", urbLoc[0], err)
			}
		}
		unused := []string{}
		dirs, err := ioutil.ReadDir(volDir)
		if err != nil {
			return fmt.Errorf("read import directory %s: %w", volDir, err)
		}
		for _, dir := range dirs {
			dirName := dir.Name()
			if dirName != "temp_dir" && dirName != "unused" {
				unused = append(unused, dirName)
			}
		}
		if len(unused) > 0 {
			if err := os.MkdirAll(unusedDir, 0755); err != nil {
				return fmt.Errorf("create unused directory %s: %w", unusedDir, err)
			}
			for _, u := range unused {
				if err := os.Rename(filepath.Join(volDir, u), filepath.Join(unusedDir, u)); err != nil {
					return fmt.Errorf("archive directory %s: %w", u, err)
				}
			}
		}
		if err := os.Rename(tempDir, pierDir); err != nil {
			return fmt.Errorf("finalize pier directory %s: %w", pierDir, err)
		}
		logger.Info(fmt.Sprintf("%v restructuring done", patp))
	} else {
		logger.Debug("No restructuring needed")
	}
	return nil
}

func registerServices(patp string) error {
	if err := shipworkflow.RegisterShipServices(patp); err != nil {
		logger.Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
		return err
	}
	return nil
}
