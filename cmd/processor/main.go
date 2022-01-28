package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var logger *zap.SugaredLogger = nil

func main() {
	var help bool
	var debug bool
	var port int
	var err error

	flag.IntVar(&port, "p", -1, "Listen port")
	flag.BoolVar(&debug, "d", false, "Enable debug logging")
	flag.BoolVar(&help, "h", false, "Print help")
	flag.Parse()
	if !flag.Parsed() || help || port < 0 {
		flag.PrintDefaults()
		os.Exit(2)
	}

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

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		logger.Fatalf("Can't listen on socket: %s", err)
		os.Exit(3)
	}

	server := grpc.NewServer()
	service := processorService{}
	extproc.RegisterExternalProcessorServer(server, &service)

	logger.Infof("Listening on %s", listener.Addr())

	server.Serve(listener)
}
