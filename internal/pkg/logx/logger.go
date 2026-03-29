package logx

import (
	"fmt"
	"io"
	"os"
	"strings"

	klog "github.com/go-kratos/kratos/v2/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Options struct {
	Level      string `json:"level" yaml:"level"`
	Format     string `json:"format" yaml:"format"`
	Caller     bool   `json:"caller" yaml:"caller"`
	Filename   string `json:"filename" yaml:"filename"`
	MaxSizeMB  int    `json:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups int    `json:"max_backups" yaml:"max_backups"`
	MaxAgeDays int    `json:"max_age_days" yaml:"max_age_days"`
	Compress   bool   `json:"compress" yaml:"compress"`
	ToStdout   bool   `json:"to_stdout" yaml:"to_stdout"`
	LocalTime  bool   `json:"local_time" yaml:"local_time"`
}

// New creates a kratos logger from environment variables.
func New() klog.Logger {
	return NewWithOptions(Options{
		Level:      strings.TrimSpace(os.Getenv("LOG_LEVEL")),
		Format:     strings.TrimSpace(os.Getenv("LOG_FORMAT")),
		Caller:     parseBoolEnv("LOG_CALLER"),
		Filename:   strings.TrimSpace(os.Getenv("LOG_FILE")),
		MaxSizeMB:  parseIntEnv("LOG_MAX_SIZE_MB", 100),
		MaxBackups: parseIntEnv("LOG_MAX_BACKUPS", 7),
		MaxAgeDays: parseIntEnv("LOG_MAX_AGE_DAYS", 30),
		Compress:   parseBoolEnv("LOG_COMPRESS"),
		ToStdout:   parseBoolEnv("LOG_TO_STDOUT"),
		LocalTime:  parseBoolEnv("LOG_LOCAL_TIME"),
	})
}

func NewWithOptions(opts Options) klog.Logger {
	base := klog.NewStdLogger(buildWriter(opts))
	return klog.NewFilter(base, klog.FilterLevel(parseLevel(opts.Level)))
}

func parseLevel(v string) klog.Level {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return klog.LevelDebug
	case "warn", "warning":
		return klog.LevelWarn
	case "error":
		return klog.LevelError
	case "fatal":
		return klog.LevelFatal
	default:
		return klog.LevelInfo
	}
}

func parseBoolEnv(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "1", "t", "true", "yes", "y":
		return true
	default:
		return false
	}
}

func parseIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	var n int
	_, err := fmt.Sscanf(raw, "%d", &n)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func buildWriter(opts Options) io.Writer {
	filename := strings.TrimSpace(opts.Filename)
	if filename == "" {
		return os.Stdout
	}

	maxSize := opts.MaxSizeMB
	if maxSize <= 0 {
		maxSize = 100
	}
	maxBackups := opts.MaxBackups
	if maxBackups <= 0 {
		maxBackups = 7
	}
	maxAge := opts.MaxAgeDays
	if maxAge <= 0 {
		maxAge = 30
	}

	rotateWriter := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   opts.Compress,
		LocalTime:  opts.LocalTime,
	}

	if opts.ToStdout {
		return io.MultiWriter(os.Stdout, rotateWriter)
	}
	return rotateWriter
}
