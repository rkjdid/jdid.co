package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
)

var (
	port      = flag.Int("p", 8080, "listen port")
	logFile   = flag.String("log", "", "log.SetOutput")
	shareFile = flag.String("share", "", "what to serve under /share/, defaults to ~/share/ if unset")

	usr *user.User
	err error
)

func init() {
	flag.Parse()

	if usr, err = user.Current(); err != nil {
		log.Fatal(err)
	}

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(f)
	}

	if *shareFile == "" {
		*shareFile = path.Join(usr.HomeDir, "share")
	}
}

type LogServer struct {
	Name  string
	Inner http.Handler
}

func (ls *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var prefix string
	if ls.Name != "" {
		prefix = ls.Name + "> "
	}
	log.Printf("%sserving %s -> %s", prefix, r.RemoteAddr, r.URL)
	ls.Inner.ServeHTTP(w, r)
}

type WatServer struct{}

func (ws *WatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<html><head><title>what?</title></head><body>looking for <em>%s</em> ?</body>", r.URL.Path)
}

func main() {
	http.Handle("/share/", &LogServer{Name: "http.FileServer", Inner: http.FileServer(http.Dir(*shareFile))})
	http.Handle("/", &LogServer{Name: "wat", Inner: &WatServer{}})

	addr := fmt.Sprintf("localhost:%d", *port)
	log.Printf("Listening on %s...", addr)
	http.ListenAndServe(addr, nil)
}
