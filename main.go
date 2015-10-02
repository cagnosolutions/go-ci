package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"strings"
)

type M map[string]interface{}

func (m M) find(ss ...string) interface{} {
	return match(m, 0, ss...)
}

func match(m map[string]interface{}, n int, ss ...string) interface{} {
	if len(ss) == 0 || ss[0] == "" {
		return m
	}
	if v, ok := m[ss[n]]; ok {
		if reflect.TypeOf(v).Kind() == reflect.Map && n < len(ss)-1 {
			n++
			return match(v.(map[string]interface{}), n, ss...)
		}
		return v
	}
	return nil
}

type CIServer struct {
	http.Handler
}

func (c *CIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// general http log
	log.Printf("Received an HTTP %q request from %q\n", r.Method, r.RemoteAddr)
	if r.Method != "POST" {
		log.Println("r.Method != POST")
		return
	}
	if r.Header.Get("X-GitHub-Event") != "push" {
		log.Printf("X-GitHub-Event != push (%s)\n", r.Header.Get("X-GitHub-Event"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("User-Agent"), "GitHub-Hookshot/") {
		log.Printf("User-Agent !startWith(GitHub-Hookshot/) (%s)\n", r.Header.Get("User-Agent"))
		return
	}
	// valid method and sender, parse body
	log.Println("Valid r.Method and sender, attempting decode/parse body...")
	dec := json.NewDecoder(r.Body)
	var m M
	if err := dec.Decode(&m); err != nil {
		log.Fatal("CIServer.ServeHTTP() -> json decode: ", err)
	}
	// get repository name from map
	log.Println("Successfully decoded body, get repository name from map...")
	repo := m.find("repository", "full_name")
	if repo == nil {
		log.Fatal("CIServer.ServeHTTP() -> M.Find(): repository is nil")
	}
	log.Println("Repository name is: ", repo.(string))
	// log that we got a repo push
	log.Printf("Received PUSH from GitHub for repoistory %q\n", repo.(string))
	// check if repo exists locally
	info, err := os.Stat(repo.(string))
	if err != nil && os.IsNotExist(err) {
		// repo not local, need to issue go get
		Exec("go", "get", "github.com/"+repo.(string))
	}
	if info.IsDir() {
		// repo exists pull
		Exec("git", "-C", repo.(string), "pull")
	}
	Exec("go", "build", "-o", repo.(string)+"/main", repo.(string)+"/*.go")
}

func Exec(cmd string, args ...string) string {
	b, err := exec.Command(cmd, args...).Output()
	if err != nil {
		log.Fatal("Exec(): ", err)
	}
	return string(b)
}

func main() {
	if err := http.ListenAndServe(":9999", &CIServer{}); err != nil {
		log.Fatal("main() -> http.ListenAndServe(): ", err)
	}
}
