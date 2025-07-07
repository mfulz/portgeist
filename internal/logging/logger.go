// Package logging provides centralized structured logging for Portgeist.
// It wraps zap.Logger and allows runtime-configurable level, output streams, and file logging.
package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config represents the logging configuration as defined in the global YAML config.
type Config struct {
	Level      string `yaml:"level" mapstructure:"level"`             // "debug", "info", "warn", "error"
	ToStdout   bool   `yaml:"to_stdout" mapstructure:"to_stdout"`     // Enable output to stdout
	ToStderr   bool   `yaml:"to_stderr" mapstructure:"to_stderr"`     // Enable output to stderr
	ToFile     bool   `yaml:"to_file" mapstructure:"to_file"`         // Enable output to file
	FilePath   string `yaml:"file" mapstructure:"file"`               // Log file path, e.g. /var/log/portgeist.log
	MaxSizeMB  int    `yaml:"max_size" mapstructure:"max_size"`       // Max size before rotation (in MB)
	MaxAge     int    `yaml:"max_age" mapstructure:"max_age"`         // Max age of logs (in days)
	MaxBackups int    `yaml:"max_backups" mapstructure:"max_backups"` // Number of rotated backups to keep
	Compress   bool   `yaml:"compress" mapstructure:"compress"`       // Gzip compress old log files
}

// Log is the globally accessible sugared logger instance.
var Log *zap.SugaredLogger

// Init initializes the global logger based on the provided config.
func Init(cfg Config) error {
	var cores []zapcore.Core
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderCfg)

	level := zapcore.InfoLevel
	_ = level.Set(cfg.Level) // falls ungültig: bleibt InfoLevel

	// Add stdout
	if cfg.ToStdout {
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, core)
	}

	// Add stderr
	if cfg.ToStderr {
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stderr), level)
		cores = append(cores, core)
	}

	// Add file logger
	if cfg.ToFile && cfg.FilePath != "" {
		writer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		})
		core := zapcore.NewCore(encoder, writer, level)
		cores = append(cores, core)
	}

	if len(cores) == 0 {
		// Fallback: always log to stdout
		core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level)
		cores = append(cores, core)
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller())
	Log = logger.Sugar()
	return nil
}
