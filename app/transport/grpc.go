package transport

// This file provides server-side bindings for the gRPC transport.
// It utilizes the transport/grpc.Server.

import (
	"context"
	"errors"

	"github.com/go-kit/kit/log"
	//"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	oldcontext "golang.org/x/net/context"

	"github.com/newtonsystems/agent-mgmt/app/endpoint"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
)

var logger = utils.GetLogger()

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func GRPCServer(endpoints endpoint.Set, tracer stdopentracing.Tracer, logger log.Logger) grpc_types.AgentMgmtServer {
	//options := []grpctransport.ServerOption{
	//	grpctransport.ServerErrorLogger(logger),
	//}
	return &grpcServer{
		getavailableagents: grpctransport.NewServer(
			endpoints.GetAvailableAgentsEndpoint,
			DecodeGRPCGetAvailableAgentsRequest,
			EncodeGRPCGetAvailableAgentsResponse,
			//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "GetAvailableAgents", logger)))...,
		),
		getagentidfromref: grpctransport.NewServer(
			endpoints.GetAgentIDFromRefEndpoint,
			DecodeGRPCGetAgentIDFromRefRequest,
			EncodeGRPCGetAgentIDFromRefResponse,
			//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		),
		heartbeat: grpctransport.NewServer(
			endpoints.HeartBeatEndpoint,
			DecodeGRPCHeartBeatRequest,
			EncodeGRPCHeartBeatResponse,
			//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		),
		addtask: grpctransport.NewServer(
			endpoints.AddTaskEndpoint,
			DecodeGRPCAddTaskRequest,
			EncodeGRPCAddTaskResponse,
			//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		),
		//acceptcall: grpctransport.NewServer(
		//	endpoints.GetAgentIDFromRefEndpoint,
		//	DecodeGRPCGetAgentIDFromRefRequest,
		//	EncodeGRPCGetAgentIDFromRefResponse,
		//	//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		//),
	}
}

type grpcServer struct {
	getavailableagents grpctransport.Handler
	getagentidfromref  grpctransport.Handler
	acceptcall         grpctransport.Handler
	heartbeat          grpctransport.Handler
	addtask            grpctransport.Handler
}

// API Server functions defined by proto file

func (s *grpcServer) GetAvailableAgents(ctx oldcontext.Context, req *grpc_types.GetAvailableAgentsRequest) (*grpc_types.GetAvailableAgentsResponse, error) {
	_, rep, err := s.getavailableagents.ServeGRPC(ctx, req)

	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.GetAvailableAgentsResponse), nil
}

func (s *grpcServer) GetAgentIDFromRef(ctx oldcontext.Context, req *grpc_types.GetAgentIDFromRefRequest) (*grpc_types.GetAgentIDFromRefResponse, error) {
	_, rep, err := s.getagentidfromref.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.GetAgentIDFromRefResponse), nil
}

func (s *grpcServer) AcceptCall(ctx oldcontext.Context, req *grpc_types.AcceptCallRequest) (*grpc_types.AcceptCallResponse, error) {
	_, rep, err := s.acceptcall.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.AcceptCallResponse), nil
}

func (s *grpcServer) HeartBeat(ctx oldcontext.Context, req *grpc_types.HeartBeatRequest) (*grpc_types.HeartBeatResponse, error) {
	_, rep, err := s.heartbeat.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.HeartBeatResponse), nil
}

func (s *grpcServer) AddTask(ctx oldcontext.Context, req *grpc_types.AddTaskRequest) (*grpc_types.AddTaskResponse, error) {
	_, rep, err := s.addtask.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.AddTaskResponse), nil
}

// ------------------------------------------------------------------------ //

// -- GetAvailableAgents()

func DecodeGRPCGetAvailableAgentsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpc_types.GetAvailableAgentsRequest)
	return endpoint.GetAvailableAgentsRequest{Limit: req.Limit}, nil
}

func EncodeGRPCGetAvailableAgentsResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(endpoint.GetAvailableAgentsResponse)
	return &grpc_types.GetAvailableAgentsResponse{AgentIds: resp.AgentIds}, nil
}

// ------------------------------------------------------------------------ //

// GetAgentIDFromRef()

// agent mgmt service (grpc_types) -> go kit
func DecodeGRPCGetAgentIDFromRefRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpc_types.GetAgentIDFromRefRequest)
	return endpoint.GetAgentIDFromRefRequest{RefId: req.RefId}, nil
}

// go-kit -> agent mgmt service (grpc_types)
func EncodeGRPCGetAgentIDFromRefResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(endpoint.GetAgentIDFromRefResponse)
	return &grpc_types.GetAgentIDFromRefResponse{AgentId: resp.AgentId}, nil
}

// ------------------------------------------------------------------------ //

// HeartBeat()

// agent mgmt service (grpc_types) -> go kit
func DecodeGRPCHeartBeatRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	//logger.Log("level", "error", "msg", "DecodeGRPCHeartBeatRequest")
	//req := grpcReq.(*grpc_types.HeartBeatRequest)
	return endpoint.HeartBeatRequest{}, nil
}

// go-kit -> agent mgmt service (grpc_types)
func EncodeGRPCHeartBeatResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(endpoint.HeartBeatResponse)
	return &grpc_types.HeartBeatResponse{Status: resp.Status}, nil
}

// ------------------------------------------------------------------------ //

// AddTask()

// DecodeGRPCAddTaskRequest agent mgmt service (grpc_types) -> go kit
func DecodeGRPCAddTaskRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpc_types.AddTaskRequest)
	return endpoint.AddTaskRequest{CustId: req.CustId, AgentIds: req.CallIds}, nil
}

// EncodeGRPCAddTaskResponse go-kit -> agent mgmt service (grpc_types)
func EncodeGRPCAddTaskResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(endpoint.AddTaskResponse)
	return &grpc_types.AddTaskResponse{TaskId: resp.TaskId}, nil
}
