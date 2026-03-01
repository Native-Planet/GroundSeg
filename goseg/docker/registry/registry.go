package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"

	"groundseg/config"
	"groundseg/dockerclient"
)

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
	var res map[string]string
	res = make(map[string]string)
	if containerType == "llama-api" {
		res["tag"] = "dev"
		res["hash"] = "ac2dcfac72bc3d8ee51ee255edecc10072ef9c0f958120971c00be5f4944a6fa"
		res["repo"] = "nativeplanet/llama-gpt"
		return res, nil
	}

	arch := config.Architecture
	hashLabel := arch + "_sha256"
	versionInfo := config.GetVersionChannel()
	jsonData, err := json.Marshal(versionInfo)
	if err != nil {
		return res, err
	}

	var m map[string]interface{}
	err = json.Unmarshal(jsonData, &m)
	if err != nil {
		return res, err
	}

	containerData, ok := m[containerType].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%s data is not a map", containerType)
	}
	tag, ok := containerData["tag"].(string)
	if !ok {
		return nil, fmt.Errorf("'tag' is not a string")
	}
	hashValue, ok := containerData[hashLabel].(string)
	if !ok {
		return nil, fmt.Errorf("'%s' is not a string", hashLabel)
	}
	repo, ok := containerData["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("'repo' is not a string")
	}

	res = make(map[string]string)
	res["tag"] = tag
	res["hash"] = hashValue
	res["repo"] = repo
	res["type"] = containerType
	return res, nil
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
