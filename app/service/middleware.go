package service

import (
	"context"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"

	"github.com/newtonsystems/agent-mgmt/app/models"
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

func (mw loggingMiddleware) GetAvailableAgents(ctx context.Context, session models.Session, db string) (v []string, err error) {
	defer func() {
		mw.logger.Log("method", "GetAvailableAgents", "agent_ids", strings.Join(v, ", "), "err", err)
	}()
	return mw.next.GetAvailableAgents(ctx, session, db)
}

func (mw loggingMiddleware) GetAgentIDFromRef(session models.Session, db string, refID string) (v int32, err error) {
	defer func() {
		mw.logger.Log("method", "GetAgentIDFromRef", "agent_id", v, "err", err)
	}()
	return mw.next.GetAgentIDFromRef(session, db, refID)
}

// InstrumentingMiddleware returns a service middleware that instruments
// the number of integers summed and characters concatenated over the lifetime of
// the service.
// references asked for
func InstrumentingMiddleware(ints, chars, refs metrics.Counter) Middleware {
	return func(next Service) Service {
		return instrumentingMiddleware{
			ints:  ints,
			chars: chars,
			refs:  refs,
			next:  next,
		}
	}
}

type instrumentingMiddleware struct {
	ints  metrics.Counter
	chars metrics.Counter
	refs  metrics.Counter
	next  Service
}

func (mw instrumentingMiddleware) Sum(ctx context.Context, a, b int) (int, error) {
	v, err := mw.next.Sum(ctx, a, b)
	mw.ints.Add(float64(v))
	return v, err
}

func (mw instrumentingMiddleware) Concat(ctx context.Context, a, b string) (string, error) {
	v, err := mw.next.Concat(ctx, a, b)
	mw.chars.Add(float64(len(v)))
	return v, err
}

func (mw instrumentingMiddleware) GetAvailableAgents(ctx context.Context, session models.Session, db string) ([]string, error) {
	v, err := mw.next.GetAvailableAgents(ctx, session, db)
	mw.chars.Add(float64(len(v)))
	return v, err
}

func (mw instrumentingMiddleware) GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error) {
	v, err := mw.next.GetAgentIDFromRef(session, db, refID)
	mw.refs.Add(1)
	return v, err
}
