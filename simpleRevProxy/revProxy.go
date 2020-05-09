package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
)

var myPort int64 = 8100
var myHost, _ = os.Hostname()
var router = mux.NewRouter()
var server = &http.Server{Addr: ":" + strconv.FormatInt(myPort, 10), Handler: router}

//	rprox := httputil.NewSingleHostReverseProxy(aurl)
//	router.HandleFunc(agent.endpoint, handler(rprox))

func main() {
	
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println("Error from server: ", err)
	}
}
