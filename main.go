package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"path"

	"github.com/gorilla/mux"
	"github.com/samuel/go-gettext/gettext"
)

// default static file servers to serve when -debug is enabled
var defaultFileServers []string = []string{
	"share",
	"css",
	"img",
	"js",
}

var (
	cfgPath     = flag.String("cfg", "", "cfg path, defaults to <root>/config.json")
	port        = flag.Int("p", 8080, "listen port")
	logFile     = flag.String("log", "", "log.SetOutput")
	rootPrefix  = flag.String("root", "~", "root path of project for resources (share, css, js, img...)")
	htmlRoot    = flag.String("html", "", "path to html templates, defaults to <root>/html/")
	localesRoot = flag.String("locales", "", "path to locales tree, defaults to <html>/locales/")
	debug       = flag.Bool("debug", false, "debug mode")

	locales *gettext.Domain
	cfg     *Config
	usr     *user.User
	err     error
)

func init() {
	// add static directory flags to command line flagset
	for _, fs := range defaultFileServers {
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
	if *localesRoot == "" {
		*localesRoot = path.Join(*htmlRoot, "locales")
	}
	if *cfgPath == "" {
		*cfgPath = path.Join(*rootPrefix, "config.json")
	}

	if *debug {
		// populate default fileServers if debug is on
		for i, name := range defaultFileServers {
			flg := flag.Lookup(name)
			if flg == nil {
				log.Printf("bad flag lookup: %s, removing from dirFlags", name)
				defaultFileServers = append(defaultFileServers[:i], defaultFileServers[i+1:]...)
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

func newWorksServer(name string, works []Work) *HtmlServer {
	s := newHtmlServer(name)
	s.Data = &TplData{Works: works}
	return s
}

func newHtmlServer(name string) *HtmlServer {
	return &HtmlServer{
		Root:          *htmlRoot,
		Debug:         *debug,
		Name:          name,
		Data:          nil,
		DefaultLocale: "en",
		LocaleDomain:  locales,
	}
}

func newSiphonServer(target string, handler http.Handler) *SiphonServer {
	return &SiphonServer{
		Handler: handler,
		Target:  target,
	}
}

func main() {
	cfg, err = LoadConfigFile(*cfgPath)
	if err != nil {
		log.Fatal("LoadConfigFile", err)
	}
	locales, err = gettext.NewDomain("messages", *localesRoot)
	if err != nil {
		log.Fatal("gettext.NewDomain", err)
	}

	r := mux.NewRouter()

	// statix & shit
	r.Handle("/favicon.ico", http.RedirectHandler("/img/favicon.png", http.StatusTemporaryRedirect))
	for _, fs := range defaultFileServers {
		flg := flag.Lookup(fs)
		if flg == nil {
			log.Fatal("got unexpected nil lookup", fs)
		}

		prefix := fmt.Sprintf("/%s/", flg.Name)
		r.PathPrefix(prefix).Handler(
			http.StripPrefix(prefix, http.FileServer(http.Dir(path.Join(*rootPrefix, flg.Name)))))
		log.Printf("file server on /%s/ -> %s/", flg.Name, path.Join(*rootPrefix, flg.Name))
	}

	// main paths, from specific to broad
	r.PathPrefix("/cv").Handler(newSiphonServer("/cv/", newHtmlServer("cv.html")))
	r.PathPrefix("/works").Handler(newSiphonServer("/works/", newWorksServer("works.html", cfg.Works)))

	r.PathPrefix("/fr").Handler(newSiphonServer("/fr/", newHtmlServer("home.html")))
	r.PathPrefix("/").Handler(newSiphonServer("/", newHtmlServer("home.html")))

	// root handle on mux Router, clear handler
	http.Handle("/", &LogServer{Handler: r})

	addr := fmt.Sprintf("localhost:%d", *port)
	log.Printf("Listening on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
