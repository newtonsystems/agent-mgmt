package service

import (
	"context"
	"strings"

	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
)

// Middleware describes a service (as opposed to endpoint) middleware.
type Middleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a ServiceMiddleware.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) Sum(ctx context.Context, a, b int) (v int, err error) {
	defer func() {
		mw.logger.Log("method", "Sum", "a", a, "b", b, "v", v, "err", err)
	}()
	return mw.next.Sum(ctx, a, b)
}

func (mw loggingMiddleware) Concat(ctx context.Context, a, b string) (v string, err error) {
	defer func() {
		mw.logger.Log("method", "Concat", "a", a, "b", b, "v", v, "err", err)
	}()
	return mw.next.Concat(ctx, a, b)
}

func (mw loggingMiddleware) GetAvailableAgents(ctx context.Context, session models.Session, db string, limit int32) (v []string, err error) {
	defer func() {
		mw.logger.Log("method", "GetAvailableAgents", "agent_ids", strings.Join(v, ", "), "err", err)
	}()
	return mw.next.GetAvailableAgents(ctx, session, db, limit)
}

func (mw loggingMiddleware) GetAgentIDFromRef(session models.Session, db string, refID string) (v int32, err error) {
	defer func() {
		mw.logger.Log("method", "GetAgentIDFromRef", "agent_id", v, "err", err)
	}()
	return mw.next.GetAgentIDFromRef(session, db, refID)
}

func (mw loggingMiddleware) HeartBeat(session models.Session, db string, agentID int32) (status grpc_types.HeartBeatResponse_HeartBeatStatus, err error) {
	defer func() {
		mw.logger.Log("method", "HeartBeat", "agent_id", agentID, "status", status)
	}()
	return mw.next.HeartBeat(session, db, agentID)
}

func (mw loggingMiddleware) AddTask(session models.Session, db string, custID int32, agentIDs []int32) (taskID int32, err error) {
	defer func() {
		mw.logger.Log("method", "AddTask", "cust_id", custID, "call_ids", agentIDs, "task_id", taskID, "err", err)
	}()
	return mw.next.AddTask(session, db, custID, agentIDs)
}

func NewMetrics() Metrics {
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

	return Metrics{
		Ints:     ints,
		Chars:    chars,
		Refs:     refs,
		Beats:    beats,
		Duration: duration,
		next:     nil,
	}

}

// Metrics returns a service middleware that instruments
// the number of integers summed and characters concatenated over the lifetime of
// the service.
// references asked for
// The number of heartbeats counted ()
func InstrumentingMiddleware(metrics *Metrics) Middleware {
	return func(next Service) Service {
		return Metrics{
			Ints:     metrics.Ints,
			Chars:    metrics.Chars,
			Refs:     metrics.Refs,
			Beats:    metrics.Beats,
			Duration: metrics.Duration,
			next:     next,
		}
	}
}

type Metrics struct {
	Ints     metrics.Counter
	Chars    metrics.Counter
	Refs     metrics.Counter
	Beats    metrics.Counter
	Addtasks metrics.Counter
	Duration metrics.Histogram
	next     Service
}

func (mw Metrics) Sum(ctx context.Context, a, b int) (int, error) {
	v, err := mw.next.Sum(ctx, a, b)
	mw.Ints.Add(float64(v))
	return v, err
}

func (mw Metrics) Concat(ctx context.Context, a, b string) (string, error) {
	v, err := mw.next.Concat(ctx, a, b)
	mw.Chars.Add(float64(len(v)))
	return v, err
}

func (mw Metrics) GetAvailableAgents(ctx context.Context, session models.Session, db string, limit int32) ([]string, error) {
	v, err := mw.next.GetAvailableAgents(ctx, session, db, limit)
	mw.Chars.Add(float64(len(v)))
	return v, err
}

func (mw Metrics) GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error) {
	v, err := mw.next.GetAgentIDFromRef(session, db, refID)
	mw.Refs.Add(1)
	return v, err
}

func (mw Metrics) HeartBeat(session models.Session, db string, agentID int32) (grpc_types.HeartBeatResponse_HeartBeatStatus, error) {
	status, err := mw.next.HeartBeat(session, db, agentID)
	mw.Beats.Add(1)
	return status, err
}

func (mw Metrics) AddTask(session models.Session, db string, custID int32, agentIDs []int32) (int32, error) {
	status, err := mw.next.AddTask(session, db, custID, agentIDs)
	mw.Addtasks.Add(1)
	return status, err
}
