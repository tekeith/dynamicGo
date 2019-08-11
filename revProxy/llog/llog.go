package llog

import (
	"log"
	"os"
	"time"
)

var theLog = log.New(os.Stdout, "", 0)

// Linfo   Log info messages
func Linfo(v ...interface{}) {
	theLog.Println(time.Now().Format("2006-01-02T15:04:05.999 MST"), " [Info] ", v)
}

// Lwarn   Log warning messages
func Lwarn(v ...interface{}) {
	theLog.Println(time.Now().Format("2006-01-02T15:04:05.999 MST"), " [Warn] ", v)
}

// Ldebug   Log debug messages
func Ldebug(v ...interface{}) {
	theLog.Println(time.Now().Format("2006-01-02T15:04:05.999 MST"), " [Debug] ", v)
}

// Lerror   Log error messages
func Lerror(v ...interface{}) {
	theLog.Println(time.Now().Format("2006-01-02T15:04:05.999 MST"), " [Error] ", v)
}

// SetFile   Set logging output to a file, name parameter will have .log added
func SetFile(name string) {
	logName := name + ".log"
	logFile, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer logFile.Close()
	theLog = log.New(logFile, "", 0)
}
