package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	var doHelp bool
	var port int

	flag.BoolVar(&doHelp, "h", false, "Print help message")
	flag.IntVar(&port, "p", -1, "TCP listen port")
	flag.Parse()

	if !flag.Parsed() || doHelp || port < 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}

	// This extra stuff lets us support HTTP/2 without
	// TLS using the "h2c" extension.
	handler := createHandler()
	h2Server := http2.Server{}
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(handler, &h2Server),
	}
	server.ListenAndServe()
}
