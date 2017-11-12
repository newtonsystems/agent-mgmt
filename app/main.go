package main

// Main file
// TODO: add prometheus & zipkin tracing

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/term"
	//"github.com/go-kit/kit/tracing/opentracing"
	stdopentracing "github.com/opentracing/opentracing-go"
	//zipkin "github.com/openzipkin/zipkin-go-opentracing"

	//"github.com/go-kit/kit/endpoint"

	"gopkg.in/mgo.v2"
	//"github.com/newtonsystems/agent-mgmt/app"
	"github.com/newtonsystems/agent-mgmt/app/endpoint"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/agent-mgmt/app/transport"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
)

const (
	serviceName          = "agent-mgmt"
	defaultMongoDatabase = "db1"
	defaultPort          = ":50000"
	defaultDebugHTTPPort = ":9090"
	defaultLinkerdHost   = "linkerd:4141"
	defaultZipkinAddr    = "zipkin:9410"
)

func main() {
	var (
		// Main configuration (via environment variables)
		addr    = envString("PORT", defaultPort)
		mongoDB = envString("MONGO_DB", defaultMongoDatabase)

		// Other services (Debug HTTP probe/metrics/debug + Tracing)
		debugAddr  = flag.String("debug.addr", defaultDebugHTTPPort, "Debug and metrics listen address")
		zipkinAddr = flag.String("zipkin.addr", defaultZipkinAddr, "Zipkin address for tracing via a Zipkin HTTP Collector endpoint")

		// Extra Options
		// Connect to minikube when running locally?
		localConn = flag.Bool("conn.local", false, "Override mongo/linkerd connection (specific for mongo-external or defaults to minikube conn)")
		// Mongo Debug enabled?
		mongoDebug = flag.Bool("mongo.debug", false, "Turns on mongo debug.")
	)

	flag.Parse()

	// ---------------------------------------------------------------------------
	// Colourised Logging

	// Color by level value
	colorFn := func(keyvals ...interface{}) term.FgBgColor {
		for i := 0; i < len(keyvals)-1; i += 2 {
			if keyvals[i] != "level" {
				continue
			}
			switch keyvals[i+1] {
			case "debug":
				return term.FgBgColor{Fg: term.DarkGray}
			case "info":
				return term.FgBgColor{Fg: term.DarkGreen}
			case "warn":
				return term.FgBgColor{Fg: term.Yellow, Bg: term.White}
			case "error":
				return term.FgBgColor{Fg: term.Red}
			case "crit":
				return term.FgBgColor{Fg: term.Gray, Bg: term.DarkRed}
			default:
				return term.FgBgColor{}
			}
		}
		return term.FgBgColor{}
	}

	var logger log.Logger
	{
		logger = term.NewLogger(os.Stdout, log.NewLogfmtLogger, colorFn)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
		logger = log.With(logger, "service", serviceName)
	}

	// ---------------------------------------------------------------------------

	var (
		started      = time.Now()
		container    string
		mongoHosts   []string
		l5dHost      string
		l5dConn      *grpc.ClientConn
		mongoSession models.Session
		mongoLogger  log.Logger
	)

	// Error channel
	errc := make(chan error, 2)

	// Depending on the environment we connect to different hosts
	// if local.conn is true we want to connecting by running the go service locally
	// we therefore want to connect to the minikube's linkerd and mongo services
	// else we are inside the kubernetes cluster and connect normally via kubernetes DNS
	if *localConn {
		container = "localhost"
		l5dHost = envString("LINKERD_SERVICE_HOST", "192.168.99.100") + ":" + envString("LINKERD_SERVICE_PORT", "31000")
		mongoHosts = []string{
			envString("MONGO_EXTERNAL_SERVICE_HOST", "192.168.99.100") + ":" + envString("MONGO_EXTERNAL_SERVICE_PORT", "31017"),
		}
	} else {
		l5dHost = defaultLinkerdHost
		mongoHosts = []string{
			"mongo-0.mongo:27017",
			"mongo-1.mongo:27017",
			"mongo-2.mongo:27017",
			// dev environments for: master / featuretest
			"mongo-0.mongo.dev-common.svc.cluster.local:27017",
			"mongo-1.mongo.dev-common.svc.cluster.local:27017",
			"mongo-2.mongo.dev-common.svc.cluster.local:27017",
			"mongodb://mongo-0.mongo,mongo-1.mongo,mongo-2.mongo:27017",
		}

		// Try and workout container id
		cmd := exec.Command("cat", "/etc/hostname")
		stdout, err := cmd.Output()

		if err != nil {
			logger.Log("level", "error", "error", err.Error(), "msg", "Failed in cat command to workout hostname")
			return
		}

		container = string(stdout)
	}

	logger.Log("level", "info", "container", container, "started", started, "msg", "starting ...", "stage", "#started")
	defer logger.Log("msg", "goodbye")

	// ---------------------------------------------------------------------------
	//
	// Mongo Setup
	//

	// Initialise mongodb connection
	// Create a session which maintains a pool of socket connections to our MongoDB.
	// Prepare the database
	// We need this object to establish a session to our MongoDB.

	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    mongoHosts,
		Timeout:  60 * time.Second,
		Database: mongoDB,
		// TODO: Add auth to mongo
		//Username: MongoUsername,
		//Password: MongoPassword,
	}

	mongoSession, mongoLogger = models.NewMongoSession(mongoDBDialInfo, logger, *mongoDebug)
	defer mongoSession.Close()

	models.PrepareDB(mongoSession, mongoDB, mongoLogger)

	// ---------------------------------------------------------------------------
	// LinkerD Setup
	// If address is incorrect retries forever at the moment
	// https://github.com/grpc/grpc-go/issues/133

	var errL5d error

	l5dLogger := log.With(logger, "connection", "linkerd")

	l5dConn, errL5d = grpc.Dial(
		l5dHost,
		grpc.WithInsecure(),
		grpc.WithTimeout(time.Second),
	)

	if errL5d != nil {
		l5dLogger.Log("level", "crit", "msg", "Failed to connect to local linkerd")
		errc <- errL5d
		return
	}

	defer l5dConn.Close()
	l5dLogger.Log("host", l5dHost, "msg", "successfully connected")

	// ---------------------------------------------------------------------------
	//
	// Main
	//

	var (
		tracer    = newTracer(logger, zipkinAddr)
		metrics   = service.NewMetrics()
		service   = service.NewService(logger, &metrics)
		endpoints = endpoint.NewEndpoint(service, logger, metrics.Duration, tracer, mongoSession, mongoDB)
	)

	// ---------------------------------------------------------------------------
	//
	// HTTP server (Probes + For debug + prom stats)
	//
	httpLogger := log.With(logger, "component", "probe", "transport", "http")

	// Liveness probe
	http.HandleFunc("/started", func(w http.ResponseWriter, r *http.Request) {
		httpLogger.Log("msg", "started")
		w.WriteHeader(200)
		data := (time.Now().Sub(started)).String()
		w.Write([]byte(data))
	})

	// Readiness probe
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		httpLogger.Log("msg", "healthz")
		var errorLinker error
		var errorMongo error
		var ok = true
		duration := time.Now().Sub(started)

		// Connected to mongo, check
		if mongoSession != nil {
			if errorMongo = mongoSession.Ping(); errorMongo != nil {
				ok = false
			}
		}

		// Connected to linkerd, check
		if l5dConn != nil {
			client := grpc_types.NewPingClient(l5dConn)
			_, errorLinker = client.Ping(
				context.Background(),
				&grpc_types.PingRequest{Message: "agent-mgmt"},
			)

			if errorLinker != nil {
				ok = false
			}
		}

		if ok {
			w.WriteHeader(200)
			w.Write([]byte("ok"))

		} else {
			httpLogger.Log("level", "error", "msg", fmt.Sprintf("Readiness Error linkerErr: %v, mongoErr: %v, duration: %v", errorLinker, errorMongo, duration.Seconds()))
			w.WriteHeader(500)
			w.Write([]byte(fmt.Sprintf("linkerErr: %v, mongoErr: %v, duration: %v", errorLinker, errorMongo, duration.Seconds())))
		}
	})

	// Debug metrics for go
	http.HandleFunc("/debug/pprof/", http.HandlerFunc(pprof.Index))
	http.HandleFunc("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	http.HandleFunc("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	http.HandleFunc("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	http.HandleFunc("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	// Metrics for prometheus
	http.Handle("/metrics", promhttp.Handler())

	go func() {
		httpLogger.Log("addr", *debugAddr, "msg", "Running debug/probe/metrics http server")
		errc <- http.ListenAndServe(*debugAddr, nil)
	}()

	httpLogger.Log("msg", "successfully connected")

	// ---------------------------------------------------------------------------
	//
	// Interrupt Go-Routines (ctrl + c)
	//

	// Interrupt handler.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// ---------------------------------------------------------------------------
	//
	// gRPC server (Main service)
	//

	go func() {
		gRPCLogger := log.With(logger, "component", "server", "transport", "gRPC")

		ln, err := net.Listen("tcp", addr)
		if err != nil {
			errc <- err
			return
		}
		defer ln.Close()

		gRPCLogger.Log("level", "debug", "addr", addr, "msg", "port is available")
		gRPCLogger.Log("addr", addr, "msg", "Running gRPC server")

		srv := transport.GRPCServer(endpoints, tracer, gRPCLogger)
		s := grpc.NewServer()
		defer s.GracefulStop()
		grpc_types.RegisterAgentMgmtServer(s, srv)

		errc <- s.Serve(ln)
	}()

	// ---------------------------------------------------------------------------

	// Exit!
	logger.Log("exit", <-errc)
}

// ----
func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

func newTracer(logger log.Logger, zipkinAddr *string) stdopentracing.Tracer {
	// Tracing domain.
	var tracer stdopentracing.Tracer
	{
		if *zipkinAddr != "" {
			logger := log.With(logger, "tracer", "Zipkin-gRPC")
			logger.Log("addr", *zipkinAddr)

			// // endpoint typically looks like: http://zipkinhost:9411/api/v1/spans
			// collector, err := zipkin.NewHTTPCollector(*zipkinAddr)
			// if err != nil {
			// 	logger.Log("err", err)
			// 	os.Exit(1)
			// }
			// defer collector.Close()

			// tracer, err = zipkin.NewTracer(
			// 	zipkin.NewRecorder(collector, false, "localhost:"+addr, serviceName),
			// )
			// if err != nil {
			// 	logger.Log("err", err)
			// 	os.Exit(1)
			// }
		} else {
			logger := log.With(logger, "tracer", "none")
			logger.Log()
			tracer = stdopentracing.GlobalTracer() // no-op
		}
	}

	return tracer
}
