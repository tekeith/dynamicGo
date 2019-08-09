package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var theLog *log.Logger

func infoL(msg string) {
	theLog.SetPrefix(time.Now().Format("2006-01-02T15:04:05.999 MST") + " [Info] ")
	theLog.Println(msg)
}
func warnL(msg string) {
	theLog.SetPrefix(time.Now().Format("2006-01-02T15:04:05.999 MST") + " [Warn] ")
	theLog.Println(msg)
}
func debugL(msg string) {
	theLog.SetPrefix(time.Now().Format("2006-01-02T15:04:05.999 MST") + " [Debug] ")
	theLog.Println(msg)
}
func errorL(msg string) {
	theLog.SetPrefix(time.Now().Format("2006-01-02T15:04:05.999 MST") + " [Error] ")
	theLog.Println(msg)
}
func setLogging(name string) {
	logName := name + ".log"
	logFile, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer logFile.Close()
	theLog.SetOutput(logFile)
	infoL(name + " starting up")
}

func exitHandler(w http.ResponseWriter, r *http.Request) {
	infoL("Exiting as requested")
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	fmt.Fprintf(w, "Exiting")
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, r.URL.String())
}

func main() {
	_, fil := filepath.Split(os.Args[0])
	name := strings.TrimSuffix(fil, filepath.Ext(fil))
	fmt.Println("Program name: " + name)
	setLogging(name)
	if len(os.Args) < 2 {
		fmt.Println("Port number must be provided as commandline argument.")
		os.Exit(1)
	}

	addr := "localhost:" + os.Args[1]

	http.HandleFunc("/exit", exitHandler)
	http.HandleFunc("/"+name, handler)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println(err)
	}

}
