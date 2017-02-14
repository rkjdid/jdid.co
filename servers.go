package main

import (
	"fmt"
	"github.com/samuel/go-gettext/gettext"
	"html/template"
	"log"
	"net/http"
	"path"
	"strings"
)

// CustomResponseWriter allows to store current status code of ResponseWriter.
type CustomResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (w *CustomResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *CustomResponseWriter) Write(data []byte) (int, error) {
	return w.ResponseWriter.Write(data)
}

func (w *CustomResponseWriter) WriteHeader(statusCode int) {
	// set w.Status then forward to inner ResposeWriter
	w.Status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func WrapCustomRW(wr http.ResponseWriter) http.ResponseWriter {
	if _, ok := wr.(*CustomResponseWriter); !ok {
		return &CustomResponseWriter{
			ResponseWriter: wr,
			Status:         http.StatusOK, // defaults to ok, some servers might not call wr.WriteHeader at all
		}
	}
	return wr
}

// HtmlServer is a simple html/template server helper
type HtmlServer struct {
	Root          string
	Name          string
	Data          *TplData
	Debug         bool
	DefaultLocale string
	LocaleDomain  *gettext.Domain
}

func (hs *HtmlServer) IsLangSupported(lang string) bool {
	if hs.LocaleDomain == nil {
		return false
	}
	_, ok := hs.LocaleDomain.Languages[lang]
	return ok
}

// ProcessLang extracts language information from r, and applies it to w
// if language is supported by hs.
//
// In order are tried:
//   - query parameter ?lang=fr
//   - cookie value from previous visit
//   - Accept-Language from request's header
//   - default language from HtmlServer
func (hs *HtmlServer) ProcessLang(w http.ResponseWriter, r *http.Request) string {
	qLang := r.URL.Query().Get("lang")
	cookie, _ := r.Cookie("lang")

	// fix bad query string
	if qLang != "" && !hs.IsLangSupported(qLang) {
		qLang = ""
	}
	// fix bad cookie
	if cookie != nil && !hs.IsLangSupported(cookie.Value) {
		cookie = nil
	}

	// now qLang & cookie are either safe or zeroed, priority order:
	// query -> cookie -> request's Accept-Language -> default
	if qLang != "" {
		// 1- use query (noop)
		qLang = qLang
	} else if cookie != nil {
		// 2- or use cookie
		qLang = cookie.Value
	} else {
		reqLang := r.Header.Get("Accept-Language")
		reqLang = strings.ToLower(reqLang)
		if len(reqLang) > 2 {
			reqLang = reqLang[:2]
		}

		// 3- or try Accept-Language
		if hs.IsLangSupported(reqLang) {
			qLang = reqLang
		} else {
			// 4- use default language
			qLang = hs.DefaultLocale
		}
	}

	if cookie == nil {
		cookie = &http.Cookie{
			Name: "lang",
		}
	}
	cookie.Path = "/"
	cookie.Value = qLang

	// now write cookie to response and return lang value
	http.SetCookie(w, cookie)
	return qLang
}

func (hs *HtmlServer) Gettext(lang string, msgid string) template.HTML {
	if hs.LocaleDomain != nil {
		return template.HTML(hs.LocaleDomain.GetText(lang, msgid))
	}
	return template.HTML(msgid)
}

func (hs *HtmlServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fns := template.FuncMap{
		"gettext": hs.Gettext,
	}

	t, err := template.New(hs.Name).Funcs(fns).ParseFiles(path.Join(hs.Root, hs.Name))
	if err != nil {
		log.Printf("%s -> err parsing %s: %s", r.URL.Path, hs.Name, err)
		if hs.Debug {
			http.Error(w, fmt.Sprintf("in template.ParseFiles of %s: %s", hs.Name, err), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	lang := hs.ProcessLang(w, r)
	err = t.ExecuteTemplate(w, hs.Name, hs.Data.SetLang(lang))
	if err != nil {
		log.Printf("%s -> err executing template %s: %s", r.URL.Path, hs.Name, err)
		if hs.Debug {
			http.Error(w, fmt.Sprintf("in template.Execute: %s", err), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}
}

// WatServer is basically a 404 fallback server
type WatServer struct{}

func (ws *WatServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "<html><head><title>what?</title></head><body>looking for <em>%s</em> ?</body>", r.URL.Path)
}

// LogServer is a simple log wrapper to either a http.Handler, or if Handler is nil,
// to HandleFunc. It provides simple logging before responding with one of the inner handlers
type LogServer struct {
	http.Handler
	Name       string
	HandleFunc func(http.ResponseWriter, *http.Request)
}

// ServeHTTP satisfies the http.Handler interface, in turns it tries for ls.Handler,
// ls.HandleFunc, or returns default http.NotFound.
func (ls *LogServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w = WrapCustomRW(w)
	if ls.Handler != nil {
		ls.Handler.ServeHTTP(w, r)
	} else if ls.HandleFunc != nil {
		ls.HandleFunc(w, r)
	} else {
		http.NotFound(w, r)
	}

	log.Printf("%s-%s> (%d) @%s: %s - agent:%s",
		r.Host, ls.Name, w.(*CustomResponseWriter).Status, r.Header.Get("X-FORWARDED-FOR"), r.RequestURI, r.Header.Get("USER-AGENT"))
}

// SiphonServer is useful to allow all patterns to redirect to the siphon url, /%s/
type SiphonServer struct {
	http.Handler
	Target        string
	SiphonQueries bool
}

func (ss *SiphonServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var uri string
	if ss.SiphonQueries {
		uri = r.URL.String()
	} else {
		uri = r.URL.Path
	}

	if uri != ss.Target {
		http.Redirect(w, r, ss.Target, http.StatusFound)
		return
	}
	ss.Handler.ServeHTTP(w, r)
	return
}
