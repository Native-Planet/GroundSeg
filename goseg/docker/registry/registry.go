package registry

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"groundseg/config"
	"groundseg/dockerclient"
	"groundseg/structs"
	"groundseg/transition"
)

type imageMetadata struct {
	repo string
	tag  string
	hash string
}

type containerMetadataResolver struct {
	static  imageMetadata
	details fnChannelImageResolver
}

type fnChannelImageResolver func(structs.Channel) (structs.VersionDetails, bool)

var containerMetadataResolvers = map[transition.ContainerType]containerMetadataResolver{
	transition.ContainerTypeVere: {
		details: func(channel structs.Channel) (structs.VersionDetails, bool) {
			return channel.Vere, true
		},
	},
	transition.ContainerTypeNetdata: {
		details: func(channel structs.Channel) (structs.VersionDetails, bool) {
			return channel.Netdata, true
		},
	},
	transition.ContainerTypeMinio: {
		details: func(channel structs.Channel) (structs.VersionDetails, bool) {
			return channel.Minio, true
		},
	},
	transition.ContainerTypeMinioMC: {
		details: func(channel structs.Channel) (structs.VersionDetails, bool) {
			return channel.Miniomc, true
		},
	},
	transition.ContainerTypeWireguard: {
		details: func(channel structs.Channel) (structs.VersionDetails, bool) {
			return channel.Wireguard, true
		},
	},
	transition.ContainerTypeLlamaAPI: {
		static: imageMetadata{
			repo: "nativeplanet/llama-gpt",
			tag:  "dev",
			hash: "ac2dcfac72bc3d8ee51ee255edecc10072ef9c0f958120971c00be5f4944a6fa",
		},
	},
}

var dockerClientNew = dockerclient.New

func SetClientFactory(factory func(...client.Opt) (*client.Client, error)) {
	if factory == nil {
		dockerClientNew = dockerclient.New
		return
	}
	dockerClientNew = factory
}

// GetLatestContainerInfo returns image metadata for a specific container service.
func GetLatestContainerInfo(containerType string) (map[string]string, error) {
	resolvedType := transition.ContainerType(containerType)
	resolver, ok := containerMetadataResolvers[resolvedType]
	if !ok {
		return nil, fmt.Errorf("unsupported container type %q", containerType)
	}
	metadata, err := resolveContainerImageMetadata(resolvedType, resolver)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"tag":  metadata.tag,
		"hash": metadata.hash,
		"repo": metadata.repo,
		"type": containerType,
	}, nil
}

func resolveContainerImageMetadata(containerType transition.ContainerType, resolver containerMetadataResolver) (imageMetadata, error) {
	if resolver.static.repo != "" && resolver.static.tag != "" && resolver.static.hash != "" {
		return resolver.static, nil
	}

	versionInfo := config.GetVersionChannel()
	details, ok := resolver.details(versionInfo)
	if !ok {
		return imageMetadata{}, fmt.Errorf("%s metadata is unavailable", containerType)
	}
	hashLabel := strings.ToLower(config.Architecture()) + "_sha256"
	info := imageMetadata{
		repo: details.Repo,
		tag:  details.Tag,
	}
	switch hashLabel {
	case "amd64_sha256":
		info.hash = details.Amd64Sha256
	case "arm64_sha256":
		info.hash = details.Arm64Sha256
	default:
		return imageMetadata{}, fmt.Errorf("unsupported architecture %q", config.Architecture())
	}
	if info.repo == "" {
		return imageMetadata{}, fmt.Errorf("missing repo for %q", containerType)
	}
	if info.tag == "" {
		return imageMetadata{}, fmt.Errorf("missing tag for %q", containerType)
	}
	if info.hash == "" {
		return imageMetadata{}, fmt.Errorf("missing %s hash for %s", hashLabel, containerType)
	}
	return info, nil
}

// PullImageIfNotExist downloads an image if it is not already available locally.
func PullImageIfNotExist(desiredImage string, imageInfo map[string]string) (bool, error) {
	ctx := context.Background()
	cli, err := dockerClientNew()
	if err != nil {
		return false, err
	}
	defer cli.Close()

	images, err := cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, img := range images {
		for _, digest := range img.RepoDigests {
			if digest == fmt.Sprintf("%s@sha256:%s", imageInfo["repo"], imageInfo["hash"]) {
				return true, nil
			}
		}
	}
	resp, err := cli.ImagePull(ctx, fmt.Sprintf("%s@sha256:%s", imageInfo["repo"], imageInfo["hash"]), image.PullOptions{})
	if err != nil {
		return false, err
	}
	defer resp.Close()

	_, _ = io.Copy(io.Discard, resp)
	return true, nil
}

// LatestContainerImage builds the canonical image reference for a container type.
func LatestContainerImage(containerType string) (string, error) {
	containerInfo, err := GetLatestContainerInfo(containerType)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s@sha256:%s", containerInfo["repo"], containerInfo["tag"], containerInfo["hash"]), nil
}
