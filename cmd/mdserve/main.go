package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

// ----

// Base dir to serve data from.
var basedir string

// ----

// Wrapper function that allows to panic() with a formatted string.
func varpanic(format string, args ...interface{}) {
	msg := fmt.Sprintf("ERROR: "+format+"\n", args...)
	panic(msg)
}

// ----

func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Sanitize the requested file path.
	requestpath := path.Clean(r.URL.Path)
	if strings.Compare(requestpath[:1], "/") == 0 {
		requestpath = requestpath[1:]
	} else if strings.Compare(requestpath[:3], "../") == 0 {
		requestpath = requestpath[3:]
	}

	// TODO: Handle static assets crunched into the binary.

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
			w.Write([]byte("Markdown"))
	}
}

// Serves HTTP requests.
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

	// Start serving.
	fmt.Printf("Serving on http://%v\n", addr)
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
	flag.Parse()

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

	addr := *addrptr
	if strings.Compare(addr[:1], ":") == 0 {
		addr = fmt.Sprintf("localhost%v", addr)
	}

	// ...and go to work.
	serveHTTP(addr)
}
