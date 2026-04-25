package logger

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

	writer := buildWriter(cfg)

	logger := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Caller().
		Logger()

	return logger
}

func buildWriter(cfg config.LogConfig) zerolog.LevelWriter {
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "", "json":
		return zerolog.MultiLevelWriter(os.Stdout)
	case "console":
		return zerolog.MultiLevelWriter(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	default:
		return zerolog.MultiLevelWriter(os.Stdout)
	}
}
