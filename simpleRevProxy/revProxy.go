package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

var name string
var myPort int
var myHost, _ = os.Hostname()
var router = mux.NewRouter()
var server *http.Server

func notFound(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, r.URL.String(), " was not found by ", name)
}

func setupDefault(purl string) {
	fmt.Println("Setup default proxy at ", purl)
	aurl, err := url.Parse(purl)
	if err != nil {
		fmt.Println("Failed to parse ", purl, " as a url: ", err)
		os.Exit(1)
	}
	rprox := httputil.NewSingleHostReverseProxy(aurl)
	router.NotFoundHandler = rprox
}

func setupProxy(path string, purl string) {
	fmt.Println("Setup proxy for ", path, " at ", purl)
	aurl, err := url.Parse(purl)
	if err != nil {
		fmt.Println("Failed to parse ", purl, " as a url: ", err)
		os.Exit(1)
	}
	rprox := httputil.NewSingleHostReverseProxy(aurl)
	router.PathPrefix(path).Handler(rprox)
}

// main should be invoked with arguments:
// Port to use:  8100
// endpoint=url
// Multiple endpoint=url arguments may be used
// example:
//   revProxy.exe 8100 /api=http://localhost:53313 default=http://localhost:4200
func main() {

	cnt := len(os.Args)
	if cnt < 4 {
		usage(name, cnt)
	}

	for idx, arg := range os.Args {
		switch idx {
		case 0:
			_, fil := filepath.Split(arg)
			name = strings.TrimSuffix(fil, filepath.Ext(fil))
		case 1:
			port, err := strconv.Atoi(arg)
			if err != nil {
				fmt.Println("Error processing args, argument: ", arg, " should be an integer port number.")
				usage(name, cnt)
			}
			myPort = port
			server = &http.Server{Addr: ":" + fmt.Sprint(myPort), Handler: router}
		default:
			// should be path=url
			parts := strings.Split(arg, "=")
			path := parts[0]
			url := parts[1]
			if path == "default" {
				setupDefault(url)
			} else {
				setupProxy(path, url)
			}
		}
	}

	fmt.Println(name, " listening on port ", myPort)
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error from server: ", err)
	}
}

func usage(name string, cnt int) {
	str := "argument"
	if cnt > 1 {
		str = str + "s"
	}
	fmt.Println(name + " invoked with " + fmt.Sprint(cnt) + " " + str + ", requires at least 4. Example:")
	fmt.Println("    " + name + " 8100 default=http://localhost:4200 /api=http://localhost:53313 ")
	fmt.Println("    " + name + " port_to_listen_on default=url path_1=url ... ")
	os.Exit(1)
}
