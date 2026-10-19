// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"flag"
	"html/template"
	"io"
	"io/fs"
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
	socketPath = "/socket"

	// tourImportPath is the import path the exercises use for the
	// packages in the pic, reader, tree and wc directories.
	tourImportPath = "golang.org/x/tour"

	dirPerm  = 0o755
	filePerm = 0o644
)

// tourPackages holds the source of the packages that the exercises
// import from tourImportPath.
//
//go:embed pic/pic.go reader/validate.go tree/tree.go wc/wc.go
var tourPackages embed.FS

var (
	httpListen  = flag.String("http", "127.0.0.1:3999", "host:port que obrirem")
	openBrowser = flag.Bool("openbrowser", true, "obrir navegador automàticament")
)

var (
	// GOPATH containing the tour packages
	gopath string

	httpAddr string
)

// installTourPackages writes tourPackages into a GOPATH under the user's
// cache directory and returns that GOPATH. The local tour builds snippets in
// GOPATH mode against it, because they run in a temporary directory with no
// go.mod, where module mode can't resolve golang.org/x/tour imports.
func installTourPackages() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "go-tour-ca", "gopath")
	err = fs.WalkDir(tourPackages, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		src, err := tourPackages.ReadFile(name)
		if err != nil {
			return err
		}
		dst := filepath.Join(dir, "src", filepath.FromSlash(tourImportPath), filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
			return err
		}
		return os.WriteFile(dst, src, filePerm)
	})
	return dir, err
}

func main() {
	flag.Parse()

	if os.Getenv("GAE_ENV") == "standard" {
		log.Println("corrent a App Engine Standard")
		gaeMain()
		return
	}

	var err error
	gopath, err = installTourPackages()
	if err != nil {
		log.Fatalf("No s'han pogut instal·lar els paquets del tour: %v", err)
	}

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

	if err := initTour(tourFiles, "SocketTransport"); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/lesson/", lessonHandler)

	origin := &url.URL{Scheme: "http", Host: host + ":" + port}
	http.Handle(socketPath, socket.NewHandler(origin))

	registerStatic(tourFiles)

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

// registerStatic registers handlers to serve static content from fsys.
func registerStatic(fsys fs.FS) {
	// Keep these static file handlers in sync with app.yaml.
	img, err := fs.Sub(fsys, "static/img")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/favicon.ico", http.FileServer(http.FS(img)))
	static := http.FileServer(http.FS(fsys))
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
// replaced (or added) with the value of the global var gopath,
// and GOPATH mode turned on.
func environ() (env []string) {
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GOPATH=") && !strings.HasPrefix(v, "GO111MODULE=") {
			env = append(env, v)
		}
	}
	env = append(env, "GOPATH="+gopath, "GO111MODULE=off")
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
