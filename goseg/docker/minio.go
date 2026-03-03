package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"groundseg/structs"
	"os"
	"path/filepath"
	"strings"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"go.uber.org/zap"
)

const (
	defaultRustFSImageRef = "rustfs/rustfs:latest"
	legacyMinIOImageRef   = "minio/minio:latest"
	minioMigrationMarker  = ".groundseg-rustfs-migration-complete"
)

func objectStoreImageRef() string {
	if image := strings.TrimSpace(os.Getenv("GROUNDSEG_S3_IMAGE")); image != "" {
		return image
	}
	if image := strings.TrimSpace(os.Getenv("GROUNDSEG_RUSTFS_IMAGE")); image != "" {
		return image
	}
	return defaultRustFSImageRef
}

// Returns "<repo>", "<tag>" from configured object storage image ref.
func GetObjectStoreRepoTag() (string, string) {
	ref := objectStoreImageRef()
	if at := strings.Index(ref, "@"); at >= 0 {
		ref = ref[:at]
	}
	repo := ref
	tag := "latest"
	lastSlash := strings.LastIndex(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	if lastColon > lastSlash {
		repo = ref[:lastColon]
		tag = ref[lastColon+1:]
	}
	return repo, tag
}

func GetObjectStoreContainerName(patp string) string {
	return fmt.Sprintf("rustfs_%s", patp)
}

func GetLegacyMinIOContainerName(patp string) string {
	return fmt.Sprintf("minio_%s", patp)
}

func GetObjectStoreDataVolumeName(patp string) string {
	return fmt.Sprintf("rustfs_%s", patp)
}

// Backward-compatible wrapper used by existing call sites.
func GetMinIODataVolumeName(containerName string) string {
	patp, err := getPatpFromObjectStoreName(containerName)
	if err != nil {
		return containerName
	}
	return GetObjectStoreDataVolumeName(patp)
}

func IsObjectStoreContainerName(containerName string) bool {
	return strings.HasPrefix(containerName, "rustfs_") || strings.HasPrefix(containerName, "minio_")
}

func GetPatpFromObjectStoreContainer(containerName string) (string, error) {
	return getPatpFromObjectStoreName(containerName)
}

func getPatpFromObjectStoreName(containerName string) (string, error) {
	switch {
	case strings.HasPrefix(containerName, "rustfs_"):
		splitStr := strings.SplitN(containerName, "_", 2)
		if len(splitStr) < 2 || splitStr[1] == "" {
			return "", fmt.Errorf("invalid RustFS container name")
		}
		return splitStr[1], nil
	case strings.HasPrefix(containerName, "minio_"):
		splitStr := strings.SplitN(containerName, "_", 2)
		if len(splitStr) < 2 || splitStr[1] == "" {
			return "", fmt.Errorf("invalid MinIO container name")
		}
		return splitStr[1], nil
	default:
		return "", fmt.Errorf("invalid object store container name")
	}
}

func setObjectStorePassword(patp, password string) error {
	if err := config.SetMinIOPassword(GetObjectStoreContainerName(patp), password); err != nil {
		return err
	}
	// Backfill the legacy key for compatibility with any older reads.
	_ = config.SetMinIOPassword(GetLegacyMinIOContainerName(patp), password)
	return nil
}

func getObjectStorePassword(patp string) (string, error) {
	pwd, err := config.GetMinIOPassword(GetObjectStoreContainerName(patp))
	if err == nil {
		return pwd, nil
	}
	return config.GetMinIOPassword(GetLegacyMinIOContainerName(patp))
}

// Legacy no-op: mc container is no longer required.
func LoadMC() error {
	return nil
}

// iterate through each ship and create object stores
func LoadMinIOs() error {
	return LoadObjectStores()
}

func LoadObjectStores() error {
	conf := config.Conf()
	if !conf.WgRegistered {
		return nil
	}
	if err := CleanupLegacyMinIOContainers(); err != nil {
		zap.L().Warn(fmt.Sprintf("Legacy MinIO container cleanup encountered errors: %v", err))
	}
	zap.L().Info("Loading RustFS containers")
	for _, pier := range conf.Piers {
		label := GetObjectStoreContainerName(pier)
		info, err := StartContainer(label, "minio")
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error starting %s RustFS: %v", pier, err))
			continue
		}
		config.UpdateContainerState(label, info)
	}
	return nil
}

// On upgrade, remove only legacy minio_<ship> containers for ships recorded in GroundSeg config.
// Volumes are intentionally preserved.
func CleanupLegacyMinIOContainers() error {
	conf := config.Conf()
	var errs []string
	for _, pier := range conf.Piers {
		legacy := GetLegacyMinIOContainerName(pier)
		if _, err := FindContainer(legacy); err == nil {
			if err := DeleteContainer(legacy); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %v", legacy, err))
			} else {
				zap.L().Info(fmt.Sprintf("Removed legacy MinIO container %s", legacy))
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// object store container config builder
func minioContainerConf(containerName string) (container.Config, container.HostConfig, error) {
	var containerConfig container.Config
	var hostConfig container.HostConfig
	shipName, err := getPatpFromObjectStoreName(containerName)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	err = config.LoadUrbitConfig(shipName)
	if err != nil {
		errmsg := fmt.Errorf("Error loading %s config: %v", shipName, err)
		return containerConfig, hostConfig, errmsg
	}
	shipConf := config.UrbitConf(shipName)

	randomBytes := make([]byte, 16)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return containerConfig, hostConfig, err
	}
	storePwd := hex.EncodeToString(randomBytes)
	if err := setObjectStorePassword(shipName, storePwd); err != nil {
		return containerConfig, hostConfig, err
	}

	environment := []string{
		fmt.Sprintf("RUSTFS_ACCESS_KEY=%s", shipName),
		fmt.Sprintf("RUSTFS_SECRET_KEY=%s", storePwd),
		fmt.Sprintf("RUSTFS_ADDRESS=0.0.0.0:%v", shipConf.WgS3Port),
		fmt.Sprintf("RUSTFS_CONSOLE_ADDRESS=0.0.0.0:%v", shipConf.WgConsolePort),
		"RUSTFS_CONSOLE_ENABLE=true",
		"RUSTFS_VOLUMES=/data",
		fmt.Sprintf("RUSTFS_SERVER_DOMAINS=s3.%s", shipConf.WgURL),
	}
	mounts := []mount.Mount{
		{
			Type:   mount.TypeVolume,
			Source: GetObjectStoreDataVolumeName(shipName),
			Target: "/data",
		},
	}
	containerConfig = container.Config{
		Image: objectStoreImageRef(),
		Env:   environment,
		// RustFS image defaults to non-root; old Docker volumes are often root-owned.
		User: "0:0",
	}
	hostConfig = container.HostConfig{
		NetworkMode: "container:wireguard",
		Mounts:      mounts,
	}
	ticker := time.NewTicker(500 * time.Millisecond)
minioNetworkLoop:
	for {
		select {
		case <-ticker.C:
			status, err := GetContainerRunningStatus("wireguard")
			if err != nil {
				return containerConfig, hostConfig, err
			}
			if strings.Contains(status, "Up") {
				break minioNetworkLoop
			}
		}
	}
	return containerConfig, hostConfig, nil
}

func newS3Client(endpoint string, accessKey string, secretKey string) (*s3v2.Client, error) {
	cfg, err := awscfg.LoadDefaultConfig(
		context.Background(),
		awscfg.WithRegion("us-east-1"),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		awscfg.WithEndpointResolverWithOptions(awsv2.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (awsv2.Endpoint, error) {
				if service == s3v2.ServiceID {
					return awsv2.Endpoint{
						URL:               endpoint,
						HostnameImmutable: true,
						SigningRegion:     region,
					}, nil
				}
				return awsv2.Endpoint{}, &awsv2.EndpointNotFoundError{}
			},
		)),
	)
	if err != nil {
		return nil, err
	}
	client := s3v2.NewFromConfig(cfg, func(o *s3v2.Options) {
		o.UsePathStyle = true
	})
	return client, nil
}

func bucketNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "nosuchbucket") || strings.Contains(msg, "notfound") || strings.Contains(msg, "404")
}

func bucketAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "bucketalreadyownedbyyou") || strings.Contains(msg, "bucketalreadyexists")
}

func ensureBucketExists(client *s3v2.Client, bucket string) error {
	_, err := client.HeadBucket(context.Background(), &s3v2.HeadBucketInput{Bucket: awsv2.String(bucket)})
	if err == nil {
		return nil
	}
	if !bucketNotFound(err) {
		// Could be startup race; try create anyway.
		zap.L().Debug(fmt.Sprintf("HeadBucket returned non-notfound, trying create: %v", err))
	}
	_, err = client.CreateBucket(context.Background(), &s3v2.CreateBucketInput{Bucket: awsv2.String(bucket)})
	if err != nil && !bucketAlreadyExists(err) {
		return err
	}
	return nil
}

func setPublicReadPolicy(client *s3v2.Client, bucket string) error {
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":"*","Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`,
		bucket,
	)
	_, err := client.PutBucketPolicy(context.Background(), &s3v2.PutBucketPolicyInput{
		Bucket: awsv2.String(bucket),
		Policy: awsv2.String(policy),
	})
	return err
}

func waitForS3Ready(client *s3v2.Client) error {
	var lastErr error
	for i := 0; i < 30; i++ {
		_, err := client.ListBuckets(context.Background(), &s3v2.ListBucketsInput{})
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out waiting for S3 readiness: %w", lastErr)
}

func migrateBucketObjects(source *s3v2.Client, target *s3v2.Client, bucket string) error {
	paginator := s3v2.NewListObjectsV2Paginator(source, &s3v2.ListObjectsV2Input{Bucket: awsv2.String(bucket)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			if bucketNotFound(err) {
				return fmt.Errorf("legacy bucket missing")
			}
			return err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || *obj.Key == "" {
				continue
			}
			getOut, err := source.GetObject(context.Background(), &s3v2.GetObjectInput{
				Bucket: awsv2.String(bucket),
				Key:    obj.Key,
			})
			if err != nil {
				return err
			}
			_, putErr := target.PutObject(context.Background(), &s3v2.PutObjectInput{
				Bucket:             awsv2.String(bucket),
				Key:                obj.Key,
				Body:               getOut.Body,
				ContentType:        getOut.ContentType,
				ContentEncoding:    getOut.ContentEncoding,
				ContentDisposition: getOut.ContentDisposition,
				ContentLanguage:    getOut.ContentLanguage,
				CacheControl:       getOut.CacheControl,
				Metadata:           getOut.Metadata,
			})
			_ = getOut.Body.Close()
			if putErr != nil {
				return putErr
			}
		}
	}
	return nil
}

func setMinIOAdminAccount(containerName string) error {
	patp, err := getPatpFromObjectStoreName(containerName)
	if err != nil {
		return err
	}
	if err := config.LoadUrbitConfig(patp); err != nil {
		return err
	}
	urbConf := config.UrbitConf(patp)
	pwd, err := getObjectStorePassword(patp)
	if err != nil {
		return err
	}
	targetClient, err := newS3Client(fmt.Sprintf("http://localhost:%v", urbConf.WgS3Port), patp, pwd)
	if err != nil {
		return err
	}
	if err := ensureBucketExists(targetClient, "bucket"); err != nil {
		return err
	}
	if err := maybeMigrateLegacyMinIOData(patp, urbConf, targetClient); err != nil {
		zap.L().Warn(fmt.Sprintf("Legacy MinIO migration failed for %s: %v", patp, err))
	}
	if err := setPublicReadPolicy(targetClient, "bucket"); err != nil {
		return err
	}
	return nil
}

func maybeMigrateLegacyMinIOData(patp string, urbConf structs.UrbitDocker, targetClient *s3v2.Client) error {
	legacyVolume := GetLegacyMinIOContainerName(patp)
	targetVolume := GetObjectStoreDataVolumeName(patp)

	legacyExists, err := volumeExists(legacyVolume)
	if err != nil {
		return err
	}
	if !legacyExists {
		return nil
	}
	targetExists, err := volumeExists(targetVolume)
	if err != nil {
		return err
	}
	if !targetExists {
		return nil
	}
	hasMarker, err := volumeHasMarker(targetVolume, minioMigrationMarker)
	if err == nil && hasMarker {
		return nil
	}
	legacyHasData, err := volumeHasAnyData(legacyVolume)
	if err != nil {
		return err
	}
	if !legacyHasData {
		return writeMigrationMarker(targetVolume, "legacy-empty")
	}
	targetHasData, err := volumeHasAnyData(targetVolume)
	if err != nil {
		return err
	}
	if targetHasData {
		return writeMigrationMarker(targetVolume, "target-populated")
	}

	sourceSecret, err := randomHex(16)
	if err != nil {
		return err
	}
	sourcePort := urbConf.WgS3Port + 10000
	if sourcePort > 64000 || sourcePort == urbConf.WgS3Port {
		sourcePort = 39000 + (urbConf.WgS3Port % 1000)
	}
	sourceConsolePort := sourcePort + 1
	sourceContainer := fmt.Sprintf("minio_migrate_%s", patp)
	if len(sourceContainer) > 63 {
		sourceContainer = sourceContainer[:63]
	}

	zap.L().Info(fmt.Sprintf("Migrating legacy MinIO volume %s -> %s for %s", legacyVolume, targetVolume, patp))
	if err := startLegacyMinIOMigrationSource(sourceContainer, legacyVolume, patp, sourceSecret, sourcePort, sourceConsolePort); err != nil {
		return err
	}
	defer func() {
		if err := DeleteContainer(sourceContainer); err != nil {
			zap.L().Warn(fmt.Sprintf("Failed to clean up migration source %s: %v", sourceContainer, err))
		}
	}()

	sourceClient, err := newS3Client(fmt.Sprintf("http://localhost:%d", sourcePort), patp, sourceSecret)
	if err != nil {
		return err
	}
	if err := waitForS3Ready(sourceClient); err != nil {
		return err
	}
	if err := migrateBucketObjects(sourceClient, targetClient, "bucket"); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "legacy bucket missing") {
			return writeMigrationMarker(targetVolume, "legacy-no-bucket")
		}
		return fmt.Errorf("failed to migrate legacy bucket: %w", err)
	}
	return writeMigrationMarker(targetVolume, "ok")
}

func startLegacyMinIOMigrationSource(containerName, volumeName, accessKey, secretKey string, s3Port, consolePort int) error {
	_ = DeleteContainer(containerName)
	if err := PullImageByRef(legacyMinIOImageRef); err != nil {
		return err
	}

	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()
	containerConfig := container.Config{
		Image:      legacyMinIOImageRef,
		Entrypoint: []string{"minio"},
		Cmd: []string{
			"server",
			"/data",
			"--address", fmt.Sprintf(":%d", s3Port),
			"--console-address", fmt.Sprintf(":%d", consolePort),
		},
		Env: []string{
			fmt.Sprintf("MINIO_ROOT_USER=%s", accessKey),
			fmt.Sprintf("MINIO_ROOT_PASSWORD=%s", secretKey),
		},
	}
	hostConfig := container.HostConfig{
		NetworkMode: "container:wireguard",
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/data",
			},
		},
	}

	if _, err := cli.ContainerCreate(ctx, &containerConfig, &hostConfig, nil, nil, containerName); err != nil {
		return err
	}
	return cli.ContainerStart(ctx, containerName, container.StartOptions{})
}

func randomHex(length int) (string, error) {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func volumeMountpoint(volumeName string) (string, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return "", err
	}
	defer cli.Close()
	vol, err := cli.VolumeInspect(context.Background(), volumeName)
	if err != nil {
		return "", err
	}
	return vol.Mountpoint, nil
}

func volumeHasAnyData(volumeName string) (bool, error) {
	mountpoint, err := volumeMountpoint(volumeName)
	if err != nil {
		return false, err
	}
	entries, err := os.ReadDir(mountpoint)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func volumeHasMarker(volumeName, marker string) (bool, error) {
	mountpoint, err := volumeMountpoint(volumeName)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(filepath.Join(mountpoint, marker))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func writeMigrationMarker(volumeName, status string) error {
	content := fmt.Sprintf("%s %s\n", status, time.Now().UTC().Format(time.RFC3339))
	return WriteFileToVolume(volumeName, minioMigrationMarker, content)
}

func CreateObjectStoreCredentials(patp string) (structs.MinIOServiceAccount, error) {
	var svcAccount structs.MinIOServiceAccount
	svcAccount.Alias = fmt.Sprintf("patp_%s", patp)
	svcAccount.User = patp
	pwd, err := getObjectStorePassword(patp)
	if err != nil {
		return svcAccount, err
	}
	// Native RustFS setup: use per-ship root credentials.
	svcAccount.AccessKey = patp
	svcAccount.SecretKey = pwd
	return svcAccount, nil
}

func CreateMinIOServiceAccount(patp string) (structs.MinIOServiceAccount, error) {
	return CreateObjectStoreCredentials(patp)
}
