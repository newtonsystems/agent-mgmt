package transport

// This file provides server-side bindings for the gRPC transport.
// It utilizes the transport/grpc.Server.

import (
	"context"

	"github.com/go-kit/kit/log"
	//"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"
	oldcontext "golang.org/x/net/context"

	"github.com/newtonsystems/agent-mgmt/app/endpoint"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
)

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
	}
}

type grpcServer struct {
	getavailableagents grpctransport.Handler
}

// API Server functions defined by proto file

func (s *grpcServer) GetAvailableAgents(ctx oldcontext.Context, req *grpc_types.GetAvailableAgentsRequest) (*grpc_types.GetAvailableAgentsResponse, error) {
	_, rep, err := s.getavailableagents.ServeGRPC(ctx, req)

	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.GetAvailableAgentsResponse), nil
}

// ------------------------------------------------------------------------ //

// -- GetAvailableAgents()

func DecodeGRPCGetAvailableAgentsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	//req := grpcReq.(*grpc_types.GetAvailableAgentsRequest)
	return endpoint.GetAvailableAgentsRequest{}, nil
}

func EncodeGRPCGetAvailableAgentsResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(endpoint.GetAvailableAgentsResponse)
	return &grpc_types.GetAvailableAgentsResponse{AgentIds: resp.AgentIds}, nil
}
