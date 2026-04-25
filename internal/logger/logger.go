package logger

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"traffic-monitor/internal/config"

	"github.com/rs/zerolog"
)

func NewLogger(cfg config.LogConfig) zerolog.Logger {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		baseDir := filepath.Base(filepath.Dir(file))
		return baseDir + "/" + filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	level, err := zerolog.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}

	logger := zerolog.New(os.Stdout).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	return logger
}
