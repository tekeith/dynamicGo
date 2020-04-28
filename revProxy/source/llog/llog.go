package llog

import (
	"io"
	"log"
	"os"
	"time"
)

var logFile *os.File
var theLog = log.New(os.Stderr, "", 0)

func prefix() string {
	return time.Now().Format("[2006-01-02T15:04:05.999 MST] ")
}

// Info   Log info messages
func Info(v ...interface{}) {
	theLog.Println(prefix(), "[Info] ", v)
}

// Warn   Log warning messages
func Warn(v ...interface{}) {
	theLog.Println(prefix(), "[Warn] ", v)
}

// Debug   Log debug messages
func Debug(v ...interface{}) {
	theLog.Println(prefix(), "[Debug] ", v)
}

// Error   Log error messages
func Error(v ...interface{}) {
	theLog.Println(prefix(), "[Error] ", v)
}

// Close   Close log file if not nil
func Close() {
	if logFile != nil {
		Info("Closing log file")
		logFile.Sync()
		logFile.Close()
	}
}

// SetFile   Set logging output to a file, name parameter will have .log added
func SetFile(name string) {
	var err error
	logFile, err = os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Error opening file %s: %v", name, err)
	}
	w := io.MultiWriter(os.Stderr, logFile)
	theLog.SetOutput(w)
}
