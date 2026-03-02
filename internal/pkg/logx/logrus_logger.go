package logx

import (
	"fmt"
	"io"
	"os"
	"strings"

	klog "github.com/go-kratos/kratos/v2/log"
	"github.com/sirupsen/logrus"
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

// New creates a logrus logger from environment variables.
func New() *logrus.Logger {
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

func NewWithOptions(opts Options) *logrus.Logger {
	l := logrus.New()
	l.SetOutput(buildWriter(opts))

	switch strings.ToLower(strings.TrimSpace(opts.Format)) {
	case "text":
		l.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	default:
		l.SetFormatter(&logrus.JSONFormatter{})
	}

	lvl := strings.ToLower(strings.TrimSpace(opts.Level))
	if lvl == "" {
		lvl = "info"
	}
	if parsed, err := logrus.ParseLevel(lvl); err == nil {
		l.SetLevel(parsed)
	} else {
		l.SetLevel(logrus.InfoLevel)
	}

	l.SetReportCaller(opts.Caller)
	return l
}

func NewEntry(logger *logrus.Logger, fields logrus.Fields) *logrus.Entry {
	if logger == nil {
		logger = logrus.New()
	}
	if fields == nil {
		return logrus.NewEntry(logger)
	}
	return logger.WithFields(fields)
}

func NewKratosLogger(entry *logrus.Entry) klog.Logger {
	if entry == nil {
		entry = logrus.NewEntry(logrus.New())
	}
	return &kratosLogrusLogger{entry: entry}
}

type kratosLogrusLogger struct {
	entry *logrus.Entry
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

func (l *kratosLogrusLogger) Log(level klog.Level, keyvals ...interface{}) error {
	fields := logrus.Fields{}
	msg := "kratos log"

	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprintf("arg_%d", i)
		if k, ok := keyvals[i].(string); ok && k != "" {
			key = k
		}
		if i+1 >= len(keyvals) {
			fields[key] = nil
			continue
		}
		val := keyvals[i+1]
		if key == "msg" {
			msg = fmt.Sprintf("%v", val)
			continue
		}
		fields[key] = val
	}

	e := l.entry.WithFields(fields)
	switch level {
	case klog.LevelDebug:
		e.Debug(msg)
	case klog.LevelInfo:
		e.Info(msg)
	case klog.LevelWarn:
		e.Warn(msg)
	case klog.LevelError:
		e.Error(msg)
	case klog.LevelFatal:
		e.Fatal(msg)
	default:
		e.Info(msg)
	}
	return nil
}
