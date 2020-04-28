package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"./source/llog"

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
	url       string // "http://thishost:8888"
	endpoint  string // "/agentname/"
	toAgent   io.WriteCloser
	fromAgent io.ReadCloser
	response  string
}

var agents = map[string]*urlAgent{}

func stopAgent(name string, agent *urlAgent) {
	agent.toAgent.Write([]byte("exit\n"))
	time.Sleep(2 * time.Second)
	fmt.Println(agent.response)
	if strings.Compare(agent.response, "exiting") == 0 {
		llog.Info(name + " stopped.")
		delete(agents, name)
	} else {
		llog.Error("Failed to stop " + name)
	}
}

func getAgent(agentName string) *urlAgent {
	agent := agents[agentName]
	if agent == nil {
		llog.Info("Call startAgent for ", agentName)
		return startAgent(agentName)
	}
	return agent
}
func listenAgent(agentName string) {
	for {
		agent := agents[agentName]
		if agent == nil {
			return
		}
		scanner := bufio.NewScanner(agent.fromAgent)
		if scanner.Scan() {
			resp := scanner.Text()
			if len(resp) > 0 {
				agent.response = resp
			}
		} else {
			err := scanner.Err()
			if err == nil {
				delete(agents, agentName)
			}
		}
	}
}

func startAgent(agentName string) *urlAgent {
	agentPath := agentDir + agentName
	agentPort := atomic.AddUint32(&port, 1)
	cmd := exec.Command(agentPath, fmt.Sprint(agentPort))
	toAgent, _ := cmd.StdinPipe()
	fromAgent, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		llog.Error("Error starting cmd: ", agentPath, ":", fmt.Sprint(agentPort))
		llog.Error(err)
		delete(agents, agentName)
		return nil
	}
	agent := new(urlAgent)
	agents[agentName] = agent
	agent.toAgent = toAgent
	agent.fromAgent = fromAgent
	agent.url = "http://localhost:" + fmt.Sprint(agentPort)
	shortName := strings.TrimSuffix(agentName, path.Ext(agentName))
	agent.endpoint = "/" + shortName
	aurl, err := url.Parse(agent.url)
	if err != nil {
		llog.Error("Error creating reverse proxy URL ", agent.endpoint, " -> ", agent.url, ", ", err)
		return nil
	}
	rprox := httputil.NewSingleHostReverseProxy(aurl)
	router.HandleFunc(agent.endpoint, handler(rprox))

	go listenAgent(agentName)
	llog.Info("Configured reverse proxy ", agent.endpoint, " -> ", agent.url)
	return agent
}

func stopAgents(w http.ResponseWriter, r *http.Request) {
	for len(agents) > 0 {
		for name, agent := range agents {
			stopAgent(name, agent)
		}
	}
	llog.Info("Exiting")
	fmt.Fprintf(w, "Exiting")
	os.Exit(0)
}

func XXcheckAgents() {
	for {
		files, _ := listAgents()
		for _, fil := range files {
			agent := getAgent(fil)
			if agent == nil {
				llog.Error("Unable to start ", fil)
			}
		}
		time.Sleep(20 * time.Second)
	}
}

func startAgents() {
	files, _ := listAgents()
	for _, fil := range files {
		agent := startAgent(fil)
		if agent == nil {
			llog.Error("Unable to start ", fil)
		}
	}
}

func listAgents() ([]string, error) {
	var files []string
	entries, err := ioutil.ReadDir(agentDir)
	if err != nil {
		llog.Error("Error getting ReadDir results: ", err)
		return files, err
	}

	for _, file := range entries {
		if !file.IsDir() {
			if isExecutable(agentDir + file.Name()) {
				files = append(files, file.Name())
			}
		}
	}
	return files, nil
}

func isExecutable(agentPath string) bool {
	_, err := exec.LookPath(agentPath)
	if err != nil {
		return false
	}
	return true
}

func main() {
	_, fil := filepath.Split(os.Args[0])
	name := strings.TrimSuffix(fil, filepath.Ext(fil))
	llog.SetFile("logs/" + name + ".log")

	//go startAgents()
	startAgents()

	router.HandleFunc("/exit", stopAgents)

	addr := "localhost:" + fmt.Sprint(myPort)
	llog.Info("Listen on addr: ", addr)
	err := http.ListenAndServe(addr, router)
	if err != nil {
		llog.Error(err)
	}
	llog.Info("Leaving main")
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		llog.Info(r.URL.String())
		p.ServeHTTP(w, r)
	}
}
