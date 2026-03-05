package system

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"groundseg/config"
	"groundseg/docker/events"
	"groundseg/dockerclient"
	"groundseg/internal/workflow"
	"groundseg/logger"
	"groundseg/shipworkflow"
	"groundseg/structs"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"go.uber.org/zap"
)

const maxBugReportErrorBodyBytes = 4096
const defaultBugEndpoint = "https://bugs.groundseg.app"

type supportRuntime struct {
	now                     func() time.Time
	unmarshal               func([]byte, any) error
	publishSystemTransition func(structs.SystemTransition)
	reportRootPath          func() (string, error)
	mkdirAllFn              func(string, os.FileMode) error
	removeAllFn             func(string) error
	zipDirFn                func(string, string) error
	dumpBugReportFn         func(string, string, string, string, []string, bool) error
	captureCPUProfileFn     func(string) error
	sendBugReportFn         func(bugReportPath, contact, description, endpoint string) error
	bugEndpoint             string
}

type supportRuntimeContract interface {
	currentTime() time.Time
	decodeJSON([]byte, any) error
	publishSupportTransition(structs.SystemTransition)
	bugReportRoot() (string, error)
	ensureDirectory(string, os.FileMode) error
	removeDirectory(string) error
	zipDirectory(string, string) error
	writeBugReport(string, string, string, string, []string, bool) error
	recordCPUProfile(string) error
	submitBugReport(string, string, string) error
}

type supportRequest struct {
	contact     string
	description string
	ships       []string
	cpuProfile  bool
	penpai      bool
	timestamp   string
}

func defaultSupportRuntime() supportRuntime {
	return supportRuntime{
		now:       time.Now,
		unmarshal: json.Unmarshal,
		publishSystemTransition: func(transition structs.SystemTransition) {
			_ = events.DefaultEventRuntime().PublishSystemTransition(context.Background(), transition)
		},
		reportRootPath:      makeBugReportPath,
		mkdirAllFn:          os.MkdirAll,
		removeAllFn:         os.RemoveAll,
		zipDirFn:            zipDir,
		dumpBugReportFn:     dumpBugReport,
		captureCPUProfileFn: captureCPUProfile,
		bugEndpoint:         defaultBugEndpoint,
		sendBugReportFn:     sendBugReportWithEndpoint,
	}
}

func (runtime supportRuntime) currentTime() time.Time {
	if runtime.now == nil {
		return time.Now()
	}
	return runtime.now()
}

func (runtime supportRuntime) decodeJSON(msg []byte, v any) error {
	if runtime.unmarshal == nil {
		return errors.New("support runtime JSON unmarshal callback is not configured")
	}
	return runtime.unmarshal(msg, v)
}

func (runtime supportRuntime) publishSupportTransition(transition structs.SystemTransition) {
	if runtime.publishSystemTransition == nil {
		return
	}
	runtime.publishSystemTransition(transition)
}

func (runtime supportRuntime) bugReportRoot() (string, error) {
	if runtime.reportRootPath == nil {
		return "", errors.New("support runtime bug report path callback is not configured")
	}
	return runtime.reportRootPath()
}

func (runtime supportRuntime) ensureDirectory(path string, perm os.FileMode) error {
	if runtime.mkdirAllFn == nil {
		return errors.New("support runtime mkdir callback is not configured")
	}
	return runtime.mkdirAllFn(path, perm)
}

func (runtime supportRuntime) removeDirectory(path string) error {
	if runtime.removeAllFn == nil {
		return errors.New("support runtime remove callback is not configured")
	}
	return runtime.removeAllFn(path)
}

func (runtime supportRuntime) zipDirectory(source, dest string) error {
	if runtime.zipDirFn == nil {
		return errors.New("support runtime zip callback is not configured")
	}
	return runtime.zipDirFn(source, dest)
}

func (runtime supportRuntime) writeBugReport(bugReportDir, timestamp, contact, description string, ships []string, llama bool) error {
	if runtime.dumpBugReportFn == nil {
		return errors.New("support runtime bug report callback is not configured")
	}
	return runtime.dumpBugReportFn(bugReportDir, timestamp, contact, description, ships, llama)
}

func (runtime supportRuntime) recordCPUProfile(path string) error {
	if runtime.captureCPUProfileFn == nil {
		return errors.New("support runtime CPU profile callback is not configured")
	}
	return runtime.captureCPUProfileFn(path)
}

func (runtime supportRuntime) submitBugReport(bugReportPath, contact, description string) error {
	if runtime.sendBugReportFn == nil {
		return errors.New("support runtime bug report submission callback is not configured")
	}
	return runtime.sendBugReportFn(bugReportPath, contact, description, runtime.bugEndpoint)
}

func InitializeSupport() {
	// Retained for compatibility; initialization is now performed lazily by SupportHandler.
}

// handle bug report requests
func SupportHandler(msg []byte) error {
	return SupportHandlerWithRuntime(msg, defaultSupportRuntime())
}

func SupportHandlerWithRuntime(msg []byte, rt supportRuntimeContract) error {
	flow := newSupportFlow(rt)
	return flow.Handle(msg)
}

type supportFlow struct {
	requestAdapter      supportRequestAdapter
	artifactService     supportArtifactService
	submissionClient    supportSubmissionClient
	transitionPublisher supportTransitionPublisher
}

func newSupportFlow(rt supportRuntimeContract) supportFlow {
	return supportFlow{
		requestAdapter: defaultSupportRequestAdapter{
			now:       rt.currentTime,
			unmarshal: rt.decodeJSON,
		},
		artifactService: defaultSupportArtifactService{
			runtime: rt,
		},
		submissionClient: defaultSupportSubmissionClient{
			runtime: rt,
		},
		transitionPublisher: defaultSupportTransitionPublisher{
			publish: rt.publishSupportTransition,
		},
	}
}

func (flow supportFlow) Handle(msg []byte) error {
	flow.transitionPublisher.PublishLoading()

	request, err := flow.requestAdapter.Decode(msg)
	if err != nil {
		operationErr := fmt.Errorf("couldn't unmarshal support payload: %w", err)
		flow.transitionPublisher.PublishFailure(operationErr.Error())
		return operationErr
	}

	bugReportDir, err := flow.artifactService.PrepareWorkspace(request.timestamp)
	if err != nil {
		operationErr := fmt.Errorf("error creating bug-report dir: %w", err)
		flow.transitionPublisher.PublishFailure(operationErr.Error())
		return operationErr
	}

	partialDumpErr, err := flow.artifactService.CollectArtifacts(request, bugReportDir)
	if err != nil {
		operationErr := fmt.Errorf("failed to dump logs: %w", err)
		flow.transitionPublisher.PublishFailure(operationErr.Error())
		return operationErr
	}

	zipPath, err := flow.artifactService.PackageArtifacts(bugReportDir, request.timestamp)
	if err != nil {
		operationErr := fmt.Errorf("error zipping bug report: %w", err)
		flow.transitionPublisher.PublishFailure(operationErr.Error())
		return operationErr
	}

	if err := flow.submissionClient.SubmitReport(zipPath, request.contact, request.description); err != nil {
		operationErr := fmt.Errorf("couldn't submit bug report: %w", err)
		flow.transitionPublisher.PublishFailure(operationErr.Error())
		return operationErr
	}

	if partialDumpErr != nil {
		operationErr := fmt.Errorf("bug report submitted with missing artifacts: %w", partialDumpErr)
		flow.transitionPublisher.PublishFailure(operationErr.Error())
		return operationErr
	}

	flow.transitionPublisher.PublishSuccess()
	return nil
}

type supportRequestAdapter interface {
	Decode([]byte) (supportRequest, error)
}

type defaultSupportRequestAdapter struct {
	unmarshal func([]byte, any) error
	now       func() time.Time
}

func (a defaultSupportRequestAdapter) Decode(msg []byte) (supportRequest, error) {
	return decodeSupportRequest(msg, a.now, a.unmarshal)
}

type supportArtifactService interface {
	PrepareWorkspace(timestamp string) (string, error)
	CollectArtifacts(req supportRequest, bugReportDir string) (*partialBugReportError, error)
	PackageArtifacts(bugReportDir, timestamp string) (string, error)
}

type defaultSupportArtifactService struct {
	runtime supportRuntimeContract
}

func (s defaultSupportArtifactService) PrepareWorkspace(timestamp string) (string, error) {
	return prepareSupportWorkspace(s.runtime, timestamp)
}

func (s defaultSupportArtifactService) CollectArtifacts(req supportRequest, bugReportDir string) (*partialBugReportError, error) {
	return collectSupportArtifacts(s.runtime, req, bugReportDir)
}

func (s defaultSupportArtifactService) PackageArtifacts(bugReportDir, timestamp string) (string, error) {
	return packageSupportArtifacts(s.runtime, bugReportDir, timestamp)
}

type supportSubmissionClient interface {
	SubmitReport(zipPath, contact, description string) error
}

type defaultSupportSubmissionClient struct {
	runtime supportRuntimeContract
}

func (c defaultSupportSubmissionClient) SubmitReport(zipPath, contact, description string) error {
	return c.runtime.submitBugReport(zipPath, contact, description)
}

type supportTransitionPublisher interface {
	PublishLoading()
	PublishSuccess()
	PublishFailure(message string)
}

type defaultSupportTransitionPublisher struct {
	publish func(structs.SystemTransition)
}

func (p defaultSupportTransitionPublisher) PublishLoading() {
	if p.publish == nil {
		return
	}
	p.publish(structs.SystemTransition{Type: "bugReport", Event: "loading"})
}

func (p defaultSupportTransitionPublisher) PublishSuccess() {
	if p.publish == nil {
		return
	}
	if err := shipworkflow.PublishTransitionWithPolicy(
		p.publish,
		structs.SystemTransition{Type: "bugReport", Event: "success"},
		structs.SystemTransition{Type: "bugReport", Event: "done"},
		3*time.Second,
	); err != nil {
		zap.L().Warn(fmt.Sprintf("failed to publish support success transition: %v", err))
	}
}

func (p defaultSupportTransitionPublisher) PublishFailure(message string) {
	if p.publish == nil {
		return
	}
	if err := shipworkflow.PublishTransitionWithPolicy(
		p.publish,
		structs.SystemTransition{Type: "bugReportError", Event: message},
		structs.SystemTransition{Type: "bugReportError", Event: ""},
		3*time.Second,
	); err != nil {
		zap.L().Warn(fmt.Sprintf("failed to publish support failure transition: %v", err))
	}
}

func decodeSupportRequest(msg []byte, now func() time.Time, unmarshal func([]byte, any) error) (supportRequest, error) {
	if now == nil {
		now = time.Now
	}
	if unmarshal == nil {
		unmarshal = json.Unmarshal
	}
	var supportPayload structs.WsSupportPayload
	if err := unmarshal(msg, &supportPayload); err != nil {
		return supportRequest{}, err
	}

	return supportRequest{
		contact:     supportPayload.Payload.Contact,
		description: supportPayload.Payload.Description,
		ships:       supportPayload.Payload.Ships,
		cpuProfile:  supportPayload.Payload.CPUProfile,
		penpai:      supportPayload.Payload.Penpai,
		timestamp:   fmt.Sprintf("%d", now().Unix()),
	}, nil
}

func prepareSupportWorkspace(rt supportRuntimeContract, timestamp string) (string, error) {
	reportRoot, err := rt.bugReportRoot()
	if err != nil {
		return "", err
	}
	bugReportDir := filepath.Join(reportRoot, timestamp)
	if err := rt.ensureDirectory(bugReportDir, 0755); err != nil {
		return "", err
	}
	return bugReportDir, nil
}

func collectSupportArtifacts(rt supportRuntimeContract, req supportRequest, bugReportDir string) (*partialBugReportError, error) {
	// write bug report to disk
	var partialDumpErr *partialBugReportError
	if err := rt.writeBugReport(bugReportDir, req.timestamp, req.contact, req.description, req.ships, req.penpai); err != nil {
		var partialErr *partialBugReportError
		if errors.As(err, &partialErr) {
			partialDumpErr = partialErr
			zap.L().Warn(fmt.Sprintf("Proceeding with partial bug report bundle: %v", partialErr))
		} else {
			return nil, err
		}
	}

	// collect cpu profile
	if req.cpuProfile {
		profilePath := filepath.Join(bugReportDir, "cpu.profile")
		if err := rt.recordCPUProfile(profilePath); err != nil {
			return nil, fmt.Errorf("couldn't collect cpu profile: %w", err)
		}
	}

	return partialDumpErr, nil
}

func packageSupportArtifacts(rt supportRuntimeContract, bugReportDir, timestamp string) (string, error) {
	reportRoot, err := rt.bugReportRoot()
	if err != nil {
		return "", err
	}
	zipPath := filepath.Join(reportRoot, timestamp+".zip")
	if err := rt.zipDirectory(bugReportDir, zipPath); err != nil {
		return "", err
	}

	// remove the bug report since we already have the zipped version
	if err := rt.removeDirectory(bugReportDir); err != nil {
		zap.L().Warn(fmt.Sprintf("couldn't remove report dir: %v", err))
	}
	return zipPath, nil
}

type partialBugReportError struct {
	err error
}

func (e *partialBugReportError) Error() string {
	return fmt.Sprintf("bug report bundle is partial: %v", e.err)
}

func (e *partialBugReportError) Unwrap() error {
	return e.err
}

// dump docker logs to path
func dumpDockerLogs(containerID string, path string) error {
	dockerClient, err := dockerclient.New()
	if err != nil {
		return fmt.Errorf("unable to create docker client: %w", err)
	}
	defer dockerClient.Close()
	// get previous 1k logs
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       "1000",
	}
	ctx := context.Background()
	existingLogs, err := dockerClient.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("error dumping %s logs: %w", containerID, err)
	}
	defer existingLogs.Close()
	allLogs, err := ioutil.ReadAll(existingLogs)
	if err != nil {
		return fmt.Errorf("error reading logs: %w", err)
	}
	var cleanedLogs strings.Builder
	reader := bytes.NewReader(allLogs)
	bufReader := bufio.NewReader(reader)
	for {
		header := make([]byte, 8)
		_, err := bufReader.Read(header)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading log header: %w", err)
		}
		line, err := bufReader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading log line: %w", err)
		}
		cleanedLogs.WriteString(string(line))
	}
	err = ioutil.WriteFile(path, []byte(cleanedLogs.String()), 0644)
	if err != nil {
		return fmt.Errorf("error writing logs to file: %w", err)
	}
	return nil
}

func dumpBugReport(bugReportDir, timestamp, contact, description string, piers []string, llama bool) error {
	var steps []workflow.Step

	// description.txt
	descPath := filepath.Join(bugReportDir, "description.txt")
	if err := ioutil.WriteFile(descPath, []byte(fmt.Sprintf("Contact:\n%s\nDetails:\n%s", contact, description)), 0644); err != nil {
		zap.L().Error(fmt.Sprintf("couldn't write details.txt"))
		return fmt.Errorf("unable to write details file: %w", err)
	}
	appendNonFatalStep := func(message string, run func() error) {
		steps = append(steps, workflow.Step{
			Name: message,
			Run:  run,
		})
	}

	// llama bug dump
	if llama {
		appendNonFatalStep("couldn't dump llama logs", func() error {
			return dumpDockerLogs("llama-gpt-api", bugReportDir+"/"+"llama.log")
		})
	}

	// selected pier logs
	for _, pier := range piers {
		p := pier
		appendNonFatalStep(fmt.Sprintf("couldn't dump pier logs for %s", p), func() error {
			return dumpDockerLogs(p, bugReportDir+"/"+p+".log")
		})
		appendNonFatalStep(fmt.Sprintf("couldn't dump minio logs for %s", p), func() error {
			return dumpDockerLogs("minio_"+p, bugReportDir+"/minio_"+p+".log")
		})
	}

	// service logs
	appendNonFatalStep("couldn't dump wireguard logs", func() error {
		return dumpDockerLogs("wireguard", bugReportDir+"/wireguard.log")
	})

	// system.json
	appendNonFatalStep("couldn't copy system config", func() error {
		srcPath := filepath.Join(config.BasePath(), "settings", "system.json")
		destPath := filepath.Join(bugReportDir, "system.json")
		return copyFile(srcPath, destPath)
	})
	appendNonFatalStep("couldn't sanitize system.json", func() error {
		destPath := filepath.Join(bugReportDir, "system.json")
		if err := sanitizeJSON(destPath, "sessions", "privkey", "salt", "pwHash"); err != nil && !os.IsNotExist(err) {
			zap.L().Error("couldn't sanitize system.json! Removing from report")
			if err := os.Remove(destPath); err != nil {
				return fmt.Errorf("unable to remove unsanitized system log: %w", err)
			}
			return err
		}
		return nil
	})

	// pier configs
	for _, pier := range piers {
		p := pier
		appendNonFatalStep(fmt.Sprintf("couldn't copy service config for %s", p), func() error {
			srcPath := filepath.Join(config.BasePath(), "settings", "pier", p+".json")
			destPath := filepath.Join(bugReportDir, p+".json")
			return copyFile(srcPath, destPath)
		})
		appendNonFatalStep(fmt.Sprintf("couldn't sanitize %s.json", p), func() error {
			destPath := filepath.Join(bugReportDir, p+".json")
			if err := sanitizeJSON(destPath, "minio_password"); err != nil && !os.IsNotExist(err) {
				zap.L().Error(fmt.Sprintf("couldn't sanitize %v.json! Removing from report", p))
				if err := os.Remove(destPath); err != nil {
					return fmt.Errorf("unable to remove unsanitized pier log: %w", err)
				}
				return err
			}
			return nil
		})
	}

	// service config jsons
	configFiles := []string{"mc.json", "netdata.json", "wireguard.json"}
	for _, configFile := range configFiles {
		cfg := configFile
		appendNonFatalStep(fmt.Sprintf("couldn't copy service config %s", cfg), func() error {
			srcPath := filepath.Join(config.BasePath(), "settings", cfg)
			destPath := filepath.Join(bugReportDir, cfg)
			return copyFile(srcPath, destPath)
		})
	}

	// current and previous syslogs
	sysLogs := lastTwoLogs()
	for _, sysLog := range sysLogs {
		logPath := sysLog
		appendNonFatalStep(fmt.Sprintf("couldn't copy system log %s", filepath.Base(logPath)), func() error {
			srcPath := logPath
			destPath := filepath.Join(bugReportDir, filepath.Base(logPath))
			return copyFile(srcPath, destPath)
		})
	}
	if joined := workflow.Join(steps, func(err error) {
		zap.L().Warn(err.Error())
	}); joined != nil {
		return &partialBugReportError{err: joined}
	}
	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return dstFile.Sync()
}

func zipDir(directory, dest string) error {
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	zipWriter := zip.NewWriter(destFile)
	defer zipWriter.Close()
	if err := filepath.Walk(directory, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(directory, filePath)
		if err != nil {
			return err
		}
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}
		fsFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer fsFile.Close()
		_, err = io.Copy(zipFile, fsFile)
		return err
	}); err != nil {
		return err
	}
	return nil
}

func sanitizeJSON(filePath string, keysToRemove ...string) error {
	// Read the file
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Unmarshal into a map
	var jsonData map[string]interface{}
	err = json.Unmarshal(data, &jsonData)
	if err != nil {
		return err
	}

	// Remove the keys
	for _, key := range keysToRemove {
		delete(jsonData, key)
	}

	// Marshal back to JSON
	sanitizedData, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return err
	}

	// Write back to the file
	err = ioutil.WriteFile(filePath, sanitizedData, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

func sendBugReportWithEndpoint(bugReportPath, contact, description, bugEndpoint string) error {
	uploadedFile, err := os.Open(bugReportPath)
	if err != nil {
		return fmt.Errorf("unable to open bug report archive: %w", err)
	}
	defer uploadedFile.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("contact", contact); err != nil {
		return fmt.Errorf("unable to write contact field: %w", err)
	}
	if err := writer.WriteField("string", description); err != nil {
		return fmt.Errorf("unable to write description field: %w", err)
	}
	part, err := writer.CreateFormFile("zip_file", bugReportPath)
	if err != nil {
		return fmt.Errorf("unable to create bug-report multipart field: %w", err)
	}
	_, err = io.Copy(part, uploadedFile)
	if err != nil {
		return fmt.Errorf("unable to copy bug-report archive into request body: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("unable to finalize bug-report multipart payload: %w", err)
	}
	req, err := http.NewRequest("POST", bugEndpoint, body)
	if err != nil {
		return fmt.Errorf("unable to construct bug-report request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	client.Timeout = 30 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to submit bug report request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxBugReportErrorBodyBytes))
		if readErr != nil {
			return fmt.Errorf("%s: failed to read error body: %w", resp.Status, readErr)
		}
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}
	zap.L().Info("Bug: Report submitted")
	return nil
}

func captureCPUProfile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	zap.L().Info("Profiling CPU for 30 seconds")
	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}
	time.Sleep(30 * time.Second)
	pprof.StopCPUProfile()
	return nil
}

func makeBugReportPath() (string, error) {
	return config.GetStoragePath("bug-reports")
}

// lastTwoLogs returns the two most recent log files from the directory.
func lastTwoLogs() []string {
	// Read the directory
	files, err := ioutil.ReadDir(logger.LogPath())
	if err != nil {
		fmt.Printf("Failed to read directory: %v\n", err)
		return nil
	}
	return logger.MostRecentPartedLogPaths(logger.LogPath(), files, 2)
}
