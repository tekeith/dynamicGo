package main

import (
	"bufio"
	"context"
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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"./source/llog"
	"github.com/gorilla/mux"
)

var agentDir = "agents/"
var myPort int64 = 8100
var port = myPort
var myHost, _ = os.Hostname()
var myClient = &http.Client{
	Timeout: time.Second * 10,
}
var router = mux.NewRouter()
var server = &http.Server{Addr: ":" + strconv.FormatInt(myPort, 10), Handler: router}
var waitGrp sync.WaitGroup
var mutex sync.Mutex
var runningFlag = true

type urlAgent struct {
	url       string // "http://thishost:8888"
	endpoint  string // "/agentname/"
	toAgent   io.WriteCloser
	fromAgent io.ReadCloser
	response  string
	agentFile string
}

var agents = map[string]*urlAgent{}

func shutDown() {
	mutex.Lock()
	runningFlag = false
	mutex.Unlock()
}

func running() bool {
	var result bool
	mutex.Lock()
	result = runningFlag
	mutex.Unlock()
	return result
}

func checkAgent(name string, agent *urlAgent) {
	waitGrp.Add(1)
	defer waitGrp.Done()

	agent.response = ""
	agent.toAgent.Write([]byte("ping\n"))
	time.Sleep(2 * time.Second)
	if len(agent.response) > 0 {
		// if strings.Compare(agent.response, "ok") == 0 {
		// 	llog.Info(name, " running.")
		// } else {
		// 	llog.Error(name, " responded with: ", agent.response)
		// }
	} else {
		llog.Error(name, " did not respond, will try to start it.")
		delete(agents, name)
		startAgent(name)
	}
}

func stopAgent(name string, agent *urlAgent) {
	waitGrp.Add(1)
	defer waitGrp.Done()

	llog.Info("Stop agent: ", name)
	agent.response = ""
	agent.toAgent.Write([]byte("exit\n"))
	time.Sleep(2 * time.Second)
	if len(agent.response) > 0 {
		llog.Info("Response from ", name, "is: ", agent.response)
		if strings.Compare(agent.response, "exiting") == 0 {
			llog.Info(name, " stopped.")
			delete(agents, name)
		} else {
			llog.Error("Failed to stop ", name)
		}
	} else {
		llog.Error("No response from ", name, " assuming it has exited.")
		delete(agents, name)
	}
}

func listenAgent(agentName string) {
	waitGrp.Add(1)
	defer waitGrp.Done()

	agent := agents[agentName]
	if agent == nil {
		return
	}
	scanner := bufio.NewScanner(agent.fromAgent)
	for scanner.Scan() {
		resp := scanner.Text()
		// llog.Warn("Read from " + agentName + ": " + resp)
		if len(resp) > 0 {
			agent.response = resp
		}
	}
	if err := scanner.Err(); err != nil {
		delete(agents, agentName)
	}
}

func startAgent(agentName string) *urlAgent {
	if !running() {
		return nil
	}
	agentPath := agentDir + agentName
	agentPort := atomic.AddInt64(&port, 1)
	cmd := exec.Command(agentPath, fmt.Sprint(agentPort))
	toAgent, _ := cmd.StdinPipe()
	fromAgent, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		llog.Error("Error starting cmd: ", agentPath, ":", fmt.Sprint(agentPort))
		llog.Error("Error: ", err)
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
	shutDown()
	go shutdownAgents()
	go stopServer()
	llog.Info("Shutting down agents.")
	w.Write([]byte("Exiting"))
}

func stopServer() {
	waitGrp.Add(1)
	defer waitGrp.Done()

	time.Sleep(5 * time.Second)
	llog.Info("Initiate server shutdown.")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := server.Shutdown(ctx); err != nil {
		llog.Error("Server Shutdown Failed: ", err)
	}
	llog.Info("Server Exited Properly")
}

func shutdownAgents() {
	waitGrp.Add(1)
	defer waitGrp.Done()

	for len(agents) > 0 {
		for name, agent := range agents {
			go stopAgent(name, agent)
		}
		time.Sleep(10 * time.Second)
	}
}

func checkAgents() {
	// need to get list of agent files,
	// use it with the list of running agents to:
	// start agents for files that have been added
	// stop any agents whose agent files have been removed

	agentFiles, _ := listAgents()     // list of agent files
	for _, name := range agentFiles { // start any agent not in the list
		agent := agents[name]
		if agent == nil {
			startAgent(name)
		}
	}

	// now we need to check to make each agent is still in list of files
	for name, agent := range agents {
		if !contains(agentFiles, name) {
			go stopAgent(name, agent)
		} else {
			// it is in the list, make sure it is running
			go checkAgent(name, agent)
		}
	}
}

func contains(alist []string, name string) bool {
	for _, itm := range alist {
		if itm == name {
			return true
		}
	}
	return false
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

func refreshAgents() {
	waitGrp.Add(1)
	defer waitGrp.Done()

	cnt := 1
	for running() {
		if cnt > 6 {
			cnt = 1
			checkAgents()
		} else {
			cnt = cnt + 1
		}
		time.Sleep(10 * time.Second)
	}
}

func main() {
	_, fil := filepath.Split(os.Args[0])
	name := strings.TrimSuffix(fil, filepath.Ext(fil))
	llog.SetFile("logs/" + name + ".log")

	startAgents()

	go refreshAgents()

	router.HandleFunc("/exit", stopAgents)

	addr := "localhost:" + fmt.Sprint(myPort)
	llog.Info("Listen on addr: ", addr)

	err := server.ListenAndServe()
	if err != nil {
		llog.Error("Error from server: ", err)
	}

	waitGrp.Wait()
	llog.Info(name, " exiting.")
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		llog.Info(r.URL.String())
		p.ServeHTTP(w, r)
	}
}
