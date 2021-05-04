package logging

import (
	"io"
	"log"
)

// Logger ...
type Logger struct {
	DebugLogger    *log.Logger
	InfoLogger     *log.Logger
	WarningLogger  *log.Logger
	ErrorLogger    *log.Logger
	CriticalLogger *log.Logger
}

// Init ...
func (logger *Logger) Init(writers ...io.Writer) {
	mw := io.MultiWriter(writers...)
	logger.DebugLogger = log.New(mw, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.InfoLogger = log.New(mw, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.WarningLogger = log.New(mw, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.ErrorLogger = log.New(mw, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	logger.CriticalLogger = log.New(mw, "CRITICAL: ", log.Ldate|log.Ltime|log.Lshortfile)

}

// Debug ...
func (logger *Logger) Debug(msg string) {
	logger.DebugLogger.Println(msg)
}

// Info ...
func (logger *Logger) Info(msg string) {
	logger.InfoLogger.Println(msg)
}

// Warning ...
func (logger *Logger) Warning(msg string) {
	logger.WarningLogger.Println(msg)
}

// Error ...
func (logger *Logger) Error(msg string) {
	logger.ErrorLogger.Println(msg)
}

// Critical ...
func (logger *Logger) Critical(msg string) {
	logger.CriticalLogger.Println(msg)
}
