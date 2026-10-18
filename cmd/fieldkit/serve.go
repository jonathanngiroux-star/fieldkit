package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/jonathanngiroux-star/fieldkit/internal/serve"
	"github.com/jonathanngiroux-star/fieldkit/internal/store"
)

// cmdServe runs the HTTP server: public submit endpoint, survey views,
// /donate, /health. Blocks until SIGINT/SIGTERM.
func cmdServe(args []string, stdout, stderr io.Writer) int {
	fs := newFlagSet("serve", stderr)
	addr := fs.String("addr", "127.0.0.1:8080", "listen address")
	dataDir := fs.String("data-dir", defaultDataDir(), "data directory for the SQLite store")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	st, err := store.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "open store: %v\n", err)
		return 1
	}
	defer st.Close()
	h := serve.New(st)
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintf(stderr, "listen %s: %v\n", *addr, err)
		return 1
	}
	srv := &http.Server{Handler: h}
	fmt.Fprintf(stdout, "fieldkit serving on http://%s (data: %s)\n", ln.Addr().String(), st.Path())
	fmt.Fprintln(stdout, "donate page: http://"+ln.Addr().String()+"/donate")
	go srv.Serve(ln)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	ln.Close()
	fmt.Fprintln(stdout, "stopped")
	return 0
}

// startServe starts the HTTP server on an ephemeral port for tests.
// Returns a stop func and the chosen port.
func startServe(st *store.Store, stdout, stderr io.Writer) (stop func(), port string, err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, "", err
	}
	srv := &http.Server{Handler: serve.New(st)}
	go srv.Serve(ln)
	return func() { srv.Close() }, strconv.Itoa(ln.Addr().(*net.TCPAddr).Port), nil
}

// cmdExport writes responses as CSV to stdout.
func cmdExport(args []string, stdout, stderr io.Writer) int {
	format := ""
	var rest []string
	if len(args) > 0 {
		format = args[0]
		rest = args[1:]
	}
	switch format {
	case "csv":
	case "":
		fmt.Fprintln(stderr, "usage: fieldkit export csv --survey <id> --data-dir <dir>")
		return 2
	default:
		fmt.Fprintf(stderr, "unknown export format %q (want csv)\n", format)
		return 2
	}
	fs := newFlagSet("export csv", stderr)
	surveyID := fs.Int64("survey", 0, "survey id")
	dataDir := fs.String("data-dir", defaultDataDir(), "data directory for the SQLite store")
	if err := fs.Parse(rest); err != nil {
		return 2
	}
	if *surveyID == 0 {
		fmt.Fprintln(stderr, "--survey is required")
		return 2
	}
	st, err := store.Open(*dataDir)
	if err != nil {
		fmt.Fprintf(stderr, "open store: %v\n", err)
		return 1
	}
	defer st.Close()
	csvOut, err := st.ExportCSV(*surveyID)
	if err != nil {
		fmt.Fprintf(stderr, "export: %v\n", err)
		return 1
	}
	w := bufio.NewWriter(stdout)
	w.WriteString(csvOut)
	w.Flush()
	return 0
}
