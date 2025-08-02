package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Logger is the application logger
var Logger *logrus.Logger

// InitLogger initializes the logger with the given configuration
func InitLogger(level, format, logFile string) error {
	Logger = logrus.New()
	
	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	Logger.SetLevel(logLevel)
	
	// Set formatter
	switch format {
	case "json":
		Logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	default:
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}
	
	// Set output
	if logFile != "" {
		// Create directory if it doesn't exist
		dir := filepath.Dir(logFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		
		// Open log file
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		
		// Set output to both file and stdout
		Logger.SetOutput(io.MultiWriter(os.Stdout, file))
	} else {
		Logger.SetOutput(os.Stdout)
	}
	
	return nil
}

// GetLogger returns the logger instance
func GetLogger() *logrus.Logger {
	if Logger == nil {
		// Initialize with defaults if not already initialized
		InitLogger("info", "text", "")
	}
	return Logger
}

// WithFields returns a logger with the given fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetLogger().WithFields(fields)
}

// WithField returns a logger with the given field
func WithField(key string, value interface{}) *logrus.Entry {
	return GetLogger().WithField(key, value)
}

// Debug logs a debug message
func Debug(args ...interface{}) {
	GetLogger().Debug(args...)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	GetLogger().Debugf(format, args...)
}

// Info logs an info message
func Info(args ...interface{}) {
	GetLogger().Info(args...)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	GetLogger().Infof(format, args...)
}

// Warn logs a warning message
func Warn(args ...interface{}) {
	GetLogger().Warn(args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	GetLogger().Warnf(format, args...)
}

// Error logs an error message
func Error(args ...interface{}) {
	GetLogger().Error(args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	GetLogger().Errorf(format, args...)
}

// Fatal logs a fatal message and exits
func Fatal(args ...interface{}) {
	GetLogger().Fatal(args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	GetLogger().Fatalf(format, args...)
}

// Panic logs a panic message and panics
func Panic(args ...interface{}) {
	GetLogger().Panic(args...)
}

// Panicf logs a formatted panic message and panics
func Panicf(format string, args ...interface{}) {
	GetLogger().Panicf(format, args...)
} 