package lifecycle

import (
	"groundseg/docker/internal/closeutil"

	"github.com/docker/docker/client"
)

func closeRuntimeDockerClient(cli *client.Client, operation string, err *error) {
	closeutil.MergeCloseError(cli, operation, err)
}
