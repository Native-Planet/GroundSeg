package routines

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"groundseg/config"
	"groundseg/structs"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/slsa-framework/slsa-verifier/v2/cli/slsa-verifier/verify"
	"go.uber.org/zap"
)

const maxVersionUpdateConsecutiveFailures = 3

func StartVersionSubsystem() {
	if err := StartVersionSubsystemWithContext(context.Background()); err != nil {
		zap.L().Warn(fmt.Sprintf("version subsystem stopped with error: %v", err))
	}
}

func StartVersionSubsystemWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	errs := make(chan error, 2)
	go func() {
		errs <- CheckVersionLoopWithContext(ctx)
	}()
	go func() {
		errs <- AptUpdateLoopWithContext(ctx)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errs:
			return err
		}
	}
}

func CheckVersionLoop() {
	if err := CheckVersionLoopWithContext(context.Background()); err != nil {
		zap.L().Warn(fmt.Sprintf("version check loop stopped with error: %v", err))
	}
}

func CheckVersionLoopWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	rt := newVersionRuntime()
	conf := rt.configOps.getConfFn()
	var updateInterval int
	if conf.Runtime.UpdateInterval < 60 {
		updateInterval = 60
	} else {
		updateInterval = conf.Runtime.UpdateInterval
	}
	checkInterval := time.Duration(updateInterval) * time.Second
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	releaseChannel := conf.Connectivity.UpdateBranch
	if conf.Connectivity.UpdateMode == "auto" {
		consecutiveFailures := 0
		if err := callUpdater(ctx, rt, releaseChannel); err != nil {
			consecutiveFailures++
			if consecutiveFailures >= maxVersionUpdateConsecutiveFailures {
				return fmt.Errorf("version update failed after %d consecutive attempts: %w", maxVersionUpdateConsecutiveFailures, err)
			}
		} else {
			consecutiveFailures = 0
		}
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				if err := callUpdater(ctx, rt, releaseChannel); err != nil {
					consecutiveFailures++
					if consecutiveFailures >= maxVersionUpdateConsecutiveFailures {
						return fmt.Errorf("version update failed after %d consecutive attempts: %w", maxVersionUpdateConsecutiveFailures, err)
					}
				} else {
					consecutiveFailures = 0
				}
			}
		}
	}
	return nil
}

func callUpdater(ctx context.Context, rt versionRuntime, releaseChannel string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	// Get latest information
	basePathFn := rt.configOps.basePathFn
	if basePathFn == nil {
		basePathFn = func() string { return "" }
	}
	sha256Fn := rt.configOps.getSha256Fn
	if sha256Fn == nil {
		return errors.New("version updater getSha256 seam is unconfigured")
	}
	architectureFn := rt.configOps.architectureFn
	if architectureFn == nil {
		architectureFn = func() string { return "amd64" }
	}
	setVersionChannelFn := rt.channelOps.setVersionChannelFn

	latestVersion, synced := rt.channelOps.syncVersionInfoFn()
	if !synced {
		return errors.New("version metadata sync failed")
	}
	currentChannelVersion := rt.channelOps.getVersionChannelFn()
	latestChannelVersion := latestVersion
	// check docker updates
	if latestChannelVersion != currentChannelVersion {
		if rt.updateOps.updateDockerFn == nil {
			return errors.New("version updater docker runtime is unconfigured")
		}
		if err := rt.updateOps.updateDockerFn(rt.configOps, releaseChannel, currentChannelVersion, latestChannelVersion); err != nil {
			return fmt.Errorf("docker update failed: %w", err)
		}
		if setVersionChannelFn != nil {
			setVersionChannelFn(latestVersion)
		} else {
			return errors.New("version channel set seam is unconfigured")
		}
	}
	// Check for gs binary updates based on hash
	binPath := filepath.Join(basePathFn(), "groundseg")
	currentHash, err := sha256Fn(binPath)
	if err != nil {
		return fmt.Errorf("couldn't hash current binary: %w", err)
	}
	latestHash := latestVersion.Groundseg.Amd64Sha256
	if architectureFn() != "amd64" {
		latestHash = latestVersion.Groundseg.Arm64Sha256
	}
	if currentHash != latestHash {
		zap.L().Info("GroundSeg Binary update!")
		// updateBinary will likely restart the program, so
		// we don't have to care about the docker updates.
		if rt.updateOps.updateBinaryFn == nil {
			return errors.New("version updater binary runtime is unconfigured")
		}
		if err := rt.updateOps.updateBinaryFn(ctx, rt.updateOps, rt.configOps, releaseChannel, latestVersion); err != nil {
			return fmt.Errorf("binary update failed: %w", err)
		}
	}
	return nil
}

func updateBinary(
	ctx context.Context,
	updateOps versionUpdateOps,
	configOps versionConfigOps,
	branch string,
	versionInfo structs.Channel,
) error {
	// get config
	conf := configOps.getConfFn()
	var displayedBranch string
	if branch != "latest" {
		displayedBranch = fmt.Sprintf("-%v", branch)
	}
	msg := fmt.Sprintf(
		"A GroundSeg binary update detected! Current Version: %v%v , Latest Version v%v.%v.%v%v",
		conf.Runtime.GsVersion, displayedBranch,
		versionInfo.Groundseg.Major, versionInfo.Groundseg.Minor,
		versionInfo.Groundseg.Patch, displayedBranch,
	)
	zap.L().Info(msg)
	// delete old instance of groundseg_new if it exists
	if _, err := os.Stat(filepath.Join(configOps.basePathFn(), "groundseg_new")); err == nil {
		// Remove the file
		zap.L().Info("Deleting old groundseg_new download")
		if err := os.Remove(filepath.Join(configOps.basePathFn(), "groundseg_new")); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove old instance of groundseg_new: %v", err))
			return fmt.Errorf("remove old groundseg_new download: %w", err)
		}
	}
	// download new binary, name it groundseg_new
	url := versionInfo.Groundseg.Arm64URL
	if configOps.architectureFn() == "amd64" {
		url = versionInfo.Groundseg.Amd64URL
	}
	// Create a new HTTP GET request
	if updateOps.downloadFn == nil {
		return errors.New("version binary download runtime is unconfigured")
	}
	resp, err := updateOps.downloadFn(ctx, url)
	zap.L().Info(fmt.Sprintf("Downloading new GroundSeg binary from %v", url))
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to download new GroundSeg binary: %v", err))
		return fmt.Errorf("download new GroundSeg binary: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("couldn't download binary: status %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// Create a new file to save the downloaded content
	zap.L().Info("Creating groundseg_new")
	file, err := os.Create(filepath.Join(configOps.basePathFn(), "groundseg_new"))
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to save GroundSeg binary: %v", err))
		return fmt.Errorf("create groundseg_new download file: %w", err)
	}
	defer file.Close()
	zap.L().Info("Writing groundseg_new contents")
	// Write the contents from the HTTP response to the new file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to write contents: %v", err))
		return fmt.Errorf("write groundseg_new contents: %w", err)
	}
	// Chmod groundseg_new
	zap.L().Info("Modifying groundseg_new permissions")
	binaryPath := filepath.Join(configOps.basePathFn(), "groundseg_new")
	if err := os.Chmod(binaryPath, 0755); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to write contents: %v", err))
		return fmt.Errorf("chmod groundseg_new: %w", err)
	}
	newVersionHash := versionInfo.Groundseg.Arm64Sha256
	if configOps.architectureFn() == "amd64" {
		newVersionHash = versionInfo.Groundseg.Amd64Sha256
	}
	newBinHash, err := configOps.getSha256Fn(filepath.Join(configOps.basePathFn(), "groundseg_new"))
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't get SHA for new binary: %v", err))
		return fmt.Errorf("hash downloaded groundseg_new: %w", err)
	}
	if newVersionHash != newBinHash {
		zap.L().Error(fmt.Sprintf("New binary hash does not match downloaded file: remote %v / downloaded %v", newVersionHash, newBinHash))
		return fmt.Errorf("binary hash mismatch: remote %v / downloaded %v", newVersionHash, newBinHash)
	}
	if !conf.Startram.DisableSlsa {
		zap.L().Info("Verifying SLSA provenance")
		if err := verifySlsaProvenance(
			ctx,
			updateOps.downloadFn,
			versionInfo.Groundseg.SlsaURL,
			binaryPath,
			"git+https://github.com/Native-Planet/GroundSeg",
		); err != nil {
			zap.L().Error(fmt.Sprintf("SLSA verification failed: %v", err))
			return fmt.Errorf("SLSA verification failed: %w", err)
		}
		zap.L().Info("SLSA verification successful")
	} else {
		zap.L().Warn("SLSA verification disabled by configuration")
	}
	// delete groundseg binary if exists
	zap.L().Info("Deleting old groundseg")
	if _, err := os.Stat(filepath.Join(configOps.basePathFn(), "groundseg")); err == nil {
		// Remove the file
		if err := os.Remove(filepath.Join(configOps.basePathFn(), "groundseg")); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to remove old instance of groundseg: %v", err))
			return fmt.Errorf("remove old groundseg: %w", err)
		}
	}
	// rename groundseg_new to groundseg
	zap.L().Info("Renaming groundseg_new to groundseg")
	oldPath := filepath.Join(configOps.basePathFn(), "groundseg_new")
	newPath := filepath.Join(configOps.basePathFn(), "groundseg")
	if err := os.Rename(oldPath, newPath); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to rename groundseg_new to groundseg: %v", err))
		return fmt.Errorf("rename groundseg_new to groundseg: %w", err)
	}
	// re-disable bypass after one update
	if conf.Startram.DisableSlsa {
		if err := config.UpdateConfTyped(config.WithDisableSlsa(false)); err != nil {
			zap.L().Error(fmt.Sprintf("Couldn't reset SLSA bypass config: %v", err))
			return fmt.Errorf("reset disable slsa config: %w", err)
		}
	}
	versionStr := "v" + strconv.Itoa(versionInfo.Groundseg.Major) + "." +
		strconv.Itoa(versionInfo.Groundseg.Minor) + "." +
		strconv.Itoa(versionInfo.Groundseg.Patch)
	binHash, err := getSha256(newPath)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't hash new binary: %v", err))
		return fmt.Errorf("hash new groundseg binary: %w", err)
	}
	if err := config.UpdateConfTyped(
		config.WithGSVersion(versionStr),
		config.WithBinHash(binHash),
	); err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't update config: %v", err))
		return fmt.Errorf("update config with new version: %w", err)
	}
	// systemctl restart groundseg
	if configOps.debugModeFn() {
		zap.L().Debug("DebugMode detected. Skipping systemd command.")
		return nil
	} else {
		zap.L().Info("Restarting GroundSeg systemd service")
		cmd := exec.Command("systemctl", "restart", "groundseg")
		err := cmd.Run()
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to restart systemd service: %v", err))
			return fmt.Errorf("restart GroundSeg service: %w", err)
		}
	}
	return nil
}

func verifySlsaProvenance(
	ctx context.Context,
	downloadFn func(context.Context, string) (*http.Response, error),
	provenanceURL string,
	binaryPath string,
	sourceURI string,
) error {
	if downloadFn == nil {
		return fmt.Errorf("version update provenance download runtime is not configured")
	}
	if _, err := rekorKey(); err != nil {
		return fmt.Errorf("failed to ensure Rekor key is available: %w", err)
	}
	provenanceFile, err := os.CreateTemp("", "provenance-*.intoto.jsonl")
	if err != nil {
		return fmt.Errorf("failed to create temp file for provenance: %w", err)
	}
	defer os.Remove(provenanceFile.Name())
	defer provenanceFile.Close()
	resp, err := downloadFn(ctx, provenanceURL)
	if err != nil {
		return fmt.Errorf("failed to download provenance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to download provenance, status: %v", resp.StatusCode)
	}
	_, err = io.Copy(provenanceFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write provenance file: %w", err)
	}
	verifyCmd := &verify.VerifyArtifactCommand{
		ProvenancePath:  provenanceFile.Name(),
		SourceURI:       sourceURI,
		PrintProvenance: false,
	}
	verifyCtx := ctx
	if verifyCtx == nil {
		verifyCtx = context.Background()
	}
	trustedBuilder, err := verifyCmd.Exec(verifyCtx, []string{binaryPath})
	if err != nil {
		return fmt.Errorf("SLSA verification failed: %w", err)
	}
	zap.L().Info(fmt.Sprintf("Verified by trusted builder: %v", trustedBuilder))
	zap.L().Info("Info: https://github.blog/security/supply-chain-security/slsa-3-compliance-with-github-actions/")
	return nil
}

func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

func updateDockerForRuntime(configOps versionConfigOps, release string, currentVersion structs.Channel, latestVersion structs.Channel) error {
	zap.L().Info(fmt.Sprintf("update docker called: Current: %v , Latest %v", currentVersion, latestVersion))
	zap.L().Info(fmt.Sprintf(
		"New version available in %s channel! Current: %+v, Latest: %+v\n",
		release, currentVersion, latestVersion,
	))
	setBootStatus := func(pier, bootStatus string) error {
		if configOps.updateUrbitRuntimeConfigFn != nil {
			return configOps.updateUrbitRuntimeConfigFn(pier, func(urbConf *structs.UrbitRuntimeConfig) error {
				urbConf.BootStatus = bootStatus
				return nil
			})
		}
		return configOps.updateUrbitFn(pier, func(urbConf *structs.UrbitDocker) error {
			urbConf.BootStatus = bootStatus
			return nil
		})
	}
	conf := configOps.getConfFn()
	statuses, err := configOps.getShipStatusFn(conf.Connectivity.Piers)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Couldn't get ship statuses: %v", err))
		return fmt.Errorf("get ship statuses: %w", err)
	}
	var updateErrs []error
	valCurrent := reflect.ValueOf(currentVersion)
	valLatest := reflect.ValueOf(latestVersion)

	typeOfVersion := valCurrent.Type()

	for i := 0; i < valCurrent.NumField(); i++ {
		component := strings.ToLower(typeOfVersion.Field(i).Name)
		if component == "groundseg" {
			continue
		}
		currentDetail := valCurrent.Field(i).Interface().(structs.VersionDetails)
		latestDetail := valLatest.Field(i).Interface().(structs.VersionDetails)
		hashChanged := latestDetail.Amd64Sha256 != currentDetail.Amd64Sha256
		if configOps.architectureFn() != "amd64" {
			hashChanged = latestDetail.Arm64Sha256 != currentDetail.Arm64Sha256
		}
		if !hashChanged {
			continue
		}
		switch component {
		case "netdata", "wireguard", "miniomc":
			if _, err := configOps.startContainerFn(component, component); err != nil {
				zap.L().Warn(fmt.Sprintf("Failed to refresh %s image: %v", component, err))
				updateErrs = append(updateErrs, fmt.Errorf("refresh %s image: %w", component, err))
			}
		case "vere":
			for pier, status := range statuses {
				isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
				if err := configOps.loadUrbitConfigFn(pier); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to load config for %s: %v", pier, err))
					updateErrs = append(updateErrs, fmt.Errorf("load %s urbit config: %w", pier, err))
					continue
				}
				// Stop ship if running
				if isRunning {
					zap.L().Info(fmt.Sprintf("Stopping %s for vere upgrade", pier))
					if err := configOps.stopContainerFn(pier); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to stop %s: %v", pier, err))
						updateErrs = append(updateErrs, fmt.Errorf("stop %s before vere upgrade: %w", pier, err))
						continue
					}
				}

				// Run urbit prep with old image (always, regardless of running status)
				zap.L().Info(fmt.Sprintf("Running urbit prep for %s with old vere image before upgrade", pier))
				if err := setBootStatus(pier, "prep"); err != nil {
					zap.L().Error(fmt.Sprintf("Failed to update %s config for prep: %v", pier, err))
					updateErrs = append(updateErrs, fmt.Errorf("set %s boot status to prep: %w", pier, err))
					continue
				}

				// Start container to run prep
				_, err := configOps.startContainerFn(pier, "vere")
				if err != nil {
					zap.L().Error(fmt.Sprintf("Failed to run prep for %s: %v", pier, err))
					updateErrs = append(updateErrs, fmt.Errorf("start prep container for %s: %w", pier, err))
					continue
				}

				// Wait for prep to complete
				zap.L().Info(fmt.Sprintf("Waiting for prep to complete for %s", pier))
				if err := configOps.waitCompleteFn(pier); err != nil {
					zap.L().Error(fmt.Sprintf("Wait for prep completion failed for %s: %v", pier, err))
					updateErrs = append(updateErrs, fmt.Errorf("wait complete for %s: %w", pier, err))
					continue
				}

				// Set boot status appropriately after prep
				if isRunning {
					// Ship was running before, boot it with new image
					zap.L().Info(fmt.Sprintf("Starting %s with new vere image", pier))
					if err := setBootStatus(pier, "boot"); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to update %s config for boot: %v", pier, err))
						updateErrs = append(updateErrs, fmt.Errorf("set %s boot status to boot: %w", pier, err))
						continue
					}
					_, err = configOps.startContainerFn(pier, "vere")
					if err != nil {
						zap.L().Error(fmt.Sprintf("Failed to start %s after vere update: %v", pier, err))
						updateErrs = append(updateErrs, fmt.Errorf("start %s after vere update: %w", pier, err))
						continue
					}
				} else {
					// Ship was not running, keep it stopped but update config
					zap.L().Info(fmt.Sprintf("%s prep complete, keeping ship stopped", pier))
					if err := setBootStatus(pier, "noboot"); err != nil {
						zap.L().Error(fmt.Sprintf("Failed to update %s config after prep: %v", pier, err))
						updateErrs = append(updateErrs, fmt.Errorf("set %s boot status to noboot: %w", pier, err))
					}
				}

				// Check if it wants a chop after upgrade (only if running)
				if isRunning {
					conf := configOps.urbitConfFn(pier)
					if conf.ChopOnUpgrade {
						go configOps.chopPierFn(pier)
					}
				}
			}
		case "minio":
			for pier, status := range statuses {
				isRunning := (status == "Up" || strings.HasPrefix(status, "Up "))
				if isRunning {
					if _, err := configOps.startContainerFn("minio_"+pier, "minio"); err != nil {
						zap.L().Warn(fmt.Sprintf("Failed to refresh minio for %s: %v", pier, err))
						updateErrs = append(updateErrs, fmt.Errorf("refresh minio for %s: %w", pier, err))
					}
				}
			}
		}
	}
	return errors.Join(updateErrs...)
}

func getSha256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	hashValue := hex.EncodeToString(hasher.Sum(nil))
	return hashValue, nil
}
