package main

// Main file
// TODO: add prometheus & zipkin tracing

import (
	"context"
	"encoding/xml"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/term"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	//"github.com/go-kit/kit/tracing/opentracing"
	stdopentracing "github.com/opentracing/opentracing-go"
	//zipkin "github.com/openzipkin/zipkin-go-opentracing"

	//"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/examples/shipping/booking"
	"github.com/go-kit/kit/examples/shipping/cargo"
	"github.com/go-kit/kit/examples/shipping/handling"
	"github.com/go-kit/kit/examples/shipping/inmem"
	"github.com/go-kit/kit/examples/shipping/inspection"
	"github.com/go-kit/kit/examples/shipping/location"
	"github.com/go-kit/kit/examples/shipping/routing"
	"github.com/go-kit/kit/examples/shipping/tracking"

	"github.com/newtonsystems/grpc_types/go/grpc_types"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	//"github.com/newtonsystems/agent-mgmt/app"
	"github.com/newtonsystems/agent-mgmt/app/endpoint"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/agent-mgmt/app/transport"
)

//var MongoDatabase string
//var MongoSession *mgo.Session

const (
	serviceName = "agent-mgmt"

	defaultMongoDatabase     = "db1"
	defaultPort              = "50000"
	defaultDebugHTTPPort     = "8080"
	defaultRoutingServiceURL = "http://localhost:7878"
	defaultLinkerdHost       = "linkerd:4141"
	defaultZipkinAddr        = "zipkin:9410"
)

type mgoDetails struct {
	db      string
	session models.Session
}

var MongoDetails mgoDetails

type TwiML struct {
	XMLName xml.Name `xml:"Response"`

	Say string `xml:",omitempty"`
}

func twiml(w http.ResponseWriter, r *http.Request) {
	twiml := TwiML{Say: "Hello World!"}
	x, err := xml.Marshal(twiml)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Write(x)
}

func main() {
	var (
		addr  = envString("PORT", defaultPort)
		rsurl = envString("ROUTINGSERVICE_URL", defaultRoutingServiceURL)

		httpAddr          = flag.String("http.addr", ":"+defaultDebugHTTPPort, "HTTP listen address")
		routingServiceURL = flag.String("service.routing", rsurl, "routing service URL")

		mongoDB    = envString("MONGO_DB", defaultMongoDatabase)
		mongoDebug = flag.Bool("mongo.debug", false, "Turns on mongo debug.")

		// Other services
		zipkinAddr = flag.String("zipkin.addr", defaultZipkinAddr, "Enable Zipkin tracing via a Zipkin HTTP Collector endpoint")

		ctx = context.Background()
	)

	flag.Parse()

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

	// Logging domain.
	var logger log.Logger
	{
		//logger = log.NewLogfmtLogger(os.Stdout)
		logger = term.NewLogger(os.Stdout, log.NewLogfmtLogger, colorFn)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
		logger = log.With(logger, "service", "agent-mgmt")
	}

	cmd := exec.Command("cat", "/etc/hostname")
	stdout, err := cmd.Output()

	if err != nil {
		println(err.Error())
		return
	}

	logger.Log("msg", "starting ...", "level", "info", "container", stdout, "dan", "dan12")
	defer logger.Log("msg", "goodbye")

	var (
		cargos         = inmem.NewCargoRepository()
		locations      = inmem.NewLocationRepository()
		voyages        = inmem.NewVoyageRepository()
		handlingEvents = inmem.NewHandlingEventRepository()
	)

	// Configure some questionable dependencies.
	var (
		handlingEventFactory = cargo.HandlingEventFactory{
			CargoRepository:    cargos,
			VoyageRepository:   voyages,
			LocationRepository: locations,
		}
		handlingEventHandler = handling.NewEventHandler(
			inspection.NewService(cargos, handlingEvents, nil),
		)
	)

	// Facilitate testing by adding some cargos.
	storeTestData(cargos)

	fieldKeys := []string{"method"}

	var rs routing.Service
	rs = routing.NewProxyingMiddleware(ctx, *routingServiceURL)(rs)

	var bs booking.Service
	bs = booking.NewService(cargos, locations, handlingEvents, rs)
	bs = booking.NewLoggingService(log.With(logger, "component", "booking"), bs)
	bs = booking.NewInstrumentingService(
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "api",
			Subsystem: "booking_service",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fieldKeys),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "api",
			Subsystem: "booking_service",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fieldKeys),
		bs,
	)

	var ts tracking.Service
	ts = tracking.NewService(cargos, handlingEvents)
	ts = tracking.NewLoggingService(log.With(logger, "component", "tracking"), ts)
	ts = tracking.NewInstrumentingService(
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "api",
			Subsystem: "tracking_service",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fieldKeys),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "api",
			Subsystem: "tracking_service",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fieldKeys),
		ts,
	)

	var hs handling.Service
	hs = handling.NewService(handlingEvents, handlingEventFactory, handlingEventHandler)
	hs = handling.NewLoggingService(log.With(logger, "component", "handling"), hs)
	hs = handling.NewInstrumentingService(
		kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "api",
			Subsystem: "handling_service",
			Name:      "request_count",
			Help:      "Number of requests received.",
		}, fieldKeys),
		kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "api",
			Subsystem: "handling_service",
			Name:      "request_latency_microseconds",
			Help:      "Total duration of requests in microseconds.",
		}, fieldKeys),
		hs,
	)

	httpLogger := log.With(logger, "component", "http")

	mux := http.NewServeMux()

	mux.Handle("/booking/v1/", booking.MakeHandler(bs, httpLogger))
	mux.Handle("/tracking/v1/", tracking.MakeHandler(ts, httpLogger))
	mux.Handle("/handling/v1/", handling.MakeHandler(hs, httpLogger))

	http.Handle("/", accessControl(mux))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/twiml", twiml)

	errs := make(chan error, 2)
	go func() {
		logger.Log("transport", "http", "address", *httpAddr, "msg", "listening")
		errs <- http.ListenAndServe(*httpAddr, nil)
	}()
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT)
		errs <- fmt.Errorf("%s", <-c)
	}()

	errc := make(chan error, 2)

	mongoHosts := []string{
		"mongo-0.mongo:27017",
		"mongo-1.mongo:27017",
		"mongo-2.mongo:27017",
		// dev environments for: master / featuretest
		"mongo-0.mongo.dev-common.svc.cluster.local:27017",
		"mongo-1.mongo.dev-common.svc.cluster.local:27017",
		"mongo-2.mongo.dev-common.svc.cluster.local:27017",
	}

	//const (
	// TODO: Add auth to mongo
	//	MongoUsername   = "YOUR_USERNAME"
	//	MongoPassword   = "YOUR_PASS"
	//MongoDatabase = mongoDB2
	//	Collection = "YOUR_COLLECTION"
	//)

	// We need this object to establish a session to our MongoDB.
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    mongoHosts,
		Timeout:  60 * time.Second,
		Database: mongoDB,
		// TODO: Add auth to mongo
		//Username: MongoUsername,
		//Password: MongoPassword,
	}

	// -------------------------------------------------------------------- //

	// Initialise mongodb connection
	// Create a session which maintains a pool of socket connections to our MongoDB.
	mongoSession, mongoLogger := models.NewMongoSession(mongoDBDialInfo, logger, *mongoDebug)
	defer mongoSession.Close()

	models.PrepareDB(mongoSession, mongoDB, mongoLogger)

	// -------------------------------------------------------------------- //

	// gRPC service / service metrics / endpoints / connect to linkerd

	// Create the (sparse) metrics we'll use in the service. They, too, are
	// dependencies that we pass to components that use them.
	// TODO: change namespace
	var ints, chars, refs, beats metrics.Counter
	{
		// Business-level metrics.
		ints = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "addsvc",
			Name:      "integers_summed",
			Help:      "Total count of integers summed via the Sum method.",
		}, []string{})
		chars = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "addsvc",
			Name:      "characters_concatenated",
			Help:      "Total count of characters concatenated via the Concat method.",
		}, []string{})
		refs = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "agentmgmt",
			Name:      "references_used",
			Help:      "Total count of references used to get agent ID via the GetAgentIDFromRef method.",
		}, []string{})
		beats = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "agentmgmt",
			Name:      "total_heartbeat_counts",
			Help:      "Total count of heartbeats service call from the HeartBeat method.",
		}, []string{})
	}

	var duration metrics.Histogram
	{
		// Transport level metrics.
		duration = kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
			Namespace: "main",
			Name:      "request_duration_ns",
			Help:      "Request duration in nanoseconds.",
		}, []string{"method", "success"})
	}

	// Connect to local linkerd
	linkerdLogger := log.With(logger, "connection", "linkerd")
	// If address is incorrect retries forever at the moment
	// https://github.com/grpc/grpc-go/issues/133
	conn, err := grpc.Dial(defaultLinkerdHost, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
	if err != nil {
		linkerdLogger.Log("msg", "Failed to connect to local linkerd", "level", "crit")
		errc <- err
		return
	}
	defer conn.Close()

	linkerdLogger.Log("host", defaultLinkerdHost, "msg", "successfully connected")

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

	MongoDetails.db = mongoDB
	MongoDetails.session = mongoSession

	var (
		service   = service.NewService(logger, ints, chars, refs, beats)
		endpoints = endpoint.NewEndpoint(service, logger, duration, tracer, mongoSession, mongoDB)
	)

	// gRPC transport
	go func() {
		gRPCLogger := log.With(logger, "transport", "gRPC")

		gRPCLogger.Log("addr", addr, "port is avasdasailable")

		ln, err := net.Listen("tcp", ":"+addr)
		if err != nil {
			errc <- err
			return
		}

		gRPCLogger.Log("addr", addr, "port is available")

		srv := transport.GRPCServer(endpoints, tracer, gRPCLogger)
		s := grpc.NewServer()
		grpc_types.RegisterAgentMgmtServer(s, srv)

		errc <- s.Serve(ln)
	}()

	// -------------------------------------------------------------------- //

	// Run!
	logger.Log("exit", <-errc)
}

func accessControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")

		if r.Method == "OPTIONS" {
			return
		}

		h.ServeHTTP(w, r)
	})
}

type Person struct {
	Name  string
	Phone string
}

func main_mongo_test(logger log.Logger) {
	session, err := mgo.Dial("mongodb://mongo-0.mongo,mongo-1.mongo,mongo-2.mongo:27017")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("test").C("people")
	err = c.Insert(&Person{"Ale", "+55 53 8116 9639"},
		&Person{"Cla", "+55 53 8402 8510"})
	if err != nil {
		logger.Log("msg", err)
	}

	result := Person{}
	err = c.Find(bson.M{"name": "Ale"}).One(&result)
	if err != nil {
		logger.Log("msg", err)
	}

	fmt.Println("Phone:", result.Phone)
}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

func storeTestData(r cargo.Repository) {
	test1 := cargo.New("FTL456", cargo.RouteSpecification{
		Origin:          location.AUMEL,
		Destination:     location.SESTO,
		ArrivalDeadline: time.Now().AddDate(0, 0, 7),
	})
	if err := r.Store(test1); err != nil {
		panic(err)
	}

	test2 := cargo.New("ABC123", cargo.RouteSpecification{
		Origin:          location.SESTO,
		Destination:     location.CNHKG,
		ArrivalDeadline: time.Now().AddDate(0, 0, 14),
	})
	if err := r.Store(test2); err != nil {
		panic(err)
	}
}
