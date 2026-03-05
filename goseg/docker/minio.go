package docker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"groundseg/config"
	"groundseg/dockerclient"
	"groundseg/structs"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	legacyMinIOImageRef   = "registry.hub.docker.com/minio/minio:latest"
	minioMigrationMarker  = ".groundseg-rustfs-migration-complete"
	objectStoreLinkMarker = ".groundseg-rustfs-link-configured"
)

var (
	forceLegacyMigration          bool
	forceLegacyMigrationConsumed  = make(map[string]bool)
	forceLegacyMigrationConsumedM sync.Mutex
	legacyEmptyRetryConsumed      = make(map[string]bool)
	legacyEmptyRetryConsumedM     sync.Mutex
)

func SetForceLegacyMigration(force bool) {
	forceLegacyMigrationConsumedM.Lock()
	defer forceLegacyMigrationConsumedM.Unlock()
	forceLegacyMigration = force
	if !force {
		forceLegacyMigrationConsumed = make(map[string]bool)
	}
}

func consumeForceLegacyMigrationForShip(patp string) bool {
	forceLegacyMigrationConsumedM.Lock()
	defer forceLegacyMigrationConsumedM.Unlock()
	if !forceLegacyMigration {
		return false
	}
	if forceLegacyMigrationConsumed[patp] {
		return false
	}
	forceLegacyMigrationConsumed[patp] = true
	return true
}

func consumeLegacyEmptyRetryForShip(patp string) bool {
	legacyEmptyRetryConsumedM.Lock()
	defer legacyEmptyRetryConsumedM.Unlock()
	if legacyEmptyRetryConsumed[patp] {
		return false
	}
	legacyEmptyRetryConsumed[patp] = true
	return true
}

func markLegacyEmptyRetryConsumed(patp string) {
	legacyEmptyRetryConsumedM.Lock()
	defer legacyEmptyRetryConsumedM.Unlock()
	legacyEmptyRetryConsumed[patp] = true
}

func objectStoreContainerInfo() (map[string]string, error) {
	// Preferred path: explicit rustfs item on version server.
	if info, err := GetLatestContainerInfo("rustfs"); err == nil {
		repo := strings.TrimSpace(info["repo"])
		tag := strings.TrimSpace(info["tag"])
		if repo != "" {
			if tag == "" {
				tag = "latest"
			}
			info["tag"] = tag
			return info, nil
		}
	}
	// Compatibility path: minio item pointed at rustfs.
	info, err := GetLatestContainerInfo("minio")
	if err != nil {
		return nil, err
	}
	repo := strings.TrimSpace(info["repo"])
	tag := strings.TrimSpace(info["tag"])
	if repo == "" {
		return nil, fmt.Errorf("empty object store repo from version info")
	}
	if tag == "" {
		tag = "latest"
	}
	info["tag"] = tag
	return info, nil
}

func objectStoreImageRef() string {
	if image := strings.TrimSpace(os.Getenv("GROUNDSEG_S3_IMAGE")); image != "" {
		return image
	}
	if image := strings.TrimSpace(os.Getenv("GROUNDSEG_RUSTFS_IMAGE")); image != "" {
		return image
	}
	if containerInfo, err := objectStoreContainerInfo(); err == nil {
		repo := strings.TrimSpace(containerInfo["repo"])
		tag := strings.TrimSpace(containerInfo["tag"])
		hash := strings.TrimSpace(containerInfo["hash"])
		if strings.Contains(strings.ToLower(repo), "rustfs") && hash != "" && !strings.EqualFold(hash, "none") {
			return fmt.Sprintf("%s:%s@sha256:%s", repo, tag, hash)
		}
		return fmt.Sprintf("%s:%s", repo, tag)
	}
	return defaultRustFSImageRef
}

func legacyMinIOMigrationImageRef() string {
	if image := strings.TrimSpace(os.Getenv("GROUNDSEG_LEGACY_MINIO_IMAGE")); image != "" {
		return image
	}
	containerInfo, err := GetLatestContainerInfo("minio")
	if err == nil {
		repo := strings.TrimSpace(containerInfo["repo"])
		tag := strings.TrimSpace(containerInfo["tag"])
		hash := strings.TrimSpace(containerInfo["hash"])
		// Use version-server minio value only when it actually points to a MinIO image.
		if strings.Contains(strings.ToLower(repo), "minio") && repo != "" && tag != "" && hash != "" {
			return fmt.Sprintf("%s:%s@sha256:%s", repo, tag, hash)
		}
		if strings.Contains(strings.ToLower(repo), "minio") && repo != "" {
			if tag == "" {
				tag = "latest"
			}
			return fmt.Sprintf("%s:%s", repo, tag)
		}
	}
	return legacyMinIOImageRef
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

	storePwd := strings.TrimSpace(shipConf.MinioPassword)
	if storePwd == "" {
		randomBytes := make([]byte, 16)
		_, err = rand.Read(randomBytes)
		if err != nil {
			return containerConfig, hostConfig, err
		}
		storePwd = hex.EncodeToString(randomBytes)
		shipConf.MinioPassword = storePwd
		update := map[string]structs.UrbitDocker{shipName: shipConf}
		if err := config.UpdateUrbitConfig(update); err != nil {
			return containerConfig, hostConfig, err
		}
	}
	if err := setObjectStorePassword(shipName, storePwd); err != nil {
		return containerConfig, hostConfig, err
	}

	serverDomains := objectStoreServerDomains(shipConf)
	environment := []string{
		fmt.Sprintf("RUSTFS_ACCESS_KEY=%s", shipName),
		fmt.Sprintf("RUSTFS_SECRET_KEY=%s", storePwd),
		fmt.Sprintf("RUSTFS_ADDRESS=0.0.0.0:%v", shipConf.WgS3Port),
		fmt.Sprintf("RUSTFS_CONSOLE_ADDRESS=0.0.0.0:%v", shipConf.WgConsolePort),
		"RUSTFS_CONSOLE_ENABLE=true",
		"RUSTFS_VOLUMES=/data",
	}
	if serverDomains != "" {
		environment = append(environment, fmt.Sprintf("RUSTFS_SERVER_DOMAINS=%s", serverDomains))
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

func normalizeObjectStoreDomain(domain string) string {
	domain = strings.TrimSpace(domain)
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimPrefix(domain, "http://")
	if cut := strings.IndexRune(domain, '/'); cut >= 0 {
		domain = domain[:cut]
	}
	return strings.TrimSpace(strings.Trim(domain, "/"))
}

func objectStoreServerDomains(shipConf structs.UrbitDocker) string {
	var domains []string
	baseDomain := strings.TrimSpace(shipConf.WgURL)
	if baseDomain != "" {
		defaultDomain := normalizeObjectStoreDomain(fmt.Sprintf("s3.%s", baseDomain))
		if defaultDomain != "" {
			domains = append(domains, defaultDomain)
		}
	}
	customDomain := normalizeObjectStoreDomain(shipConf.CustomS3Web)
	if customDomain != "" && !contains(domains, customDomain) {
		domains = append(domains, customDomain)
	}
	return strings.Join(domains, ",")
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

func bucketHasObjects(client *s3v2.Client, bucket string) (bool, error) {
	out, err := client.ListObjectsV2(context.Background(), &s3v2.ListObjectsV2Input{
		Bucket:  awsv2.String(bucket),
		MaxKeys: awsv2.Int32(1),
	})
	if err != nil {
		if bucketNotFound(err) {
			return false, fmt.Errorf("bucket missing")
		}
		return false, err
	}
	return len(out.Contents) > 0, nil
}

func listBucketNames(client *s3v2.Client) ([]string, error) {
	out, err := client.ListBuckets(context.Background(), &s3v2.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, bucket := range out.Buckets {
		if bucket.Name != nil && strings.TrimSpace(*bucket.Name) != "" {
			names = append(names, *bucket.Name)
		}
	}
	return names, nil
}

func getContainerIPv4(containerName string) (string, error) {
	cli, err := dockerclient.New()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	inspect, err := cli.ContainerInspect(context.Background(), containerName)
	if err != nil {
		return "", err
	}
	if inspect.NetworkSettings == nil {
		return "", fmt.Errorf("container %s has no network settings", containerName)
	}
	for _, netInfo := range inspect.NetworkSettings.Networks {
		if netInfo != nil && strings.TrimSpace(netInfo.IPAddress) != "" {
			return netInfo.IPAddress, nil
		}
	}
	return "", fmt.Errorf("container %s has no IPv4 address", containerName)
}

func wireguardEndpoint(port int) (string, error) {
	ip, err := getContainerIPv4("wireguard")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("http://%s:%d", ip, port), nil
}

func migrateBucketObjects(source *s3v2.Client, target *s3v2.Client, sourceBucket string, targetBucket string) (int, error) {
	copied := 0
	paginator := s3v2.NewListObjectsV2Paginator(source, &s3v2.ListObjectsV2Input{Bucket: awsv2.String(sourceBucket)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			if bucketNotFound(err) {
				return copied, fmt.Errorf("legacy bucket missing")
			}
			return copied, err
		}
		for _, obj := range page.Contents {
			if obj.Key == nil || *obj.Key == "" {
				continue
			}
			getOut, err := source.GetObject(context.Background(), &s3v2.GetObjectInput{
				Bucket: awsv2.String(sourceBucket),
				Key:    obj.Key,
			})
			if err != nil {
				return copied, err
			}
			tmp, err := os.CreateTemp("", "groundseg-migrate-object-*")
			if err != nil {
				_ = getOut.Body.Close()
				return copied, err
			}
			written, copyErr := io.Copy(tmp, getOut.Body)
			_ = getOut.Body.Close()
			if copyErr != nil {
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
				return copied, copyErr
			}
			if _, err := tmp.Seek(0, io.SeekStart); err != nil {
				_ = tmp.Close()
				_ = os.Remove(tmp.Name())
				return copied, err
			}
			_, putErr := target.PutObject(context.Background(), &s3v2.PutObjectInput{
				Bucket:             awsv2.String(targetBucket),
				Key:                obj.Key,
				Body:               tmp,
				ContentLength:      awsv2.Int64(written),
				ContentType:        getOut.ContentType,
				ContentEncoding:    getOut.ContentEncoding,
				ContentDisposition: getOut.ContentDisposition,
				ContentLanguage:    getOut.ContentLanguage,
				CacheControl:       getOut.CacheControl,
				Metadata:           getOut.Metadata,
			})
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			if putErr != nil {
				return copied, putErr
			}
			copied++
		}
	}
	return copied, nil
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
	targetEndpoint, err := wireguardEndpoint(urbConf.WgS3Port)
	if err != nil {
		return err
	}
	targetClient, err := newS3Client(targetEndpoint, patp, pwd)
	if err != nil {
		return err
	}
	if err := waitForS3Ready(targetClient); err != nil {
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
	forceMigration := consumeForceLegacyMigrationForShip(patp)
	if forceMigration {
		zap.L().Info(fmt.Sprintf("Forcing legacy migration for %s (ignoring success marker)", patp))
	}

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

	if !forceMigration {
		markerStatus, err := readMigrationMarkerStatus(targetVolume, minioMigrationMarker)
		if err != nil {
			return err
		}
		switch markerStatus {
		case "ok":
			targetHasObjects, err := bucketHasObjects(targetClient, "bucket")
			if err == nil && targetHasObjects {
				return nil
			}
			zap.L().Info(fmt.Sprintf("Ignoring stale success marker for %s and retrying legacy migration", patp))
		case "target-populated", "target-bucket-populated":
			targetHasObjects, err := bucketHasObjects(targetClient, "bucket")
			if err == nil && targetHasObjects {
				return nil
			}
			zap.L().Info(fmt.Sprintf("Ignoring stale migration marker for %s and retrying legacy migration", patp))
		case "legacy-empty":
			if !consumeLegacyEmptyRetryForShip(patp) {
				return nil
			}
			zap.L().Info(fmt.Sprintf("Retrying legacy migration for %s (previous status: %s, once this run)", patp, markerStatus))
		case "legacy-no-bucket":
			zap.L().Info(fmt.Sprintf("Retrying legacy migration for %s (previous status: %s)", patp, markerStatus))
		}

		targetHasObjects, err := bucketHasObjects(targetClient, "bucket")
		if err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "bucket missing") {
				return err
			}
		}
		if targetHasObjects {
			return writeMigrationMarker(targetVolume, "target-bucket-populated")
		}
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

	sourceEndpoint, err := wireguardEndpoint(sourcePort)
	if err != nil {
		return err
	}
	sourceClient, err := newS3Client(sourceEndpoint, patp, sourceSecret)
	if err != nil {
		return err
	}
	if err := waitForS3Ready(sourceClient); err != nil {
		return err
	}

	sourceBuckets, err := listBucketNames(sourceClient)
	if err != nil {
		return fmt.Errorf("failed to list legacy buckets: %w", err)
	}
	if len(sourceBuckets) == 0 {
		return fmt.Errorf("legacy volume %s has no discoverable buckets", legacyVolume)
	}
	zap.L().Info(fmt.Sprintf("Legacy source buckets for %s: %v", patp, sourceBuckets))

	nonEmptySource := false
	for _, sourceBucket := range sourceBuckets {
		hasObjects, err := bucketHasObjects(sourceClient, sourceBucket)
		if err != nil {
			return err
		}
		if hasObjects {
			nonEmptySource = true
			break
		}
	}
	if !nonEmptySource {
		markLegacyEmptyRetryConsumed(patp)
		return writeMigrationMarker(targetVolume, "legacy-empty")
	}

	totalCopied := 0
	copyBucket := func(sourceBucket string, targetBucket string) error {
		if err := ensureBucketExists(targetClient, targetBucket); err != nil {
			return err
		}
		copied, err := migrateBucketObjects(sourceClient, targetClient, sourceBucket, targetBucket)
		if err != nil {
			return err
		}
		totalCopied += copied
		zap.L().Info(fmt.Sprintf("Legacy migration copied %d objects from %s to %s for %s", copied, sourceBucket, targetBucket, patp))
		return nil
	}

	primaryBucket := "bucket"
	if contains(sourceBuckets, primaryBucket) {
		if err := copyBucket(primaryBucket, primaryBucket); err != nil {
			return fmt.Errorf("failed to migrate legacy bucket %s: %w", primaryBucket, err)
		}
		for _, sourceBucket := range sourceBuckets {
			if sourceBucket == primaryBucket {
				continue
			}
			if err := copyBucket(sourceBucket, sourceBucket); err != nil {
				return fmt.Errorf("failed to migrate auxiliary legacy bucket %s: %w", sourceBucket, err)
			}
		}
	} else if len(sourceBuckets) == 1 {
		// Preserve existing URL shape by normalizing single legacy bucket into target "bucket".
		if err := copyBucket(sourceBuckets[0], primaryBucket); err != nil {
			return fmt.Errorf("failed to normalize legacy bucket %s into %s: %w", sourceBuckets[0], primaryBucket, err)
		}
	} else {
		for _, sourceBucket := range sourceBuckets {
			if err := copyBucket(sourceBucket, sourceBucket); err != nil {
				return fmt.Errorf("failed to migrate legacy bucket %s: %w", sourceBucket, err)
			}
		}
	}

	if totalCopied == 0 {
		return fmt.Errorf("legacy migration found non-empty source buckets but copied 0 objects for %s", patp)
	}
	return writeMigrationMarker(targetVolume, "ok")
}

func startLegacyMinIOMigrationSource(containerName, volumeName, accessKey, secretKey string, s3Port, consolePort int) error {
	_ = DeleteContainer(containerName)
	imageRef := legacyMinIOMigrationImageRef()
	if err := PullImageByRef(imageRef); err != nil {
		return err
	}

	cli, err := dockerclient.New()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()
	containerConfig := container.Config{
		Image:      imageRef,
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

func readMigrationMarkerStatus(volumeName, marker string) (string, error) {
	mountpoint, err := volumeMountpoint(volumeName)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(filepath.Join(mountpoint, marker))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	parts := strings.Fields(string(content))
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], nil
}

func writeMigrationMarker(volumeName, status string) error {
	content := fmt.Sprintf("%s %s\n", status, time.Now().UTC().Format(time.RFC3339))
	return WriteFileToVolume(volumeName, minioMigrationMarker, content)
}

func IsObjectStoreLinkConfigured(patp string) (bool, error) {
	volumeName := GetObjectStoreDataVolumeName(patp)
	exists, err := volumeExists(volumeName)
	if err != nil {
		return false, err
	}
	if !exists {
		return false, nil
	}
	return volumeHasMarker(volumeName, objectStoreLinkMarker)
}

func MarkObjectStoreLinkConfigured(patp string) error {
	volumeName := GetObjectStoreDataVolumeName(patp)
	exists, err := volumeExists(volumeName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	content := fmt.Sprintf("ok %s\n", time.Now().UTC().Format(time.RFC3339))
	return WriteFileToVolume(volumeName, objectStoreLinkMarker, content)
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
