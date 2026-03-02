package routines

import (
	"context"

	"groundseg/routines/logstream"
)

func SysLogStreamer() {
	logstream.SysLogStreamer()
}

func SysLogStreamerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return logstream.SysLogStreamerWithContext(ctx)
}

func OldLogsCleaner() {
	logstream.OldLogsCleaner()
}

func OldLogsCleanerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return logstream.OldLogsCleanerWithContext(ctx)
}

func DockerLogStreamer() {
	logstream.DockerLogStreamer()
}

func DockerLogStreamerWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return logstream.DockerLogStreamerWithContext(ctx)
}

func DockerLogConnRemover() {
	logstream.DockerLogConnRemover()
}

func DockerLogConnRemoverWithContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return logstream.DockerLogConnRemoverWithContext(ctx)
}
