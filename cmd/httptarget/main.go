package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var logger *zap.SugaredLogger = nil

func main() {
	var doHelp bool
	var debug bool
	var port int

	flag.BoolVar(&doHelp, "h", false, "Print help message")
	flag.BoolVar(&debug, "d", false, "Enable debug logging")
	flag.IntVar(&port, "p", -1, "TCP listen port")
	flag.Parse()

	if !flag.Parsed() || doHelp || port < 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}

	var err error
	var zapLogger *zap.Logger
	if debug {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		panic(fmt.Sprintf("Can't initialize logger: %s", err))
	}
	logger = zapLogger.Sugar()

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
