package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"time"

	"./llog"
	"github.com/gorilla/mux"
)

var agentDir = "agents/"
var myPort uint32 = 8100
var port = myPort
var myHost, _ = os.Hostname()
var myClient = &http.Client{
	Timeout: time.Second * 10,
}
var router = mux.NewRouter()

type urlAgent struct {
	url      string // "http://thishost:8888"
	endpoint string // "/agentname/"
	running  bool
}

var agents = map[string]*urlAgent{}

func stopAgents(w http.ResponseWriter, r *http.Request) {
	for nam, agent := range agents {
		if agent.running {
			llog.Linfo("Stopping " + nam)
			url := agent.url + "/exit"
			_, err := myClient.Get(url)
			if err != nil {
				llog.Lerror("Error stopping ", nam)
				llog.Lerror(err)
			}
		}
	}
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	llog.Linfo("Exiting")
	fmt.Fprint(w, "Exiting")
}

func getAgent(agentName string) *urlAgent {
	agent := agents[agentName]
	if agent == nil {
		llog.Linfo("Call startAgent for ", agentName)
		return startAgent(agentName)
	}
	return agent
}

func agentExists(agentPath string) int {
	_, err := exec.LookPath(agentPath)
	if err != nil {
		llog.Lwarn(agentPath, " was not found")
		return 1
	}
	return 0
}

func startAgent(agentName string) *urlAgent {
	agentPath := agentDir + agentName

	agentPort := atomic.AddUint32(&port, 1)
	cmd := exec.Command(agentPath, fmt.Sprint(agentPort))
	err := cmd.Start()
	agent := new(urlAgent)
	agent.running = true
	agents[agentName] = agent
	if err != nil {
		llog.Lerror("Error starting cmd: ", agentPath, ":", fmt.Sprint(agentPort))
		llog.Lerror(err)
		agent.running = false
	}
	if agent.running {
		agent.url = "http://localhost:" + fmt.Sprint(agentPort)
		shortName := strings.TrimSuffix(agentName, path.Ext(agentName))
		agent.endpoint = "/" + shortName
		aurl, err := url.Parse(agent.url)
		if err != nil {
			llog.Lerror("Error creating reverse proxy URL ", agent.endpoint, " -> ", agent.url, ", ", err)
		}
		rprox := httputil.NewSingleHostReverseProxy(aurl)
		router.HandleFunc(agent.endpoint, handler(rprox))

		llog.Linfo("Configured reverse proxy ", agent.endpoint, " -> ", agent.url)
		return agent
	}
	return nil
}

func listFiles() ([]string, error) {
	var files []string
	entries, err := ioutil.ReadDir(agentDir)
	if err != nil {
		llog.Lerror("Error getting ReadDir results: ", err)
		return files, err
	}

	for _, file := range entries {
		if !file.IsDir() {
			if agentExists(agentDir+file.Name()) == 0 {
				files = append(files, file.Name())
			}
		}
	}
	return files, nil
}

func checkAgents() {
	for {
		files, _ := listFiles()
		for _, fil := range files {
			agent := getAgent(fil)
			if agent == nil {
				llog.Lerror("Unable to start ", fil)
			}
		}
		time.Sleep(20 * time.Second)
	}
}

func main() {
	//_, fil := filepath.Split(os.Args[0])
	//name := strings.TrimSuffix(fil, filepath.Ext(fil))
	//llog.SetFile(name)
	go checkAgents()

	router.HandleFunc("/exit", stopAgents)

	addr := "localhost:" + fmt.Sprint(myPort)
	llog.Linfo("Listen on addr: ", addr)
	err := http.ListenAndServe(addr, router)
	if err != nil {
		llog.Lerror(err)
	}
	llog.Linfo("Leaving main")
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		llog.Linfo(r.URL.String())
		p.ServeHTTP(w, r)
	}
}
