package main

import (
	"flag"
	"fmt"
	"jdid.co/xhttp"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

	"github.com/gorilla/mux"
)

const (
	flagShare = "share"
	flagCss   = "css"
	flagJs    = "js"
	flagImg   = "img"
)

// default file servers, if not handled by some reverse proxy.. useful for localhosting.
// Use w/ -debug [-<flagName>...]
var fileServers []string = []string{flagShare, flagCss, flagJs, flagImg}

var (
	port       = flag.Int("p", 8080, "listen port")
	logFile    = flag.String("log", "", "log.SetOutput")
	rootPrefix = flag.String("root", "~", "root path of project for resources (share, css, js, img...)")
	htmlRoot   = flag.String("html", "", "path to html templates, defaults to <root>/html/")
	debug      = flag.Bool("debug", false, "debug mode")

	usr *user.User
	err error
)

func init() {
	// add static directory flags to command line flagset
	for _, fs := range fileServers {
		_ = flag.Bool(fs, true, fmt.Sprintf("[debug] enable file server on <root>/%s/", fs))
	}
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

	if *rootPrefix == "~" || *rootPrefix == "" {
		*rootPrefix = usr.HomeDir
	}
	if *htmlRoot == "" {
		*htmlRoot = path.Join(*rootPrefix, "html")
	}

	if *debug {
		// populate default fileServers if debug is on
		for i, name := range fileServers {
			flg := flag.Lookup(name)
			if flg == nil {
				log.Printf("bad flag lookup: %s, removing from dirFlags", name)
				fileServers = append(fileServers[:i], fileServers[i+1:]...)
				continue
			}
			if flg.Value.String() == "" {
				if err := flg.Value.Set(*rootPrefix); err != nil {
					log.Printf("flg.ValueSet: %s", err)
				}
			}
		}
	}
}

func newHtmlServer(name string) *xhttp.HtmlServer {
	return &xhttp.HtmlServer{
		Root:  *htmlRoot,
		Debug: *debug,
		Name:  name,
		Data:  nil,
	}
}

func newSiphonServer(target string, handler http.Handler) *xhttp.SiphonServer {
	return &xhttp.SiphonServer{
		Handler: handler,
		Target:  target,
	}
}

func main() {
	r := mux.NewRouter()
	r.Handle("/favicon.ico", http.RedirectHandler("/img/favicon.png", http.StatusTemporaryRedirect))
	for _, fs := range fileServers {
		flg := flag.Lookup(fs)
		if flg == nil {
			log.Fatal("got unexpected nil lookup", fs)
		}

		prefix := fmt.Sprintf("/%s/", flg.Name)
		r.PathPrefix(prefix).Handler(
			http.StripPrefix(prefix, http.FileServer(http.Dir(path.Join(*rootPrefix, flg.Name)))))
		log.Printf("file server on /%s/ -> %s/", flg.Name, path.Join(*rootPrefix, flg.Name))
	}
	r.PathPrefix("/cv/").Handler(newSiphonServer("/cv/", newHtmlServer("cv.html")))
	r.PathPrefix("/works/").Handler(newSiphonServer("/works/", newHtmlServer("works.html")))
	r.PathPrefix("/fr/").Handler(newSiphonServer("/fr/", newHtmlServer("home.html")))
	r.PathPrefix("/fr/cv").Handler(newSiphonServer("/fr/cv/", newHtmlServer("cv.html")))
	r.PathPrefix("/fr/works").Handler(newSiphonServer("/fr/works/", newHtmlServer("works.html")))
	r.PathPrefix("/").Handler(newSiphonServer("/", newHtmlServer("home.html")))

	http.Handle("/", &xhttp.LogServer{Handler: r})

	addr := fmt.Sprintf("localhost:%d", *port)
	log.Printf("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
