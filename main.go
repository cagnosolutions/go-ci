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
	if r.Method != "POST" {
		return
	}
	if r.Header.Get("X-GitHub-Event") != "push" {
		return
	}
	if strings.HasPrefix(r.Header.Get("User-Agent"), "GitHub-Hookshot/") {
		return
	}
	// valid method and sender, parse body
	dec := json.NewDecoder(r.Body)
	var m M
	if err := dec.Decode(&m); err != nil {
		log.Fatal("CIServer.ServeHTTP() -> json decode: ", err)
	}
	// get repository name from map
	repo := m.find("repository", "full_name")
	if repo == nil {
		log.Fatal("CIServer.ServeHTTP() -> M.Find(): repository is nil")
	}
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
