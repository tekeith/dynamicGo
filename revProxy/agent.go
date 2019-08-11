package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"./llog"
)

func exitHandler(w http.ResponseWriter, r *http.Request) {
	llog.Linfo("Exiting as requested")
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	llog.Linfo("Exiting")
	fmt.Fprintf(w, "Exiting")
}

func handler(w http.ResponseWriter, r *http.Request) {
	llog.Linfo(r.URL.String())
	fmt.Fprint(w, r.URL.String())
}

func main() {
	_, fil := filepath.Split(os.Args[0])
	name := strings.TrimSuffix(fil, filepath.Ext(fil))
	llog.SetFile(name)
	llog.Linfo("Program name: ", name)
	if len(os.Args) < 2 {
		llog.Linfo("Port number must be provided as commandline argument.")
		os.Exit(1)
	}

	addr := "localhost:" + os.Args[1]

	http.HandleFunc("/exit", exitHandler)
	http.HandleFunc("/"+name, handler)

	llog.Linfo(name + " starting up.")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println(err)
	}

}
