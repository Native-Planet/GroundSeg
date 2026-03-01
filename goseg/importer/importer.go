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
		return fmt.Errorf("timed out while combining chunks")
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
	sesh := validated.Session
	patp := validated.Patp

	// Debug session info
	zap.L().Debug(fmt.Sprintf("Upload session information for %v: %+v", validated.SessionID, sesh))

	// Drive handling
	sesh, err = ensureUploadDriveReady(validated.SessionID, sesh)
	if err != nil {
		failUploadRequest(w, http.StatusBadRequest, "Failed to format and use custom drive")
		return
	}

	// Update status
	docker.PublishImportShipTransition(structs.UploadTransition{Type: "patp", Event: patp})
	publishImportStatus("uploading")

	// Handle file upload with streaming to disk
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		failUploadRequest(w, http.StatusBadRequest, fmt.Sprintf("Unable to read uploaded file: %v", err))
		return
	}
	defer file.Close()

	filename := fileHeader.Filename
	index, total, err := parseUploadChunkMetadata(r, filename)
	if err != nil {
		failUploadRequest(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := persistChunkToTemp(file, filename, index); err != nil {
		failUploadRequest(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if all chunks have been received
	if allChunksReceived(filename, total) {
		if err := combineChunksWithTimeout(filename, total, 30*time.Minute); err != nil {
			if strings.Contains(err.Error(), "timed out") {
				failUploadRequest(w, http.StatusRequestTimeout, err.Error())
			} else {
				failUploadRequest(w, http.StatusBadRequest, fmt.Sprintf("Failed to combine chunks: %v", err))
			}
			return
		}
		sendUploadResponse(w, http.StatusOK, "success", "Upload successful")
		ClearUploadSession(validated.SessionID)
		go configureUploadedPier(filename, patp, sesh.Remote, sesh.Fix, sesh.CustomDrive)
		return
	}

	// If we get here, just acknowledge the chunk reception
	sendUploadResponse(w, http.StatusOK, "success", "Chunk received successfully")
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

func configureUploadedPier(filename, patp string, remote, fix bool, dirPath string) {
	defer system.RemoveMultipartFiles("/tmp")
	var customPath string
	if dirPath != "" {
		customPath = dirPath
	}
	volPath := filepath.Join(config.DockerDir, patp, "_data")
	if customPath != "" {
		volPath = filepath.Join(customPath, patp)
	}

	err := runImportPhases(
		filename,
		patp,
		customPath,
		lifecycle.Step{
			Phase: lifecycle.Phase("creating"),
			Run: func() error {
				if err := shipcreator.CreateUrbitConfig(patp, customPath); err != nil {
					return fmt.Errorf("failed to create urbit config: %w", err)
				}
				if err := shipcreator.AppendSysConfigPier(patp); err != nil {
					return fmt.Errorf("failed to update system.json: %w", err)
				}
				zap.L().Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
				if err := docker.DeleteContainer(patp); err != nil {
					zap.L().Error(fmt.Sprintf("%v", err))
				}
				if err := docker.DeleteVolume(patp); err != nil {
					zap.L().Info(fmt.Sprintf("%v (harmless)", err))
				}
				if customPath == "" {
					if err := docker.CreateVolume(patp); err != nil {
						return fmt.Errorf("failed to create volume: %w", err)
					}
					return nil
				}
				if err := os.MkdirAll(customPath, os.ModePerm); err != nil {
					return fmt.Errorf("create custom pier directory error: %w", err)
				}
				return nil
			},
		},
		lifecycle.Step{
			Phase: lifecycle.Phase("extracting"),
			Run: func() error {
				extractionTimeout := time.NewTimer(4 * time.Hour)
				extractionDone := make(chan struct{})
				go func() {
					select {
					case <-extractionTimeout.C:
						docker.PublishImportShipTransition(structs.UploadTransition{Type: "extracted", Value: 100})
						publishImportStatus("checking")
					case <-extractionDone:
						extractionTimeout.Stop()
					}
				}()
				defer close(extractionDone)

				compressedPath := filepath.Join(uploadDir, filename)
				switch checkExtension(filename) {
				case ".zip":
					if err := extractZip(compressedPath, volPath); err != nil {
						return fmt.Errorf("failed to extract %v: %w", filename, err)
					}
				case ".tar.gz", ".tgz":
					if err := extractTarGz(compressedPath, volPath); err != nil {
						return fmt.Errorf("failed to extract %v: %w", filename, err)
					}
				case ".tar":
					if err := extractTar(compressedPath, volPath); err != nil {
						return fmt.Errorf("failed to extract %v: %w", filename, err)
					}
				default:
					return fmt.Errorf("unsupported file type %v", filename)
				}
				zap.L().Debug(fmt.Sprintf("%v extracted to %v", filename, volPath))
				if err := restructureDirectory(patp); err != nil {
					return fmt.Errorf("failed to restructure directory: %w", err)
				}
				return nil
			},
		},
		lifecycle.Step{
			Phase: lifecycle.Phase("booting"),
			Run: func() error {
				zap.L().Info(fmt.Sprintf("Starting extracted pier: %v", patp))
				info, err := docker.StartContainer(patp, "vere")
				if err != nil {
					return fmt.Errorf("failed to start imported ship: %w", err)
				}
				config.UpdateContainerState(patp, info)
				if err := os.Remove(filepath.Join(uploadDir, filename)); err != nil {
					zap.L().Warn(fmt.Sprintf("Failed to remove uploaded archive %s: %v", filename, err))
				}
				return nil
			},
		},
	)
	if err != nil {
		return
	}

	// if startram is registered
	startramSettings := config.StartramSettingsSnapshot()
	if startramSettings.WgRegistered {
		// Register Services
		go registerServices(patp)
	}
	// check for +code
	go waitForShipReady(filename, patp, remote, fix, customPath)
}

func waitForShipReady(filename, patp string, remote, fix bool, customDrive string) {
	zap.L().Info(fmt.Sprintf("Booting ship: %v", patp))
	shipworkflow.WaitForBootCode(patp, 1*time.Second)
	if fix {
		if err := acme.Fix(patp); err != nil {
			errmsg := fmt.Sprintf("Failed to update urbit config for imported ship: %v", err)
			zap.L().Error(errmsg)
		}
	}
	startramSettings := config.StartramSettingsSnapshot()
	if startramSettings.WgRegistered && startramSettings.WgOn && remote {
		publishImportStatus("remote")
		shipworkflow.WaitForRemoteReady(patp, 1*time.Second)
		if err := shipworkflow.SwitchShipToWireguard(patp, true); err != nil {
			errmsg := fmt.Sprintf("%v", err)
			zap.L().Error(errmsg)
			errorCleanup(filename, patp, customDrive, errmsg)
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
	customDir, ok := shipConf.CustomPierLocation.(string)
	if ok {
		zap.L().Info("Custom pier location found!")
		volDir = customDir
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
	_ = filepath.Walk(volDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && filepath.Base(path) == ".urb" && !strings.Contains(path, "__MACOSX") {
			urbLoc = append(urbLoc, filepath.Dir(path))
		}
		return nil
	})
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
			_ = os.MkdirAll(tempDir, 0755)
			items, _ := ioutil.ReadDir(urbLoc[0])
			for _, item := range items {
				if item.Name() != patp {
					os.Rename(filepath.Join(urbLoc[0], item.Name()), filepath.Join(tempDir, item.Name()))
				}
			}
		} else {
			os.Rename(urbLoc[0], tempDir)
		}
		unused := []string{}
		dirs, _ := ioutil.ReadDir(volDir)
		for _, dir := range dirs {
			dirName := dir.Name()
			if dirName != "temp_dir" && dirName != "unused" {
				unused = append(unused, dirName)
			}
		}
		if len(unused) > 0 {
			_ = os.MkdirAll(unusedDir, 0755)
			for _, u := range unused {
				os.Rename(filepath.Join(volDir, u), filepath.Join(unusedDir, u))
			}
		}
		os.Rename(tempDir, pierDir)
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
