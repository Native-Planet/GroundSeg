package importer

import (
	"context"
	"errors"
	"fmt"
	"groundseg/auth"
	"groundseg/click/acme"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/dockerclient"
	"groundseg/driveresolver"
	"groundseg/lifecycle"
	"groundseg/orchestration"
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

	"github.com/docker/docker/api/types/volume"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
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
	uploadSessions = make(map[string]uploadSession) // todo: add checkbox data to struct
	uploadDir      string
	tempDir        string
	uploadMu       sync.Mutex
	uploadTTL      = 20 * time.Minute
	uploadKeyRegex = regexp.MustCompile(`^[a-f0-9]{32}$`)

	errChunkCombineTimeout = errors.New("import chunk combining timed out")
)

func Initialize() error {
	uploadDir = config.GetStoragePath("uploads")
	tempDir = config.GetStoragePath("temp")
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
	if !auth.TokenIdAuthed(auth.ClientManager, token.ID) {
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
	// check if token is valid
	if existingSession.Token != token {
		return errors.New("token mismatch")
	}
	// Modify checkboxes
	existingSession.Remote = remote
	existingSession.Fix = fix
	existingSession.SelectedDrive = sel
	existingSession.CustomDrive = customDrive
	existingSession.NeedsFormatting = driveResolution.NeedsFormatting
	existingSession.ExpiresAt = time.Now().Add(uploadTTL)

	uploadSessions[endpoint] = existingSession
	zap.L().Warn(fmt.Sprintf("current upload configuration: %+v", cmd))
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
	_, exists := uploadSessions[session]
	// todo: check token for user agent info
	return exists
}

func Reset() error {
	publishImportStatus("")
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "patp", Event: ""})
	publishImportError("")
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "extracted", Value: 0})
	return nil
}

type validatedUploadRequest struct {
	SessionID string
	Patp      string
	Session   uploadSession
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

func sendUploadResponse(w http.ResponseWriter, code int, status, message string) {
	w.WriteHeader(code)
	w.Write([]byte(fmt.Sprintf(`{"status": "%s", "message": "%s"}`, status, message)))
}

func publishImportStatus(event string) {
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "status", Event: event})
}

func publishImportError(message string) {
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "error", Event: message})
}

func failUploadRequest(w http.ResponseWriter, code int, message string) {
	zap.L().Error(fmt.Sprintf("Upload error: %v", message))
	publishImportError(message)
	publishImportStatus("aborted")
	sendUploadResponse(w, code, "failure", message)
}

func validateUploadRequest(r *http.Request) (validatedUploadRequest, error) {
	var validated validatedUploadRequest
	vars := mux.Vars(r)
	session := vars["uploadSession"]
	patp := vars["patp"]

	uploadMu.Lock()
	sesh, validSession := uploadSessions[session]
	uploadMu.Unlock()

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
	if tokenID != sesh.Token.ID || tokenHash != sesh.Token.Token {
		return validated, fmt.Errorf("upload token does not match upload session")
	}
	verifiedToken := map[string]string{
		"id":    tokenID,
		"token": tokenHash,
	}
	if _, err := auth.ValidateAndAuthorizeRequestToken(verifiedToken["id"], verifiedToken["token"], r); err != nil {
		return validated, fmt.Errorf("upload token validation failed: %w", err)
	}

	validated.SessionID = session
	validated.Patp = patp
	validated.Session = sesh
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
	zap.L().Debug(fmt.Sprintf("%v chunkIndex: %v, totalChunks: %v", filename, chunkIndex, totalChunks))

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
	defer tempFile.Close()

	buffer := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(tempFile, file, buffer); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to save chunk: %w", err)
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

func runImportPhases(filename, patp, customDrive string, steps ...lifecycle.Step) error {
	return orchestration.RunPhases(
		steps,
		func(phase lifecycle.Phase) {
			if phase == "" {
				return
			}
			publishImportStatus(string(phase))
		},
		func(_ lifecycle.Phase, err error) {
			errorCleanup(filename, patp, customDrive, err.Error())
		},
		nil,
		nil,
	)
}

func HTTPUploadHandler(w http.ResponseWriter, r *http.Request) {
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

	validated, err := validateUploadRequest(r)
	if err != nil {
		failUploadRequest(w, http.StatusUnauthorized, err.Error())
		return
	}

	chunk, err := readUploadChunk(r)
	if err != nil {
		failUploadRequest(w, http.StatusBadRequest, err.Error())
		return
	}
	defer chunk.File.Close()

	pipelineResult := runImportUploadPipeline(validated, chunk)
	if pipelineResult.err != nil {
		failUploadRequest(w, pipelineResult.statusCode, pipelineResult.err.Error())
		return
	}
	if pipelineResult.completed {
		sendUploadResponse(w, http.StatusOK, "success", "Upload successful")
		ClearUploadSession(validated.SessionID)
		return
	}

	// If we get here, just acknowledge the chunk reception
	sendUploadResponse(w, http.StatusOK, "success", "Chunk received successfully")
}

type uploadPipelineResult struct {
	completed  bool
	statusCode int
	err        error
}

func runImportUploadPipeline(validated validatedUploadRequest, chunk uploadChunkPayload) uploadPipelineResult {
	session, err := prepareUploadSessionForChunk(validated.SessionID, validated.Session)
	if err != nil {
		return uploadPipelineResult{
			completed:  false,
			statusCode: http.StatusBadRequest,
			err:        fmt.Errorf("Failed to format and use custom drive: %w", err),
		}
	}
	validated.Session = session

	zap.L().Debug(fmt.Sprintf("Upload session information for %v: %+v", validated.SessionID, session))
	publishUploadStatus(validated.Patp)

	if err := persistUploadedChunk(chunk.Filename, chunk.Index, chunk.File); err != nil {
		return uploadPipelineResult{
			completed:  false,
			statusCode: http.StatusBadRequest,
			err:        err,
		}
	}

	chunkProgress := uploadChunkProgress{
		Filename:  chunk.Filename,
		Total:     chunk.Total,
		AllChunks: allChunksReceived(chunk.Filename, chunk.Total),
	}
	outcome, err := finalizeUploadOnCompletion(chunkProgress, validated.Patp, validated.Session)
	if err == nil {
		return uploadPipelineResult{completed: outcome}
	}
	if errors.Is(err, errChunkCombineTimeout) {
		return uploadPipelineResult{
			completed:  true,
			statusCode: http.StatusRequestTimeout,
			err:        err,
		}
	}
	if strings.HasPrefix(err.Error(), "Failed to configure imported pier") {
		return uploadPipelineResult{
			completed:  true,
			statusCode: http.StatusInternalServerError,
			err:        err,
		}
	}
	return uploadPipelineResult{
		completed:  true,
		statusCode: http.StatusBadRequest,
		err:        err,
	}
}

func publishUploadStatus(patp string) {
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "patp", Event: patp})
	publishImportStatus("uploading")
}

func prepareUploadSessionForChunk(sessionID string, session uploadSession) (uploadSession, error) {
	return ensureUploadDriveReady(sessionID, session)
}

func readUploadChunk(r *http.Request) (uploadChunkPayload, error) {
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		return uploadChunkPayload{}, fmt.Errorf("Unable to read uploaded file: %v", err)
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

func finalizeUploadOnCompletion(progress uploadChunkProgress, patp string, session uploadSession) (bool, error) {
	if !progress.AllChunks {
		return false, nil
	}
	if err := combineChunksWithTimeout(progress.Filename, progress.Total, 30*time.Minute); err != nil {
		if errors.Is(err, errChunkCombineTimeout) {
			return true, errChunkCombineTimeout
		}
		return true, fmt.Errorf("Failed to combine chunks: %v", err)
	}
	if err := configureUploadedPier(progress.Filename, patp, session.Remote, session.Fix, session.CustomDrive); err != nil {
		return true, fmt.Errorf("Failed to configure imported pier: %v", err)
	}
	return true, nil
}

func allChunksReceived(filename string, total int) bool {
	for i := 0; i < total; i++ {
		if _, err := os.Stat(filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))); os.IsNotExist(err) {
			return false
		}
	}
	return true
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
			zap.L().Warn(fmt.Sprintf("Failed to remove chunk file %s: %v", partFilePath, err))
			// Continue despite error removing the file
		}
	}

	return nil
}

func configureUploadedPier(filename, patp string, remote, fix bool, dirPath string) error {
	defer system.RemoveMultipartFiles("/tmp")
	ctx := newImportedPierContext(filename, patp, remote, fix, dirPath)
	if err := runImportedPierWorkflow(ctx); err != nil {
		return err
	}
	go startImportedPierPostImportPipeline(ctx)
	return nil
}

type importedPierContext struct {
	Filename    string
	Patp        string
	Remote      bool
	Fix         bool
	CustomDrive string
	VolumePath  string
}

func newImportedPierContext(filename, patp string, remote, fix bool, customDrive string) importedPierContext {
	volumePath := filepath.Join(config.DockerDir, patp, "_data")
	if customDrive != "" {
		volumePath = filepath.Join(customDrive, patp)
	}
	return importedPierContext{
		Filename:    filename,
		Patp:        patp,
		Remote:      remote,
		Fix:         fix,
		CustomDrive: customDrive,
		VolumePath:  volumePath,
	}
}

func runImportedPierWorkflow(ctx importedPierContext) error {
	return runImportPhases(
		ctx.Filename,
		ctx.Patp,
		ctx.CustomDrive,
		lifecycle.Step{
			Phase: lifecycle.Phase("creating"),
			Run: func() error {
				return prepareImportedPierEnvironment(ctx)
			},
		},
		lifecycle.Step{
			Phase: lifecycle.Phase("extracting"),
			Run: func() error {
				return extractUploadedPier(ctx)
			},
		},
		lifecycle.Step{
			Phase: lifecycle.Phase("booting"),
			Run: func() error {
				return bootImportedPier(ctx)
			},
		},
	)
}

func startImportedPierPostImportPipeline(ctx importedPierContext) {
	startImportedPierRegistration(ctx.Patp)
	startImportedPierReadiness(ctx)
}

func startImportedPierRegistration(patp string) {
	startramSettings := config.StartramSettingsSnapshot()
	if startramSettings.WgRegistered {
		go registerServices(patp)
	}
}

func startImportedPierReadiness(ctx importedPierContext) {
	go finalizeImportedPierReadiness(ctx)
}

func prepareImportedPierEnvironment(ctx importedPierContext) error {
	if err := shipcreator.CreateUrbitConfig(ctx.Patp, ctx.CustomDrive); err != nil {
		return fmt.Errorf("failed to create urbit config: %w", err)
	}
	if err := shipcreator.AppendSysConfigPier(ctx.Patp); err != nil {
		return fmt.Errorf("failed to update system.json: %w", err)
	}
	zap.L().Info(fmt.Sprintf("Preparing environment for pier: %v", ctx.Patp))
	if err := docker.DeleteContainer(ctx.Patp); err != nil {
		zap.L().Error(fmt.Sprintf("%v", err))
	}
	if err := docker.DeleteVolume(ctx.Patp); err != nil {
		zap.L().Info(fmt.Sprintf("%v (harmless)", err))
	}
	if ctx.CustomDrive == "" {
		if err := docker.CreateVolume(ctx.Patp); err != nil {
			return fmt.Errorf("failed to create volume: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(ctx.CustomDrive, os.ModePerm); err != nil {
		return fmt.Errorf("create custom pier directory error: %w", err)
	}
	return nil
}

func extractUploadedPier(ctx importedPierContext) error {
	extractionDone := make(chan struct{})
	defer close(extractionDone)
	go monitorExtractionProgress(extractionDone)

	compressedPath := filepath.Join(uploadDir, ctx.Filename)
	if err := extractUploadedArchive(compressedPath, ctx.VolumePath, ctx.Filename); err != nil {
		return fmt.Errorf("failed to extract %v: %w", ctx.Filename, err)
	}

	zap.L().Debug(fmt.Sprintf("%v extracted to %v", ctx.Filename, ctx.VolumePath))
	if err := restructureDirectory(ctx.Patp); err != nil {
		return fmt.Errorf("failed to restructure directory: %w", err)
	}
	return nil
}

func monitorExtractionProgress(done <-chan struct{}) {
	extractionTimeout := time.NewTimer(4 * time.Hour)
	select {
	case <-extractionTimeout.C:
		docker.PublishImportShipTransition(structs.UploadTransition{Type: "extracted", Value: 100})
		publishImportStatus("checking")
	case <-done:
		extractionTimeout.Stop()
	}
}

func bootImportedPier(ctx importedPierContext) error {
	zap.L().Info(fmt.Sprintf("Starting extracted pier: %v", ctx.Patp))
	info, err := docker.StartContainer(ctx.Patp, "vere")
	if err != nil {
		return fmt.Errorf("failed to start imported ship: %w", err)
	}
	config.UpdateContainerState(ctx.Patp, info)
	if err := os.Remove(filepath.Join(uploadDir, ctx.Filename)); err != nil {
		zap.L().Warn(fmt.Sprintf("Failed to remove uploaded archive %s: %v", ctx.Filename, err))
	}
	return nil
}

func finalizeImportedPierReadiness(ctx importedPierContext) {
	zap.L().Info(fmt.Sprintf("Booting ship: %v", ctx.Patp))
	shipworkflow.WaitForBootCode(ctx.Patp, 1*time.Second)
	if ctx.Fix {
		if err := acme.Fix(ctx.Patp); err != nil {
			errmsg := fmt.Sprintf("Failed to update urbit config for imported ship: %v", err)
			zap.L().Error(errmsg)
		}
	}
	startramSettings := config.StartramSettingsSnapshot()
	if startramSettings.WgRegistered && startramSettings.WgOn && ctx.Remote {
		publishImportStatus("remote")
		shipworkflow.WaitForRemoteReady(ctx.Patp, 1*time.Second)
		if err := shipworkflow.SwitchShipToWireguard(ctx.Patp, true); err != nil {
			errmsg := fmt.Sprintf("%v", err)
			zap.L().Error(errmsg)
			errorCleanup(ctx.Filename, ctx.Patp, ctx.CustomDrive, errmsg)
			return
		}
	}
	publishImportStatus("completed")
}

func errorCleanup(filename, patp, customDrive, errmsg string) {
	// notify that we are cleaning up
	zap.L().Info(fmt.Sprintf("Pier import process failed: %s: %s", patp, errmsg))
	zap.L().Info(fmt.Sprintf("Running cleanup routine"))

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
		zap.L().Error(fmt.Sprintf("Import rollback encountered errors: %v", err))
	}
	publishImportError(errmsg)
	publishImportStatus("aborted")
	return
}

func restructureDirectory(patp string) error {
	zap.L().Info("Checking pier directory")
	// get docker volume path for patp
	volDir := ""
	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	volumes, err := cli.VolumeList(context.Background(), volume.ListOptions{})
	if err != nil {
		return err
	}
	// if no customDir is set, check volume
	shipConf := config.UrbitConf(patp)
	if shipConf.CustomPierLocation != "" {
		zap.L().Info("Custom pier location found!")
		volDir = shipConf.CustomPierLocation
	} else {
		for _, vol := range volumes.Volumes {
			if vol.Name == patp {
				volDir = vol.Mountpoint
				break
			}
		}
	}
	if volDir == "" {
		return fmt.Errorf("No docker volume for %s!", patp)
	}
	zap.L().Info(fmt.Sprintf("%v pier path: %v", patp, volDir))
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
	zap.L().Debug(fmt.Sprintf(".urb subdirectory in %v", urbLoc[0]))
	pierDir := filepath.Join(volDir, patp)
	tempDir := filepath.Join(volDir, "temp_dir")
	unusedDir := filepath.Join(volDir, "unused")
	// move it into the right place
	if filepath.Join(pierDir, ".urb") != filepath.Join(urbLoc[0], ".urb") {
		zap.L().Info(".urb location incorrect! Restructuring directory structure")
		zap.L().Debug(fmt.Sprintf(".urb found in %v", urbLoc[0]))
		zap.L().Debug(fmt.Sprintf("Moving to %v", tempDir))
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
		zap.L().Info(fmt.Sprintf("%v restructuring done", patp))
	} else {
		zap.L().Debug("No restructuring needed")
	}
	return nil
}

func registerServices(patp string) {
	if err := shipworkflow.RegisterShipServices(patp); err != nil {
		zap.L().Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
