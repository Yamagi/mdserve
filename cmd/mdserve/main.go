package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
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
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark-highlighting"
	"github.com/yuin/goldmark-meta"

	// Parcello...
	"github.com/phogolabs/parcello"

	// ...and the assets.
	_ "github.com/yamagi/mdserve/assets"
)

// ----

// Base dir to serve data from.
var basedir string

// Global Goldmark instance.
var gm goldmark.Markdown

// Language to generate for.
var lang string

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
	if strings.Compare(requestpath, "/assets/md.css") == 0 ||
		strings.Compare(requestpath, "assets/md.css") == 0 {
		w.Header().Set("Content-Type", "text/css")
		w.Write(css)
		return
	}

	// Everything else are files read from the filesystem.
	// Make sure that they exist and we've got permissions.
	requestpath = filepath.Join(basedir, requestpath)
	if stat, err := os.Stat(requestpath); err == nil {
		mode := stat.Mode()
		if !mode.IsRegular() || mode & (1 << 7) == 0 {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("403: Forbidden"))
			return
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
	srv := http.Server {
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
	indexpath := path.Join(basedir, "index.md")
	if _, err := os.Stat(indexpath); err != nil {
		fmt.Printf("Serving on http://%v\n", addr)
	} else {
		fmt.Printf("Serving on http://%v/index.md\n", addr)
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
	var langptr = flag.String("l", "de", "Typographic language")
	flag.Parse()

	if stat, err := os.Stat(*dirptr); err == nil {
		if !stat.IsDir() {
			varpanic("Not a directory: %v", *dirptr)
		}
	} else {
		varpanic("No such file or directory: %v", *dirptr)
	}

	addr := *addrptr
	if strings.Compare(addr[:1], ":") == 0 {
		addr = fmt.Sprintf("localhost%v", addr)
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

	// ...load static assets...
	cssfile, err := parcello.Open("md.css")
	if err != nil {
		varpanic("Couldn't load md.css: %v", err)
	}
	if css, err = ioutil.ReadAll(cssfile); err != nil {
		varpanic("Couldn't read md.css: %v", err)
	}
	cssfile.Close()

	templatefile, err := parcello.Open("md.tmpl")
	if err != nil {
		varpanic("Couldn't load md.tmpl: %v", err)
	}
	if template, err = ioutil.ReadAll(templatefile); err != nil {
		varpanic("Couldn't read md.tmpl: %v", err)
	}
	templatefile.Close()

	// ...initialize global Goldmark instance...
	gm = goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			extension.Footnote,
			meta.Meta,
			extension.NewTypographer(
				extension.WithTypographicSubstitutions(
					extension.TypographicSubstitutions{
						extension.LeftSingleQuote: []byte(typo_lsq),
						extension.RightSingleQuote: []byte(typo_rsq),
						extension.LeftDoubleQuote: []byte(typo_ldq),
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
