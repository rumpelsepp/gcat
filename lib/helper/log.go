package helper

import (
	"log/slog"
	"os"
)

func GetLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, nil))
}
