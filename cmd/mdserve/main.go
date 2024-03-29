package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	// Chroma HTML formatter.
	// (Used by Goldmark)
	"github.com/alecthomas/chroma/formatters/html"

	// Goldmark CommonMark parser.
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"

	// KaTeX support for Goldmark.
	"github.com/FurqanSoftware/goldmark-katex"

	// Wikilink support for Goldmark
	"go.abhg.dev/goldmark/wikilink"

	// The assets.
	"github.com/yamagi/mdserve/assets"
)

// ----

// Base dir to serve data from.
var basedir string

// Global Goldmark instance.
var gm goldmark.Markdown

// Language to generate for.
var lang string

// Be quiet.
var quiet bool

// Static assets.
var css []byte
var template []byte

// ----

// Wrapper function that allows to panic() with a formatted string.
func varpanic(format string, args ...interface{}) {
	msg := fmt.Sprintf("ERROR: "+format+"\n", args...)
	panic(msg)
}

// ----

func getMarkdown(w http.ResponseWriter, filepath string) {
	markdown, err := ioutil.ReadFile(filepath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500: Internal server error"))
		return
	}

	// Convert the Markdown to HTML...
	var rawhtml bytes.Buffer
	context := parser.NewContext()
	if err := gm.Convert(markdown, &rawhtml, parser.WithContext(context)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("500: Internal server error"))
		return
	}
	html := rawhtml.String()

	// ...extract the metadata...
	metadata := meta.Get(context)

	var title string
	rawtitle := metadata["Title"]
	if strtitle, ok := rawtitle.(string); ok {
		if len(strtitle) == 0 {
			title = "mdserve: Markdown webserver"
		} else {
			title = strtitle
		}
	} else {
		title = "mdserve: Markdown webserver"
	}

	var date string
	rawdate := metadata["Date"]
	if strdate, ok := rawdate.(string); ok {
		if len(strdate) == 0 {
			date = time.Now().Format("02. January 2006")
		} else {
			date = strdate
		}
	} else {
		date = time.Now().Format("02. January 2006")
	}

	// ...put everything into the template...
	finalhtml := fmt.Sprintf(string(template), lang, title, html, date)

	// ...and return it to the client.
	w.Write([]byte(finalhtml))
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Sanitize the requested file path.
	requestpath := path.Clean(r.URL.Path)
	if strings.Compare(requestpath[:1], "/") == 0 {
		requestpath = requestpath[1:]
	} else if strings.Compare(requestpath[:3], "../") == 0 {
		requestpath = requestpath[3:]
	}

	// Serve static assets.
	if strings.Compare(requestpath, "assets/md.css") == 0 {
		w.Header().Set("Content-Type", "text/css")
		w.Write(css)
		return
	}
	if strings.HasPrefix(requestpath, "assets/") {
		file := strings.TrimPrefix(requestpath, "assets/")
		w.Header().Set("Content-Type", mime.TypeByExtension(file))
		if asset, err := assets.FS.ReadFile(file); err != nil {
			varpanic("Couldn't read %v: %v", file, err)
		} else {
			w.Write(asset)
			return
		}
	}

	// Everything else are files read from the filesystem.
	// Make sure that they exist and we've got permissions.
	requestpath = filepath.Join(basedir, requestpath)
	if reqstat, err := os.Stat(requestpath); err == nil {
		reqmode := reqstat.Mode()
		if reqmode.IsDir() {
			// A dir -> Redirect to index.md if any.
			indexpath := filepath.Join(requestpath, "index.md")
			if indexstat, err := os.Stat(indexpath); err == nil {
				indexmode := indexstat.Mode()
				if indexmode.IsRegular() {
					if fd, err := os.Open(indexpath); err == nil {
						// A readable index.md -> Redirect to it.
						fd.Close()
						target := url.URL{
							Scheme: "http",
							Host:   r.Host,
							Path:   path.Join(r.URL.Path, "index.md"),
						}
						http.Redirect(w, r, target.String(), http.StatusMovedPermanently)
						return
					}
				}
			}
		}
		if !reqmode.IsRegular() {
			// Not a file -> Don't serve it.
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403: Forbidden"))
			return
		} else {
			// Not readable -> Don't serve it.
			if fd, err := os.Open(requestpath); err != nil {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("403: Forbidden"))
				return
			} else {
				fd.Close()
			}
		}
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404: Not found"))
		return
	}

	// If it's not a Markdown file, just return it.
	// Otherwise convert the Mardown file to HTML.
	requestext := filepath.Ext(requestpath)
	if strings.Compare(requestext, ".md") != 0 &&
		strings.Compare(requestext, ".markdown") != 0 {
		http.ServeFile(w, r, requestpath)
	} else {
		getMarkdown(w, requestpath)
	}
}

func serveHTTP(addr string) {
	// Setup server.
	srv := http.Server{
		Addr: addr,
	}

	// Shut HTTP server down when signaled.
	done := make(chan bool)
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig

		if err := srv.Shutdown(context.Background()); err != nil {
			varpanic("%v", err)
		}
		close(done)
	}()

	// Print URL string
	if !quiet {
		indexpath := path.Join(basedir, "index.md")
		if _, err := os.Stat(indexpath); err != nil {
			fmt.Printf("Serving on http://%v\n", addr)
		} else {
			if fd, err := os.Open(indexpath); err != nil {
				fmt.Printf("Serving on http://%v\n", addr)
			} else {
				fd.Close()
				fmt.Printf("Serving on http://%v/index.md\n", addr)
			}
		}
	}

	// Start serving.
	http.HandleFunc("/", handleRequest)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		varpanic("%v", err)
	}
	<-done
}

// ----

func main() {
	// Die with nicer error messages.
	defer func() {
		if msg := recover(); msg != nil {
			fmt.Fprintf(os.Stderr, "%v", msg)
		}
	}()

	// Parse and check flags...
	var addrptr = flag.String("a", "localhost:8080", "Listen address")
	var dirptr = flag.String("d", ".", "Directory to serve")
	var justcssptr = flag.Bool("j", false, "Text with full justification")
	var langptr = flag.String("l", "de", "Typographic language")
	var quietptr = flag.Bool("q", false, "Be quiet")
	flag.Parse()

	addr := *addrptr
	if strings.Compare(addr[:1], ":") == 0 {
		addr = fmt.Sprintf("localhost%v", addr)
	}

	if stat, err := os.Stat(*dirptr); err == nil {
		if !stat.IsDir() {
			varpanic("Not a directory: %v", *dirptr)
		}
	} else {
		varpanic("No such file or directory: %v", *dirptr)
	}

	var err error
	basedir, err = filepath.EvalSymlinks(*dirptr)
	if err != nil {
		varpanic("Couldn't get full path: %v", *dirptr)
	}
	basedir, err = filepath.Abs(basedir)
	if err != nil {
		varpanic("Couldn't get full path: %v", *dirptr)
	}

	var csstype string
	if *justcssptr {
		csstype = "md-block.css"
	} else {
		csstype = "md-left.css"
	}

	var typo_lsq string
	var typo_rsq string
	var typo_ldq string
	var typo_rdq string
	if strings.Compare(*langptr, "de") == 0 {
		typo_lsq = "&sbquo;"
		typo_rsq = "&lsquo;"
		typo_ldq = "&bdquo;"
		typo_rdq = "&ldquo;"
		lang = "de"
	} else {
		typo_lsq = "&lsquo;"
		typo_rsq = "&rsquo;"
		typo_ldq = "&ldquo;"
		typo_rdq = "&rdquo;"
		lang = "en"
	}

	quiet = *quietptr

	// ...preload the CSS and the HTML template...
	if css, err = assets.FS.ReadFile(csstype); err != nil {
		varpanic("Couldn't read %v: %v", csstype, err)
	}
	if template, err = assets.FS.ReadFile("md.tmpl"); err != nil {
		varpanic("Couldn't read md.tmpl: %v", err)
	}

	// ...initialize global Goldmark instance...
	gm = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			extension.Footnote,
			&katex.Extender{},
			meta.Meta,
			&wikilink.Extender{},
			extension.NewTypographer(
				extension.WithTypographicSubstitutions(
					extension.TypographicSubstitutions{
						extension.LeftSingleQuote:  []byte(typo_lsq),
						extension.RightSingleQuote: []byte(typo_rsq),
						extension.LeftDoubleQuote:  []byte(typo_ldq),
						extension.RightDoubleQuote: []byte(typo_rdq),
					}),
			),
			highlighting.NewHighlighting(
				highlighting.WithStyle("tango"),
				highlighting.WithFormatOptions(
					html.WithLineNumbers(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	// ...and go to work.
	serveHTTP(addr)
}
