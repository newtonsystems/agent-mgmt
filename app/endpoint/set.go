package endpoint

import (
	"context"

	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
)

// Set collects all of the endpoints that compose an add service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
	SumEndpoint                endpoint.Endpoint
	ConcatEndpoint             endpoint.Endpoint
	SayHelloEndpoint           endpoint.Endpoint
	SayWorldEndpoint           endpoint.Endpoint
	GetAvailableAgentsEndpoint endpoint.Endpoint
	GetAgentIDFromRefEndpoint  endpoint.Endpoint
	HeartBeatEndpoint          endpoint.Endpoint
}

// New returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func NewEndpoint(svc service.Service, logger log.Logger, duration metrics.Histogram, trace stdopentracing.Tracer, session models.Session, db string) Set {
	// var sumEndpoint endpoint.Endpoint
	// {
	// 	sumEndpoint = MakeSumEndpoint(svc)
	// 	sumEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(1, 1))(sumEndpoint)
	// 	sumEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(sumEndpoint)
	// 	sumEndpoint = opentracing.TraceServer(trace, "Sum")(sumEndpoint)
	// 	sumEndpoint = LoggingMiddleware(log.With(logger, "method", "Sum"))(sumEndpoint)
	// 	sumEndpoint = InstrumentingMiddleware(duration.With("method", "Sum"))(sumEndpoint)
	// }
	// var concatEndpoint endpoint.Endpoint
	// {
	// 	concatEndpoint = MakeConcatEndpoint(svc)
	// 	concatEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(concatEndpoint)
	// 	concatEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(concatEndpoint)
	// 	concatEndpoint = opentracing.TraceServer(trace, "Concat")(concatEndpoint)
	// 	concatEndpoint = LoggingMiddleware(log.With(logger, "method", "Concat"))(concatEndpoint)
	// 	concatEndpoint = InstrumentingMiddleware(duration.With("method", "Concat"))(concatEndpoint)
	// }
	var getAvailableAgentsEndpoint endpoint.Endpoint
	{
		getAvailableAgentsEndpoint = MakeGetAvailableAgentsEndpoint(svc, session, db)
		//getAvailableAgentsEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(getAvailableAgentsEndpoint)
		//getAvailableAgentsEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(getAvailableAgentsEndpoint)
		//getAvailableAgentsEndpoint = opentracing.TraceServer(trace, "GetAvailableAgents")(getAvailableAgentsEndpoint)
		if logger != nil {
			getAvailableAgentsEndpoint = LoggingMiddleware(log.With(logger, "method", "GetAvailableAgents"))(getAvailableAgentsEndpoint)
		}
		//getAvailableAgentsEndpoint = InstrumentingMiddleware(duration.With("method", "GetAvailableAgents"))(getAvailableAgentsEndpoint)
	}
	var getAgentIDFromRefEndpoint endpoint.Endpoint
	{
		getAgentIDFromRefEndpoint = MakeGetAgentIDFromRefEndpoint(svc, session, db)
		//getAgentIDFromRefEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(getAgentIDFromRefEndpoint)
		//getAgentIDFromRefEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(getAgentIDFromRefEndpoint)
		//getAgentIDFromRefEndpoint = opentracing.TraceServer(trace, "GetAgentIDFromRef")(getAgentIDFromRefEndpoint)
		if logger != nil {
			getAgentIDFromRefEndpoint = LoggingMiddleware(log.With(logger, "method", "GetAgentIDFromRef"))(getAgentIDFromRefEndpoint)
		}
		//getAgentIDFromRefEndpoint = InstrumentingMiddleware(duration.With("method", "GetAgentIDFromRef"))(getAgentIDFromRefEndpoint)
	}
	var heartBeatEndpoint endpoint.Endpoint
	{
		heartBeatEndpoint = MakeHeartBeatEndpoint(svc, session, db)
		//heartBeatEndpoint = ratelimit.NewTokenBucketLimiter(rl.NewBucketWithRate(100, 100))(heartBeatEndpoint)
		//heartBeatEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{}))(heartBeatEndpoint)
		//heartBeatEndpoint = opentracing.TraceServer(trace, "GetAgentIDFromRef")(heartBeatEndpoint)
		if logger != nil {
			heartBeatEndpoint = LoggingMiddleware(log.With(logger, "method", "HeartBeat"))(heartBeatEndpoint)
		}
		//getAgentIDFromRefEndpoint = InstrumentingMiddleware(duration.With("method", "GetAgentIDFromRef"))(getAgentIDFromRefEndpoint)
	}
	return Set{
		//SumEndpoint:                sumEndpoint,
		//ConcatEndpoint:             concatEndpoint,
		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
		HeartBeatEndpoint:          heartBeatEndpoint,
	}
}

// Sum implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) Sum(ctx context.Context, a, b int) (int, error) {
	resp, err := s.SumEndpoint(ctx, SumRequest{A: a, B: b})
	if err != nil {
		return 0, err
	}
	response := resp.(SumResponse)
	return response.V, response.Err
}

// Concat implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) Concat(ctx context.Context, a, b string) (string, error) {
	resp, err := s.ConcatEndpoint(ctx, ConcatRequest{A: a, B: b})
	if err != nil {
		return "", err
	}
	response := resp.(ConcatResponse)
	return response.V, response.Err
}

// ------------------

// MakeSumEndpoint constructs a Sum endpoint wrapping the service.
func MakeSumEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(SumRequest)
		v, err := s.Sum(ctx, req.A, req.B)
		return SumResponse{V: v, Err: err}, nil
	}
}

// MakeConcatEndpoint constructs a Concat endpoint wrapping the service.
func MakeConcatEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(ConcatRequest)
		v, err := s.Concat(ctx, req.A, req.B)
		return ConcatResponse{V: v, Err: err}, nil
	}
}

// MakeGetAvailableAgentsEndpoint constructs a GetAvailableAgents endpoint wrapping the service.
func MakeGetAvailableAgentsEndpoint(s service.Service, session models.Session, db string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(GetAvailableAgentsRequest)
		v, err := s.GetAvailableAgents(ctx, session, db, req.Limit)
		return GetAvailableAgentsResponse{AgentIds: v, Err: err}, service.WrapError(ctx, err)
	}
}

// MakeGetAgentIDFromRefEndpoint constructs a GetAgentIDFromRef endpoint wrapping the service.
func MakeGetAgentIDFromRefEndpoint(s service.Service, session models.Session, db string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(GetAgentIDFromRefRequest)
		v, err := s.GetAgentIDFromRef(session, db, req.RefId)
		return GetAgentIDFromRefResponse{AgentId: v, Err: err}, service.WrapError(ctx, err)
	}
}

// MakeGetAgentIDFromRefEndpoint constructs a GetAgentIDFromRef endpoint wrapping the service.
func MakeHeartBeatEndpoint(s service.Service, session models.Session, db string) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(HeartBeatRequest)
		v, err := s.HeartBeat(session, db, req.AgentId)
		return HeartBeatResponse{Status: v, Message: err}, nil
	}
}

// Failer is an interface that should be implemented by response types.
// Response encoders can check if responses are Failer, and if so if they've
// failed, and if so encode them using a separate write path based on the error.
type Failer interface {
	Failed() error
}

// SumRequest collects the request parameters for the Sum method.
type SumRequest struct {
	A, B int
}

// SumResponse collects the response values for the Sum method.
type SumResponse struct {
	V   int   `json:"v"`
	Err error `json:"-"` // should be intercepted by Failed/errorEncoder
}

// Failed implements Failer.
func (r SumResponse) Failed() error { return r.Err }

// ConcatRequest collects the request parameters for the Concat method.
type ConcatRequest struct {
	A, B string
}

// ConcatResponse collects the response values for the Concat method.
type ConcatResponse struct {
	V   string `json:"v"`
	Err error  `json:"-"`
}

// Failed implements Failer.
func (r ConcatResponse) Failed() error { return r.Err }

// NOTE: json package only stringifies fields start with capital letter.
// see http://golang.org/pkg/encoding/json/
// i.e. Uses first letter caps for fields

type SayHelloRequest struct{ Name string }

type SayHelloResponse struct {
	Message string
	Err     error
}

type SayWorldRequest struct{ Name string }

type SayWorldResponse struct {
	Message string
}

// GetAvailableAgents()
type GetAvailableAgentsRequest struct {
	Limit int32
}

type GetAvailableAgentsResponse struct {
	AgentIds []string
	Err      error
}

// GetAgentIDFromRef()
type GetAgentIDFromRefRequest struct {
	RefId string
}

type GetAgentIDFromRefResponse struct {
	AgentId int32
	Err     error
}

// HeartBeat()
type HeartBeatRequest struct {
	AgentId int32
}

type HeartBeatResponse struct {
	Message error
	Status  grpc_types.HeartBeatResponse_HeartBeatStatus
}
