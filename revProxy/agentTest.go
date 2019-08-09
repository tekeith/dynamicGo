package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync/atomic"
	"time"

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

func setLogging(executableName string) {
	logName := executableName + ".log"
	logFile, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.Println(executableName, " starting up")
}

func stopAgents(w http.ResponseWriter, r *http.Request) {
	for nam, agent := range agents {
		if agent.running {
			fmt.Println("Stopping " + nam)
			url := agent.url + "exit"
			_, err := myClient.Get(url)
			if err != nil {
				fmt.Println("Error stopping ", nam)
				fmt.Println(err)
			}
		}
	}
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
	fmt.Fprintf(w, "Exiting")
}

func getAgent(agentName string) *urlAgent {
	agent := agents[agentName]
	if agent == nil {
		fmt.Println("Call startAgent for " + agentName)
		return startAgent(agentName)
	}
	return agent
}

func agentExists(agentPath string) int {
	_, err := exec.LookPath(agentPath)
	if err != nil {
		fmt.Println(agentPath, " was not found")
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
		fmt.Println("Error starting cmd: ", agentPath, ":", fmt.Sprint(agentPort))
		fmt.Println(err)
		agent.running = false
	}
	if agent.running {
		agent.url = "http://localhost:" + fmt.Sprint(agentPort)
		shortName := strings.TrimSuffix(agentName, path.Ext(agentName))
		agent.endpoint = "/" + shortName + "/"
		aurl, err := url.Parse(agent.url)
		if err != nil {
			fmt.Printf("Error creating reverse proxy URL %s -> %s, %v\n", agent.endpoint, agent.url, err)
		}
		rprox := httputil.NewSingleHostReverseProxy(aurl)
		router.HandleFunc("/forward/{rest:.*}", handler(rprox))

		fmt.Printf("Configured reverse proxy %s -> %s\n", agent.endpoint, agent.url)
		return agent
	}
	return nil
}

/*
   remote, err := url.Parse("http://localhost:8101")
   if err != nil {
           panic(err)
   }

   proxy := httputil.NewSingleHostReverseProxy(remote)
   http.HandleFunc("/", handler(proxy))
   info("Ready")
   error("I'm Listening!")
   err = http.ListenAndServe("localhost:8100", nil)
   if err != nil {
           panic(err)
   }

*/
/*
func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = mux.Vars(r)["rest"]
		p.ServeHTTP(w, r)
	}
}
*/
func listFiles() ([]string, error) {
	var files []string
	entries, err := ioutil.ReadDir(agentDir)
	if err != nil {
		fmt.Println("Error getting ReadDir results: ", err)
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
				fmt.Println("Unable to start ", fil)
			}
		}
		time.Sleep(20 * time.Second)
	}
}

/*
func main() {
	_, fil := filepath.Split(os.Args[0])
	name := strings.TrimSuffix(fil, filepath.Ext(fil))
	setLogging(name)
	go checkAgents()

	router.HandleFunc("/exit", stopAgents)

	addr := ":" + fmt.Sprint(myPort)
	fmt.Println("Listen on addr: ", addr)
	err := http.ListenAndServe(addr, router)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Leaving main")
}
*/

var theLog = log.New(os.Stdout, "", 0)

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

var addrs = []string{"http://localhost:8101", "http://localhost:8103"}
var endpoints = []string{"/agent", "/nother"}

func main() {
	for idx, addr := range addrs {
		remote, err := url.Parse(addr)
		if err != nil {
			panic(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(remote)
		endpoint := endpoints[idx]
		http.HandleFunc(endpoint, handler(proxy))
		info("Ready")
	}

	info("I'm Listening!")
	http.ListenAndServe("localhost:8100", nil)
	// if err != nil {
	// 	panic(err)
	// }
}

func handler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		info(r.URL.String())
		p.ServeHTTP(w, r)
	}
}
