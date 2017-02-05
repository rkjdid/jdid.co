package main

import (
	"flag"
	"fmt"
	"jdid.co/logger"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"
)

var (
	port        = flag.Int("p", 8080, "listen port")
	logFile     = flag.String("log", "", "log.SetOutput")
	sharePrefix = flag.String("share", "", "what to serve under /share/, defaults to ~/share/ if unset")

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

	if *sharePrefix == "" {
		*sharePrefix = path.Join(usr.HomeDir, "share")
	}
}

type WatServer struct{}

func (ws *WatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "<html><head><title>what?</title></head><body>looking for <em>%s</em> ?</body>", r.URL.Path)
}

func main() {
	http.Handle("/", &logger.LogServer{Name: "wat", Inner: &WatServer{}})
	http.Handle("/share/", &logger.LogServer{Name: " fs", Inner: http.FileServer(http.Dir(*sharePrefix))})

	addr := fmt.Sprintf("localhost:%d", *port)
	log.Printf("Listening on %s...", addr)
	log.Printf("/share/ -> %s", *sharePrefix)
	http.ListenAndServe(addr, nil)
}
