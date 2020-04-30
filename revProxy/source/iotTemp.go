package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"./llog"
)

func exitHandler(w http.ResponseWriter, r *http.Request) {
	llog.Info("Exiting as requested")
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	llog.Info("Exiting")
	fmt.Fprintf(w, "Exiting")
}

func handler(w http.ResponseWriter, r *http.Request) {
	llog.Info(r.URL.String())
	fmt.Fprint(w, r.URL.String())
}

func monitor() {
	var input string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input = scanner.Text()
		llog.Warn("Read from stdin: " + input)
		if len(input) > 0 {
			llog.Info("Read from stdin: " + input)
		}
		switch input {
		case "ping":
			fmt.Println("ok")
		case "exit":
			fmt.Println("exiting")
			time.Sleep(2 * time.Second)
			os.Exit(0)
		}
	}
	if err := scanner.Err(); err != nil {
		llog.Error("While reading standard input: ", err)
	}
}

func main() {
	_, fil := filepath.Split(os.Args[0])
	name := strings.TrimSuffix(fil, filepath.Ext(fil))
	llog.SetFile("logs/" + name + ".log")
	if len(os.Args) < 2 {
		llog.Error("Port number must be provided as commandline argument.")
		os.Exit(1)
	}

	go monitor()

	addr := "localhost:" + os.Args[1]

	llog.Info("Handle endpoint: /exit")
	http.HandleFunc("/exit", exitHandler)
	llog.Info("Handle endpoint: /" + name)
	http.HandleFunc("/"+name, handler)

	llog.Info(name + " listening for requests")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		llog.Error(err)
	}

}
