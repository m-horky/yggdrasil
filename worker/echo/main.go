package main

import (
	"context"
	"flag"
	"net"
	"os"
	"path/filepath"

	"git.sr.ht/~spc/go-log"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/fftoml"
	pb "github.com/redhatinsights/yggdrasil/protocol/grpc"
	worker "github.com/redhatinsights/yggdrasil/protocol/varlink/worker"
	"github.com/sgreben/flagvar"
	"github.com/varlink/go/varlink"
	"google.golang.org/grpc"
)

var (
	protocol      = flagvar.Enum{Choices: []string{"grpc", "varlink"}}
	yggSocketAddr = ""
	yggListenAddr = ""
	logLevel      = flagvar.Enum{Choices: []string{"error", "warn", "info", "debug", "trace"}}
)

func main() {
	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
	fs.StringVar(&yggSocketAddr, "socket-addr", "", "dispatcher socket address")
	fs.StringVar(&yggListenAddr, "listen-addr", "", "worker socket address")
	fs.Var(&logLevel, "log-level", "log verbosity level (error (default), warn, info, debug, trace)")
	fs.Var(&protocol, "protocol", "desired RPC protocol (grpc (default), varlink)")
	_ = fs.String("config", "", "path to `file` containing configuration values (optional)")

	if err := ff.Parse(fs, os.Args[1:], ff.WithEnvVarPrefix("YGG"), ff.WithConfigFileFlag("config"), ff.WithConfigFileParser(fftoml.Parser)); err != nil {
		log.Fatalf("error: cannot parse flags: %v", err)
	}

	if logLevel.Value != "" {
		l, err := log.ParseLevel(logLevel.Value)
		if err != nil {
			log.Fatalf("error: cannot parse log level: %v", err)
		}
		log.SetLevel(l)
	}

	if log.CurrentLevel() >= log.LevelDebug {
		log.SetFlags(log.LstdFlags | log.Llongfile)
	}

	switch protocol.Value {
	case "grpc":
		// Listen on the provided socket address.
		l, err := net.Listen("unix", yggListenAddr)
		if err != nil {
			log.Fatal(err)
		}

		// Register as a Worker service with gRPC and start accepting connections.
		s := grpc.NewServer()
		pb.RegisterWorkerServer(s, &echoServer{})
		if err := s.Serve(l); err != nil {
			log.Fatal(err)
		}
	case "varlink":
		service, err := varlink.NewService(
			"Red Hat",
			"yggdrasil-echo-worker",
			"1",
			"https://github.com/RedHatInsights/yggdrasil",
		)
		if err != nil {
			log.Fatalf("error: cannot create VARLINK service: %v", err)
		}

		if err := service.RegisterInterface(worker.VarlinkNew(&echoWorker{})); err != nil {
			log.Fatalf("error: cannot register VARLINK interface: %v", err)
		}

		log.Infoln("listening on socket: " + yggListenAddr)
		if err := service.Listen(context.Background(), "unix:"+yggListenAddr, 0); err != nil {
			log.Fatalf("error: cannot listen on socket %v: %v", yggListenAddr, err)
		}
	default:
		log.Fatal("error: unsupported RPC protocol: %v", protocol.Value)
	}
}
