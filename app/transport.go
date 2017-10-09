package main

// This file provides server-side bindings for the gRPC transport.
// It utilizes the transport/grpc.Server.

import (
	"context"

	"github.com/go-kit/kit/log"
	oldcontext "golang.org/x/net/context"
	//"github.com/go-kit/kit/tracing/opentracing"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	stdopentracing "github.com/opentracing/opentracing-go"

	"github.com/newtonsystems/grpc_types/go/grpc_types"
)

func MakeAllServicesGRPCServer(endpoints Endpoints, tracer stdopentracing.Tracer, logger log.Logger) grpc_types.GlobalAPIServer {
	//options := []grpctransport.ServerOption{
	//	grpctransport.ServerErrorLogger(logger),
	//}
	return &grpcAllServicesServer{
		sayhello: grpctransport.NewServer(
			endpoints.SayHelloEndpoint,
			DecodeGRPCSayHelloRequest,
			EncodeGRPCSayHelloResponse,
			//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		),
		sayworld: grpctransport.NewServer(
			endpoints.SayWorldEndpoint,
			DecodeGRPCSayHelloRequest,
			EncodeGRPCSayHelloResponse,
			//append(options, grpctransport.ServerBefore(opentracing.FromGRPCRequest(tracer, "Sum", logger)))...,
		),
	}
}

type grpcAllServicesServer struct {
	sayhello grpctransport.Handler
	sayworld grpctransport.Handler
}

// Convert internal grpc type to grpc_types (GlobalAPI)
//
//
func (s *grpcAllServicesServer) SayHello(ctx oldcontext.Context, req *grpc_types.HelloRequest) (*grpc_types.HelloResponse, error) {
	_, rep, err := s.sayhello.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*grpc_types.HelloResponse), nil
}

// Decode SayHello response i.e from hello service to go-kit structure endpoint
func DecodeGRPCSayHelloResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*grpc_types.HelloResponse)
	return sayHelloResponse{Message: reply.Message}, nil
}

// go-kit -> hello request service
// Encode from go-kit request to hello service message

// -- Hello Service

func DecodeGRPCSayHelloRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*grpc_types.HelloRequest)
	return sayHelloRequest{Name: req.Name}, nil
}

func EncodeGRPCSayHelloResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(sayHelloResponse)
	return &grpc_types.HelloResponse{Message: resp.Message}, nil
}
