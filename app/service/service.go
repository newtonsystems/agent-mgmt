package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
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
	GetAvailableAgents(ctx context.Context, session models.Session, db string, limit int32) ([]string, error)
	GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error)
	HeartBeat(session models.Session, db string, agentID int32) (grpc_types.HeartBeatResponse_HeartBeatStatus, error)
	AddTask(session models.Session, db string, custID int32, agentIDs []int32) (int32, error)
}

// Elegantly ripped from https://github.com/letsencrypt/boulder/blob/f193137405a22057fe46a1e0e27f9d1c9e07de8b/grpc/errors.go
// WrapError wraps the internal error types we use for transport across the gRPC
// layer and appends an appropriate errortype to the gRPC trailer via the provided
// context. errors.BoulderError error types are encoded using the grpc/metadata
// in the context.Context for the RPC which is considered to be the 'proper'
// method of encoding custom error types (grpc/grpc#4543 and grpc/grpc-go#478)
func WrapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if aerr, ok := err.(*amerrors.AgentMgmtError); ok {
		// Ignoring the error return here is safe because if setting the metadata
		// fails, we'll still return an error, but it will be interpreted on the
		// other side as an InternalServerError instead of a more specific one.
		logger.Log("msg", "wrapping is current working I think ", "type", int(aerr.Type))
		_ = grpc.SetTrailer(ctx, metadata.Pairs("errortype", strconv.Itoa(int(aerr.Type))))
		return grpc.Errorf(codes.Unknown, err.Error())
	}
	return grpc.Errorf(codes.Unknown, err.Error())
}

// unwrapError unwraps errors returned from gRPC client calls which were wrapped
// with wrapError to their proper internal error type. If the provided metadata
// object has an "errortype" field, that will be used to set the type of the
// error.
func UnWrapError(err error, md metadata.MD) error {
	if err == nil {
		return nil
	}
	logger.Log("level", "debug", "msg", "UnWrapError()")
	if errTypeStrs, ok := md["errortype"]; ok {

		unwrappedErr := grpc.ErrorDesc(err)
		if len(errTypeStrs) != 1 {
			return amerrors.InternalServerError(
				"multiple errorType metadata, wrapped error %q",
				unwrappedErr,
			)
		}

		errType, decErr := strconv.Atoi(errTypeStrs[0])
		if decErr != nil {
			return amerrors.InternalServerError(
				"failed to decode error type, decoding error %q, wrapped error %q",
				decErr,
				unwrappedErr,
			)
		}
		return amerrors.New(amerrors.ErrorType(errType), unwrappedErr)
	}
	logger.Log("level", "debug", "msg", "Failed to find errortype")
	return err
}

// NewService returns a basic Service with all of the expected middlewares wired in.
func NewService(logger log.Logger, metrics *Metrics) Service {

	var svc Service
	{
		svc = NewBasicService()

		if logger != nil {
			svc = LoggingMiddleware(logger)(svc)
		}

		if metrics != nil {
			svc = InstrumentingMiddleware(metrics)(svc)
		}
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
	//ErrAgentIDNotFound = errors.New("failed to find an Agent from ref id")
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

// HeartBeat() updates heartbeat for given agent id (LastHeartBeat)
func (s basicService) HeartBeat(session models.Session, db string, agentID int32) (grpc_types.HeartBeatResponse_HeartBeatStatus, error) {

	logger.Log("level", "debug", "msg", "Updating heartbeat for agent ID: "+strconv.Itoa(int(agentID)))

	exists, err := session.DB(db).AgentExists(agentID)

	if !exists {
		logger.Log("level", "err", "err", err)
		return grpc_types.HeartBeatResponse_HEARTBEAT_FAILED, err
	}

	err = session.DB(db).HeartBeat(agentID)

	if err != nil {
		logger.Log("level", "err", "msg", "Failed to update heartbeat for agent id: "+strconv.Itoa(int(agentID)), "err", err)
		return grpc_types.HeartBeatResponse_HEARTBEAT_FAILED, err
	}

	return grpc_types.HeartBeatResponse_HEARTBEAT_SUCCESSFUL, err
}

// TODO: Will need to create some sort of cleanup for the database?
func (s basicService) GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error) {
	// Get Agent ID from session data
	logger.Log("level", "debug", "msg", "Getting available agent ID from ref ID: "+refID)

	agentID, err := session.DB(db).GetAgentIDFromRef(refID)

	if agentID == 0 {
		logger.Log("level", "warn", "msg", "Failed to get agent ID from ref ID", "err", err)
		return 0, amerrors.ErrAgentIDNotFoundError("failed to find an Agent from ref id")
	}

	if err != nil {
		logger.Log("level", "err", "msg", "Failed to get agent ID from ref ID", "err", err)
		return 0, err
	}

	return agentID, err
}

func (s basicService) GetAvailableAgents(_ context.Context, session models.Session, db string, limit int32) ([]string, error) {
	// Find available agents from Mongo.
	// models.Agents are considered available if the heartbeat has been received in
	// the last minute (heartbeats should be every 30 secs)
	logger.Log("level", "debug", "msg", "Getting available agents from mongo with limit: "+strconv.Itoa(int(limit)))

	minuteAgoDate := NowFunc().Add(-time.Minute)
	logger.Log("level", "debug", "msg", "Getting available agents with heartbeats no older than "+minuteAgoDate.Format("01/02/2006 03:04:05"))

	var agentIDs []string
	agents, err := session.DB(db).GetAgents(minuteAgoDate, limit)

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

// AddTask adds a new task to the db and returns the new task's taskid
func (s basicService) AddTask(session models.Session, db string, custID int32, agentIDs []int32) (int32, error) {
	logger.Log("level", "debug", "msg", fmt.Sprintf("Adding task with custID: %d, agentIDs: %#v", custID, agentIDs))

	taskID, err := session.DB(db).AddTask(custID, agentIDs)

	if err != nil {
		logger.Log("level", "err", "msg", "Failed to add task", "err", err)
		return 0, err
	}

	return taskID, nil
}
