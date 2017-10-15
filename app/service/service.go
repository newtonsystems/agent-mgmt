package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	//"gopkg.in/mgo.v2/bson"
)

var logger = utils.GetLogger()

type nowFuncT func() time.Time

var NowFunc nowFuncT

func init() {
	NowFunc = func() time.Time {
		return time.Now()
	}
}

// Service describes a service that adds things together.
type Service interface {
	Sum(ctx context.Context, a, b int) (int, error)
	Concat(ctx context.Context, a, b string) (string, error)
	GetAvailableAgents(ctx context.Context, session models.Session, db string) ([]string, error)
	GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error)
}

// New returns a basic Service with all of the expected middlewares wired in.
func NewService(logger log.Logger, ints, chars, refs metrics.Counter) Service {

	var svc Service
	{
		svc = NewBasicService()
		svc = LoggingMiddleware(logger)(svc)
		svc = InstrumentingMiddleware(ints, chars, refs)(svc)
	}
	return svc
}

var (
	// ErrTwoZeroes is an arbitrary business rule for the Add method.
	ErrTwoZeroes = errors.New("can't sum two zeroes")

	// ErrIntOverflow protects the Add method. We've decided that this error
	// indicates a misbehaving service and should count against e.g. circuit
	// breakers. So, we return it directly in endpoints, to illustrate the
	// difference. In a real service, this probably wouldn't be the case.
	ErrIntOverflow = errors.New("integer overflow")

	// ErrMaxSizeExceeded protects the Concat method.
	ErrMaxSizeExceeded = errors.New("result exceeds maximum size")

	// ErrAgentIDNotFound if we cannot find an agent ID from an reference ID
	ErrAgentIDNotFound = errors.New("failed to find an Agent from ref id")
	//ErrAgentIDNotFound.metadata = metadata.
)

// NewBasicService returns a naïve, stateless implementation of Service.
func NewBasicService() Service {
	return basicService{}
}

type basicService struct{}

const (
	intMax = 1<<31 - 1
	intMin = -(intMax + 1)
	maxLen = 10
)

func (s basicService) Sum(_ context.Context, a, b int) (int, error) {
	if a == 0 && b == 0 {
		return 0, ErrTwoZeroes
	}
	if (b > 0 && a > (intMax-b)) || (b < 0 && a < (intMin-b)) {
		return 0, ErrIntOverflow
	}
	return a + b, nil
}

// Concat implements Service.
func (s basicService) Concat(_ context.Context, a, b string) (string, error) {
	if len(a)+len(b) > maxLen {
		return "", ErrMaxSizeExceeded
	}
	return a + b, nil
}

func TBD() {
	//err1 := c.Insert(&models.Agent{1, time.Now()})
	// if err1 != nil {
	// 	logger.Log("msg", err1)
	// }
	//c := session.DB(db).C("agents")
	//c.Insert(&models.Agent{2, time.Now()})
	// TODO: Local timezone
	//var agents []models.Agent
	//Limit(10)
	//err := c.Find(bson.M{"lastheartbeat": bson.M{"$gt": minuteAgoDate}}).All(&agents)

}

// TODO: Will need to create some sort of cleanup for the database?
func (s basicService) GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error) {
	// Get Agent ID from session data
	logger.Log("level", "debug", "msg", "Getting available agent ID from ref ID: "+refID)

	agentID, err := session.DB(db).GetAgentIDFromRef(refID)

	if agentID == 0 {
		logger.Log("level", "warn", "msg", "Failed to get agent ID from ref ID", "err", err)
		return 0, ErrAgentIDNotFound
	}

	if err != nil {
		logger.Log("level", "err", "msg", "Failed to get agent ID from ref ID", "err", err)
		return 0, err
	}

	return agentID, err
}

func (s basicService) GetAvailableAgents(_ context.Context, session models.Session, db string) ([]string, error) {
	// Find available agents from Mongo.
	// models.Agents are considered available if the heartbeat has been received in
	// the last minute (heartbeats should be every 30 secs)
	logger.Log("level", "debug", "msg", "Getting available agents from mongo")

	minuteAgoDate := NowFunc().Add(-time.Minute)
	logger.Log("level", "debug", "msg", "Getting available agents with heartbeats no older than "+minuteAgoDate.Format("01/02/2006 03:04:05"))

	var agentIDs []string
	agents, err := session.DB(db).GetAgents(minuteAgoDate)

	if err != nil {
		logger.Log("level", "err", "msg", "Failed to get agents", "err", err)
		return agentIDs, err
	}

	logger.Log("level", "info", "msg", "Found "+strconv.Itoa(len(agents))+" available agents")
	logger.Log("level", "debug", "query", fmt.Sprintf("%#v", agents))

	// get agent ids only
	for _, agent := range agents {
		agentIDs = append(agentIDs, strconv.Itoa(int(agent.AgentID)))
	}

	return agentIDs, nil
}
