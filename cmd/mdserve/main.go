package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"net/http"
	"path/filepath"
)

// ----

// Wrapper function that allows to panic() with a formatted string.
func varpanic(format string, args ...interface{}) {
	msg := fmt.Sprintf("ERROR: "+format+"\n", args...)
	panic(msg)
}

// ----

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hallo Welt"))
}

// Serves HTTP requests.
func serveHTTP(addr string, dir string) {
	fmt.Printf("Serving on http://%v\n", addr)
	http.HandleFunc("/", handleRequest)
	if err := http.ListenAndServe(addr, nil); err != nil {
		varpanic("%v", err)
	}
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
	dir, err := filepath.EvalSymlinks(*dirptr)
	if err != nil {
		varpanic("Couldn't get full path: %v", *dirptr)
	}

	addr := *addrptr
	if strings.Compare(addr[:1], ":") == 0 {
		addr = fmt.Sprintf("localhost%v", addr)
	}

	// ...and go to work.
	serveHTTP(addr, dir)
}
