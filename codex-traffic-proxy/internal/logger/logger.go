package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"codex-traffic-proxy/internal/config"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
	config *config.Config
}

type LogEntry struct {
	Level     string                 `json:"level"`
	Timestamp string                 `json:"timestamp"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

func NewLogger(cfg *config.Config) *Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Logger.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set output
	var output io.Writer = os.Stdout
	switch strings.ToLower(cfg.Logger.Output) {
	case "stderr":
		output = os.Stderr
	case "stdout":
		output = os.Stdout
	}

	logger.SetOutput(output)

	// Set formatter
	switch strings.ToLower(cfg.Logger.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return &Logger{
		Logger: logger,
		config: cfg,
	}
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	if len(fields)%2 != 0 {
		fields = append(fields, "missing_value")
	}

	entry := l.Logger.WithFields(convertToFields(fields...))
	entry.Info(msg)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	if len(fields)%2 != 0 {
		fields = append(fields, "missing_value")
	}

	entry := l.Logger.WithFields(convertToFields(fields...))
	entry.Error(msg)
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	if len(fields)%2 != 0 {
		fields = append(fields, "missing_value")
	}

	entry := l.Logger.WithFields(convertToFields(fields...))
	entry.Debug(msg)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	if len(fields)%2 != 0 {
		fields = append(fields, "missing_value")
	}

	entry := l.Logger.WithFields(convertToFields(fields...))
	entry.Warn(msg)
}

func (l *Logger) LogRequest(reqInfo interface{}, fields ...interface{}) {
	allFields := append([]interface{}{"request", reqInfo}, fields...)
	l.Info("Request logged", allFields...)
}

func (l *Logger) LogResponse(reqInfo interface{}, fields ...interface{}) {
	allFields := append([]interface{}{"response", reqInfo}, fields...)
	l.Info("Response logged", allFields...)
}

func (l *Logger) LogSummary(summary interface{}, fields ...interface{}) {
	allFields := append([]interface{}{"summary", summary}, fields...)
	l.Info("Traffic summary", allFields...)
}

func convertToFields(args ...interface{}) logrus.Fields {
	fields := make(logrus.Fields)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			key := fmt.Sprintf("%v", args[i])
			fields[key] = args[i+1]
		}
	}
	return fields
}