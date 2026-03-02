package lifecycle

import (
	"strings"

	"github.com/docker/docker/api/types/container"
)

// ContainerStatusIndex provides fast lookup of container status by canonical and absolute names.
type ContainerStatusIndex map[string]string

// NewContainerStatusIndex builds a name-index from a Docker container summary set.
func NewContainerStatusIndex(containers []container.Summary) ContainerStatusIndex {
	index := make(ContainerStatusIndex, len(containers))
	for _, containerSummary := range containers {
		for _, name := range containerSummary.Names {
			trimmed := strings.TrimSpace(name)
			index[trimmed] = containerSummary.Status
			index[strings.TrimPrefix(trimmed, "/")] = containerSummary.Status
		}
	}
	return index
}

// ResolveStatuses returns statuses for a set of query names using an index.
func ResolveStatuses(index ContainerStatusIndex, names []string) map[string]string {
	results := make(map[string]string, len(names))
	for _, name := range names {
		if status, ok := index[name]; ok {
			results[name] = status
			continue
		}
		results[name] = "not found"
		if status, ok := index["/"+name]; ok {
			results[name] = status
		}
	}
	return results
}
