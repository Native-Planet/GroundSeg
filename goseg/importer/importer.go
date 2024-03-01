package importer

import (
	"context"
	"errors"
	"fmt"
	"groundseg/click"
	"groundseg/config"
	"groundseg/docker"
	"groundseg/logger"
	"groundseg/shipcreator"
	"groundseg/startram"
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
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

type uploadSession struct {
	Token           structs.WsTokenStruct
	Remote          bool
	Fix             bool
	SelectedDrive   string
	CustomDrive     string
	NeedsFormatting bool
}

var (
	uploadSessions = make(map[string]uploadSession) // todo: add checkbox data to struct
	uploadDir      = filepath.Join(config.BasePath, "uploads")
	tempDir        = filepath.Join(config.BasePath, "temp")
	uploadMu       sync.Mutex
)

func init() {
	conf := config.Conf()
	if !strings.HasPrefix(conf.SwapFile, "/opt") {
		var tempPath string
		lastSlashIndex := strings.LastIndex(conf.SwapFile, "/")
		if lastSlashIndex != -1 {
			tempPath = conf.SwapFile[:lastSlashIndex]
			tempDir = filepath.Join(tempPath, "temp")
			uploadDir = filepath.Join(tempPath, "uploads")
		}
	}
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(tempDir, 0755)
}

func SetUploadSession(uploadPayload structs.WsUploadPayload) error {
	uploadMu.Lock()
	defer uploadMu.Unlock()
	// grab from payload
	endpoint := uploadPayload.Payload.Endpoint
	token := uploadPayload.Token
	remote := uploadPayload.Payload.Remote
	fix := uploadPayload.Payload.Fix

	// check which drive the user wants us to keep the pier in
	sel := uploadPayload.Payload.SelectedDrive

	// custom drive, leave empty if not on other drive
	var customDrive string

	// we don't need to do anything for system-drive, only if its something else
	if sel != "system-drive" {
		blockDevices, err := system.ListHardDisks()
		if err != nil {
			return fmt.Errorf("Failed to retrieve block devices: %v", err)
		}

		// we're looking for the drive the user specified
		for _, dev := range blockDevices.BlockDevices {
			// we have the drive
			if dev.Name == sel {
				for _, m := range dev.Mountpoints {
					// check if mountpoint matches groundseg's expectations
					matched, err := regexp.MatchString(`^/groundseg-\d+$`, m)
					if err != nil {
						logger.Logger.Error(fmt.Sprintf("Regex error for mountpoint: %v", m))
						continue
					}
					// yes
					if matched {
						customDrive = m
						// breaks inner loop, we've got our directory
						break
					}
				}
				// breaks outer loop after we finish getting the info we need
				break
			}
		}
	}

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
			NeedsFormatting: sel != "system-drive" && customDrive == "",
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
	existingSession.NeedsFormatting = sel != "system-drive" && customDrive == ""

	uploadSessions[endpoint] = existingSession
	logger.Logger.Warn(fmt.Sprintf("current upload configuration: %+v", uploadPayload.Payload))
	return nil
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
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: ""}
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "patp", Event: ""}
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "error", Event: ""}
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "extracted", Value: 0}
	return nil
}

func HTTPUploadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Cache-Control, X-Requested-With")

	// Handle pre-flight OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	session := vars["uploadSession"]
	validSession := VerifySession(session)

	patp := vars["patp"]
	handleSend := func(requestCode int, status, message string) {
		if status == "failure" {
			logger.Logger.Error(fmt.Sprintf("Upload error: %v", message))
			docker.ImportShipTransBus <- structs.UploadTransition{Type: "error", Event: message}
			docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "aborted"}
		}
		w.WriteHeader(requestCode)
		w.Write([]byte(fmt.Sprintf(`{"status": "%s", "message": "%s"}`, status, message)))
	}

	if validSession {
		// get session info
		sesh, exists := uploadSessions[session]
		// debug
		if exists {
			logger.Logger.Debug(fmt.Sprintf("Upload session information for %v: %+v", session, sesh))
		} else {
			logger.Logger.Debug(fmt.Sprintf("Upload session information for %v doesn't exist!", session))
		}
		// end debug

		// we first make sure we know which drive the user wants the pier written on
		//we skip this step if system-drive was selected
		if sesh.SelectedDrive != "system-drive" {
			// check if drive needs formatting
			if sesh.NeedsFormatting {
				if mountpoint, err := system.CreateGroundSegFilesystem(sesh.SelectedDrive); err != nil {
					handleSend(http.StatusBadRequest, "failure", fmt.Sprintf("failed to format and use custom drive"))
					return
				} else {
					sesh.NeedsFormatting = false
					sesh.CustomDrive = mountpoint
					uploadSessions[session] = sesh
				}
			}
		}

		// we then handle upload
		docker.ImportShipTransBus <- structs.UploadTransition{Type: "patp", Event: patp}
		docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "uploading"}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			handleSend(http.StatusBadRequest, "failure", fmt.Sprintf("Unable to read uploaded file: %v", err))
			return
		}
		defer file.Close()

		filename := fileHeader.Filename
		chunkIndex := r.FormValue("dzchunkindex")
		totalChunks := r.FormValue("dztotalchunkcount")
		logger.Logger.Debug(fmt.Sprintf("%v chunkIndex: %v, totalChunks: %v", filename, chunkIndex, totalChunks))
		index, err := strconv.Atoi(chunkIndex)
		if err != nil {
			handleSend(http.StatusBadRequest, "failure", "Invalid chunk index")
			return
		}
		total, err := strconv.Atoi(totalChunks)
		if err != nil {
			handleSend(http.StatusBadRequest, "failure", "Invalid total chunk count")
			return
		}
		tempFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, index))
		tempFile, err := os.Create(tempFilePath)
		if err != nil {
			handleSend(http.StatusBadRequest, "failure", "Failed to create temp file")
			return
		}
		defer tempFile.Close()
		io.Copy(tempFile, file)
		if allChunksReceived(filename, total) {
			if err := combineChunks(filename, total); err != nil {
				handleSend(http.StatusBadRequest, "failure", "Failed to combine chunks")
				return
			}
			handleSend(http.StatusOK, "success", "Upload successful")
			uploadMu.Lock()
			uploadMu.Unlock()
			ClearUploadSession(session)
			go configureUploadedPier(filename, patp, sesh.Remote, sesh.Fix, sesh.CustomDrive)
			return
		}
	} else {
		logger.Logger.Error(fmt.Sprintf("Invalid upload session request %v", session))
		handleSend(http.StatusUnauthorized, "failure", "Invalid upload session")
		return
	}
	handleSend(http.StatusOK, "success", "Chunk received successfully")
	return
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

	destFile, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return err
	}
	defer destFile.Close()
	for i := 0; i < total; i++ {
		partFilePath := filepath.Join(tempDir, fmt.Sprintf("%s-part-%d", filename, i))
		partFile, err := os.Open(partFilePath)
		if err != nil {
			return err
		}
		io.Copy(destFile, partFile)
		partFile.Close()
		os.Remove(partFilePath)
	}
	return nil
}

func configureUploadedPier(filename, patp string, remote, fix bool, dirPath string) {
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "creating"}
	// create pier config
	var customPath string
	if dirPath != "" {
		customPath = dirPath
	}
	err := shipcreator.CreateUrbitConfig(patp, customPath)
	if err != nil {
		errmsg := fmt.Sprintf("Failed to create urbit config: %v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(filename, patp, errmsg)
		return
	}
	// update system.json
	err = shipcreator.AppendSysConfigPier(patp)
	if err != nil {
		errmsg := fmt.Sprintf("Failed to update system.json: %v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(filename, patp, errmsg)
		return
	}
	// Prepare environment for pier
	logger.Logger.Info(fmt.Sprintf("Preparing environment for pier: %v", patp))
	// delete container if exists
	err = docker.DeleteContainer(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// delete volume if exists
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v (harmless)", err)
		logger.Logger.Info(errmsg)
	}
	if customPath == "" { // no custom path provided
		// create new docker volume
		err = docker.CreateVolume(patp)
		if err != nil {
			errmsg := fmt.Sprintf("failed to create volume: %v", err)
			logger.Logger.Error(errmsg)
			errorCleanup(filename, patp, errmsg)
			return
		}
	} else { // create custom directory for upload
		if err := os.MkdirAll(customPath, os.ModePerm); err != nil {
			errmsg := fmt.Sprintf("create custom pier directory error: %v", err)
			errorCleanup(filename, patp, errmsg)
			return
		}
	}
	// extract file to volume directory
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "extracting"}
	// set default path
	volPath := filepath.Join(config.DockerDir, patp, "_data")
	// modify if custom path
	if customPath != "" {
		volPath = filepath.Join(customPath, patp)
	}
	compressedPath := filepath.Join(uploadDir, filename)
	switch checkExtension(filename) {
	case ".zip":
		err := extractZip(compressedPath, volPath)
		if err != nil {
			errmsg := fmt.Sprintf("Failed to extract %v: %v", filename, err)
			errorCleanup(filename, patp, errmsg)
			return
		}
	case ".tar.gz", ".tgz":
		err := extractTarGz(compressedPath, volPath)
		if err != nil {
			errmsg := fmt.Sprintf("Failed to extract %v: %v", filename, err)
			errorCleanup(filename, patp, errmsg)
			return
		}
	case ".tar":
		err := extractTar(compressedPath, volPath)
		if err != nil {
			errmsg := fmt.Sprintf("Failed to extract %v: %v", filename, err)
			errorCleanup(filename, patp, errmsg)
			return
		}
	default:
		errmsg := fmt.Sprintf("Unsupported file type %v", filename)
		errorCleanup(filename, patp, errmsg)
		return
	}
	logger.Logger.Debug(fmt.Sprintf("%v extracted to %v", filename, volPath))
	// run restructure
	if err := restructureDirectory(patp); err != nil {
		errorCleanup(filename, patp, fmt.Sprintf("Failed to restructure directory: %v", err))
		return
	}
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "booting"}
	// start container
	logger.Logger.Info(fmt.Sprintf("Starting extracted pier: %v", patp))
	info, err := docker.StartContainer(patp, "vere")
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
		errorCleanup(filename, patp, errmsg)
		return
	}
	config.UpdateContainerState(patp, info)
	os.Remove(filepath.Join(uploadDir, filename))

	// debug, force error
	//errmsg := "Self induced error, for debugging purposes"
	//errorCleanup(filename, patp, errmsg)
	//return

	// if startram is registered
	conf := config.Conf()
	if conf.WgRegistered {
		// Register Services
		go registerServices(patp)
	}
	// check for +code
	go waitForShipReady(filename, patp, remote, fix)
}

func waitForShipReady(filename, patp string, remote, fix bool) {
	logger.Logger.Info(fmt.Sprintf("Booting ship: %v", patp))
	lusCodeTicker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-lusCodeTicker.C:
			code, err := click.GetLusCode(patp)
			if err != nil {
				continue
			}
			if len(code) == 27 {
				break
			} else {
				continue
			}
		}
		if fix {
			if err := click.FixAcme(patp); err != nil {
				errmsg := fmt.Sprintf("Failed to update urbit config for imported ship: %v", err)
				logger.Logger.Error(errmsg)
			}
		}
		conf := config.Conf()
		if conf.WgRegistered && conf.WgOn && remote {
			importShipToggleRemote(patp)
			shipConf := config.UrbitConf(patp)
			shipConf.Network = "wireguard"
			update := make(map[string]structs.UrbitDocker)
			update[patp] = shipConf
			if err := config.UpdateUrbitConfig(update); err != nil {
				errmsg := fmt.Sprintf("Failed to update urbit config for imported ship: %v", err)
				errorCleanup(filename, patp, errmsg)
				return
			}
			logger.Logger.Debug(fmt.Sprintf("Deleting container %s for switching networks", patp))
			statuses, err := docker.GetShipStatus([]string{patp})
			if err != nil {
				logger.Logger.Error(fmt.Sprintf("Failed to get statuses for %s when rebuilding container: %v", patp, err))
			}
			status, exists := statuses[patp]
			if !exists {
				logger.Logger.Error(fmt.Sprintf("%s status doesn't exist: %v"))
			}
			isRunning := strings.Contains(status, "Up")
			if isRunning {
				if err := click.BarExit(patp); err != nil {
					logger.Logger.Error(fmt.Sprintf("Failed to stop %s with |exit for rebuilding container: %v", patp, err))
				}
			}
			if err := docker.DeleteContainer(patp); err != nil {
				errmsg := fmt.Sprintf("Failed to delete local container for imported ship: %v", err)
				logger.Logger.Error(errmsg)
			}
			docker.StartContainer("minio_"+patp, "minio")
			logger.Logger.Debug(fmt.Sprintf("Starting container %s after switching networks", patp))
			info, err := docker.StartContainer(patp, "vere")
			if err != nil {
				errmsg := fmt.Sprintf("%v", err)
				logger.Logger.Error(errmsg)
				errorCleanup(filename, patp, errmsg)
				return
			}
			config.UpdateContainerState(patp, info)
		}
		docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "completed"}
		return
	}
}

func importShipToggleRemote(patp string) {
	docker.ImportShipTransBus <- structs.UploadTransition{Type: "status", Event: "remote"}
	remoteTicker := time.NewTicker(1 * time.Second)
	// break if all subdomains with this patp has status of "ok"
	for {
		select {
		case <-remoteTicker.C:
			tramConf := config.StartramConfig
			for _, subd := range tramConf.Subdomains {
				if strings.Contains(subd.URL, patp) {
					if subd.Status != "ok" {
						continue
					}
				}
			}
			return
		}
	}
}

func errorCleanup(filename, patp, errmsg string) {
	// notify that we are cleaning up
	logger.Logger.Info(fmt.Sprintf("Pier import process failed: %s: %s", patp, errmsg))
	logger.Logger.Info(fmt.Sprintf("Running cleanup routine"))
	//remove file
	logger.Logger.Info(fmt.Sprintf("Removing %v", filename))
	os.Remove(filepath.Join(uploadDir, filename))
	// remove <patp>.json
	logger.Logger.Info(fmt.Sprintf("Removing Urbit Config: %s", patp))
	if err := config.RemoveUrbitConfig(patp); err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// remove patp from system.json
	logger.Logger.Info(fmt.Sprintf("Removing pier entry from System Config: %v", patp))
	err := shipcreator.RemoveSysConfigPier(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	// remove docker volume
	err = docker.DeleteVolume(patp)
	if err != nil {
		errmsg := fmt.Sprintf("%v", err)
		logger.Logger.Error(errmsg)
	}
	return
}

func restructureDirectory(patp string) error {
	logger.Logger.Info("Checking pier directory")
	// get docker volume path for patp
	volDir := ""
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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
		logger.Logger.Info("Custom pier location found!")
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
		return fmt.Errorf("No docker volume for %d!", patp)
	}
	logger.Logger.Info(fmt.Sprintf("%v pier path: %v", patp, volDir))
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
	logger.Logger.Debug(fmt.Sprintf(".urb subdirectory in %v", urbLoc[0]))
	pierDir := filepath.Join(volDir, patp)
	tempDir := filepath.Join(volDir, "temp_dir")
	unusedDir := filepath.Join(volDir, "unused")
	// move it into the right place
	if filepath.Join(pierDir, ".urb") != filepath.Join(urbLoc[0], ".urb") {
		logger.Logger.Info(".urb location incorrect! Restructuring directory structure")
		logger.Logger.Debug(fmt.Sprintf(".urb found in %v", urbLoc[0]))
		logger.Logger.Debug(fmt.Sprintf("Moving to %v", tempDir))
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
		logger.Logger.Info(fmt.Sprintf("%v restructuring done", patp))
	} else {
		logger.Logger.Debug("No restructuring needed")
	}
	return nil
}

func registerServices(patp string) {
	if err := startram.RegisterNewShip(patp); err != nil {
		logger.Logger.Error(fmt.Sprintf("Unable to register StarTram service for %s: %v", patp, err))
	}
}
