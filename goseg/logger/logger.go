package logger

import (
	"log/slog"
	"os"
)

var (
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)
