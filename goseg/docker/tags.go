package docker

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const dockerHubTagCacheTTL = 30 * time.Minute

var (
	vereTagsMu       sync.Mutex
	vereTagsCache    []string
	vereTagsCacheErr error
	vereTagsCacheAt  time.Time
)

type dockerHubTagsResponse struct {
	Next    string `json:"next"`
	Results []struct {
		Name string `json:"name"`
	} `json:"results"`
}

func GetVereImageTags() ([]string, error) {
	vereTagsMu.Lock()
	defer vereTagsMu.Unlock()
	if time.Since(vereTagsCacheAt) < dockerHubTagCacheTTL {
		return append([]string{}, vereTagsCache...), vereTagsCacheErr
	}
	info, err := GetLatestContainerInfo("vere")
	if err != nil {
		vereTagsCacheErr = err
		vereTagsCacheAt = time.Now()
		return nil, err
	}
	tags, err := DockerHubTags(info["repo"])
	if err == nil {
		tags = appendUnique(tags, info["tag"])
		sort.Strings(tags)
	}
	vereTagsCache = append([]string{}, tags...)
	vereTagsCacheErr = err
	vereTagsCacheAt = time.Now()
	return tags, err
}

func DockerHubTags(repo string) ([]string, error) {
	path := dockerHubRepoPath(repo)
	if path == "" {
		return nil, fmt.Errorf("unsupported Docker Hub repo %q", repo)
	}
	endpoint := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?page_size=100", path)
	client := http.Client{Timeout: 15 * time.Second}
	seen := map[string]bool{}
	var tags []string
	for endpoint != "" && len(tags) < 500 {
		resp, err := client.Get(endpoint)
		if err != nil {
			return tags, err
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			statusCode := resp.StatusCode
			resp.Body.Close()
			return tags, fmt.Errorf("Docker Hub tags request failed: HTTP %d", statusCode)
		}
		var payload dockerHubTagsResponse
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			return tags, err
		}
		resp.Body.Close()
		for _, result := range payload.Results {
			tag := strings.TrimSpace(result.Name)
			if tag != "" && !seen[tag] {
				seen[tag] = true
				tags = append(tags, tag)
			}
		}
		endpoint = payload.Next
	}
	return tags, nil
}

func dockerHubRepoPath(repo string) string {
	repo = strings.TrimSpace(repo)
	for _, prefix := range []string{
		"registry.hub.docker.com/",
		"docker.io/",
		"index.docker.io/",
	} {
		repo = strings.TrimPrefix(repo, prefix)
	}
	repo = strings.TrimPrefix(repo, "library/")
	if strings.Count(repo, "/") != 1 {
		return ""
	}
	return url.PathEscape(strings.Split(repo, "/")[0]) + "/" + url.PathEscape(strings.Split(repo, "/")[1])
}
