// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"go/build"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/tools/playground/socket"

	// Imports so that go build/install automatically installs them.
	_ "golang.org/x/tour/pic"
	_ "golang.org/x/tour/tree"
	_ "golang.org/x/tour/wc"
)

const (
	basePkg    = "github.com/zorion/go-tour-ca"
	socketPath = "/socket"
)

var (
	httpListen  = flag.String("http", "127.0.0.1:3999", "host:port que obrirem")
	openBrowser = flag.Bool("openbrowser", true, "obrir navegador automàticament")
)

var (
	// GOPATH containing the tour packages
	gopath = os.Getenv("GOPATH")

	httpAddr string
)

// isRoot reports whether path is the root directory of the tour tree.
// To be the root, it must have content and template subdirectories.
func isRoot(path string) bool {
	_, err := os.Stat(filepath.Join(path, "content", "welcome.article"))
	if err == nil {
		_, err = os.Stat(filepath.Join(path, "template", "index.tmpl"))
	}
	return err == nil
}

func findRoot() (string, error) {
	ctx := build.Default
	p, err := ctx.Import(basePkg, "", build.FindOnly)
	if err == nil && isRoot(p.Dir) {
		return p.Dir, nil
	}
	tourRoot := filepath.Join(runtime.GOROOT(), "misc", "tour")
	ctx.GOPATH = tourRoot
	p, err = ctx.Import(basePkg, "", build.FindOnly)
	if err == nil && isRoot(tourRoot) {
		gopath = tourRoot
		return tourRoot, nil
	}
	return "", fmt.Errorf("No s'ha trobat contingut de go-tour-ca; mireu $GOROOT i $GOPATH")
}

func main() {
	flag.Parse()

	if os.Getenv("GAE_ENV") == "standard" {
		log.Println("corrent a App Engine Standard")
		gaeMain()
		return
	}

	// find and serve the go tour files
	root, err := findRoot()
	if err != nil {
		log.Fatalf("No s'ha trobat el contingut del tour: %v", err)
	}

	log.Println("Servint contingut des de", root)

	host, port, err := net.SplitHostPort(*httpListen)
	if err != nil {
		log.Fatal(err)
	}
	if host == "" {
		host = "localhost"
	}
	if host != "127.0.0.1" && host != "localhost" {
		log.Print(localhostWarning)
	}
	httpAddr = host + ":" + port

	if err := initTour(root, "SocketTransport"); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/lesson/", lessonHandler)

	origin := &url.URL{Scheme: "http", Host: host + ":" + port}
	http.Handle(socketPath, socket.NewHandler(origin))

	registerStatic(root)

	go func() {
		url := "http://" + httpAddr
		if waitServer(url) && *openBrowser && startBrowser(url) {
			log.Printf("S'hauria d'haver obert una finestra de navegador. Si no és així, si us plau, visiteu %s", url)
		} else {
			log.Printf("Si us plau, obriu el navegador i visiteu %s", url)
		}
	}()
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

// registerStatic registers handlers to serve static content
// from the directory root.
func registerStatic(root string) {
	// Keep these static file handlers in sync with app.yaml.
	http.Handle("/favicon.ico", http.FileServer(http.Dir(filepath.Join(root, "static", "img"))))
	static := http.FileServer(http.Dir(root))
	http.Handle("/content/img/", static)
	http.Handle("/static/", static)
}

// rootHandler returns a handler for all the requests except the ones for lessons.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if err := renderUI(w); err != nil {
		log.Println(err)
	}
}

// lessonHandler handler the HTTP requests for lessons.
func lessonHandler(w http.ResponseWriter, r *http.Request) {
	lesson := strings.TrimPrefix(r.URL.Path, "/lesson/")
	if err := writeLesson(lesson, w); err != nil {
		if err == lessonNotFound {
			http.NotFound(w, r)
		} else {
			log.Println(err)
		}
	}
}

const localhostWarning = `
PERILL!  PERILL!  PERILL!

El servidor del tour sembla estar servint el contingut des
d'una adreça que no és localhost i està configurat per
executar localment els fragments de codi.
Qualsevol amb accès a aquesta adreça i port podria guanyar
accès a aquest màquina com a l'usuari que executa go-tour-ca.

Si no enteneu aquest missatge, premeu Contro-C per abortar el procès.

PERILL!  PERILL!  PERILL!
`

type response struct {
	Output string `json:"output"`
	Errors string `json:"compile_errors"`
}

func init() {
	socket.Environ = environ
}

// environ returns the original execution environment with GOPATH
// replaced (or added) with the value of the global var gopath.
func environ() (env []string) {
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GOPATH=") {
			env = append(env, v)
		}
	}
	env = append(env, "GOPATH="+gopath)
	return
}

// waitServer waits some time for the http Server to start
// serving url. The return value reports whether it starts.
func waitServer(url string) bool {
	tries := 20
	for tries > 0 {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
		tries--
	}
	return false
}

// startBrowser tries to open the URL in a browser, and returns
// whether it succeed.
func startBrowser(url string) bool {
	// try to start the browser
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open"}
	case "windows":
		args = []string{"cmd", "/c", "start"}
	default:
		args = []string{"xdg-open"}
	}
	cmd := exec.Command(args[0], append(args[1:], url)...)
	return cmd.Start() == nil
}

// prepContent for the local tour simply returns the content as-is.
var prepContent = func(r io.Reader) io.Reader { return r }

// socketAddr returns the WebSocket handler address.
var socketAddr = func() string { return "ws://" + httpAddr + socketPath }

// analyticsHTML is optional analytics HTML to insert at the beginning of <head>.
var analyticsHTML template.HTML
