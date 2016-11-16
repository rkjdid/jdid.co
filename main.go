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

var port = flag.Int("p", 8080, "listen port")
var logFile = flag.String("log", "", "log.SetOutput")
var usr *user.User
var err error

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
}

func main() {
	var share = path.Join(usr.HomeDir, "share")
	http.Handle("/share/", http.FileServer(http.Dir(share)))
	log.Printf("/share/ -> %s", share)

	addr := fmt.Sprintf("localhost:%d", *port)
	log.Printf("Listening on %s...", addr)
	http.ListenAndServe(addr, nil)
}
