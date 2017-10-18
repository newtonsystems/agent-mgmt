package main_test

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/endpoint"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	appendpoint "github.com/newtonsystems/agent-mgmt/app/endpoint"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/agent-mgmt/app/tests"
	"github.com/newtonsystems/agent-mgmt/app/transport"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var update = flag.Bool("update", false, "update golden files")
var debug = flag.Bool("debug", false, "turn on mongo debug")

var logger = utils.GetLogger()

const (
	dataDir = "./testdata"
)

type entry struct {
	source      string
	golden      string
	description string
}

var data = []entry{
	{
		//"getavailableagents",
		//"",
		"getavailableagents.input",
		"getavailableagents.golden",
		"A basic test of service's GetAvailableAgents()",
		//nil,
	},
}

const (
	hostPort        string = ":8004"
	mongoDBDatabase string = "test"
)

func createTestServer(t *testing.T) {

	// Initialise mongo connection
	moSession := tests.CreateTestMongoConnection(*debug)
	defer moSession.Refresh()
	defer moSession.Close()

	// Create Service &  Endpoints (no logger, tracer, metrics etc)
	var (
		service   = service.NewService(nil, nil, nil, nil, nil)
		endpoints = appendpoint.NewEndpoint(service, nil, nil, nil, moSession, "test")
	)

	// gRPC server
	go func() {
		ln, err := net.Listen("tcp", hostPort)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		srv := transport.GRPCServer(endpoints, nil, nil)
		s := grpc.NewServer()
		grpc_types.RegisterAgentMgmtServer(s, srv)
		//defer s.GracefulStop()

		s.Serve(ln)
	}()

}

func createTestClient(t *testing.T, conn *grpc.ClientConn) service.Service {
	var getAvailableAgentsEndpoint endpoint.Endpoint
	{
		getAvailableAgentsEndpoint = grpctransport.NewClient(
			conn, "grpc_types.AgentMgmt", "GetAvailableAgents",
			transport.EncodeGRPCGetAvailableAgentsRequest,
			transport.DecodeGRPCGetAvailableAgentsResponse,
			grpc_types.GetAvailableAgentsResponse{},
		).Endpoint()
	}
	var getAgentIDFromRefEndpoint endpoint.Endpoint
	{
		getAgentIDFromRefEndpoint = grpctransport.NewClient(
			conn, "grpc_types.AgentMgmt", "GetAgentIDFromRef",
			transport.EncodeGRPCGetAvailableAgentsRequest,
			transport.DecodeGRPCGetAvailableAgentsResponse,
			grpc_types.GetAgentIDFromRefResponse{},
		).Endpoint()
	}

	return appendpoint.Set{
		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
	}
}

func TestGRPCClient(t *testing.T) {

	// Initialise mongo connection
	moSession := tests.CreateTestMongoConnection(*debug)
	defer moSession.Refresh()
	defer moSession.Close()

	// Create Service &  Endpoints (no logger, tracer, metrics etc)
	var (
		service   = service.NewService(nil, nil, nil, nil, nil)
		endpoints = appendpoint.NewEndpoint(service, nil, nil, nil, moSession, "test")
	)

	// gRPC server
	go func() {
		ln, err := net.Listen("tcp", hostPort)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		srv := transport.GRPCServer(endpoints, nil, nil)
		s := grpc.NewServer()
		grpc_types.RegisterAgentMgmtServer(s, srv)
		//defer s.GracefulStop()

		s.Serve(ln)
	}()

	// Connection to grpc server and create a client

	conn, err := grpc.Dial(hostPort, grpc.WithInsecure())
	defer conn.Close()
	if err != nil {
		t.Fatalf("unable to Dial: %+v", err)
		t.FailNow()
	}

	clientService := createTestClient(t, conn)

	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			checkClient(t, moSession, clientService, source, golden, e.description)
		})
		//clearAgentsCollection(moSession)
	}

	// var (
	// 	a   = "the answer to life the universe and everything"
	// 	b   = int64(42)
	// 	cID = "request-1"
	// 	ctx = test.SetCorrelationID(context.Background(), cID)
	// )
	//
	// responseCTX, v, err := client.Test(ctx, a, b)
	// if err != nil {
	// 	t.Fatalf("unable to Test: %+v", err)
	// }
	// if want, have := fmt.Sprintf("%s = %d", a, b), v; want != have {
	// 	t.Fatalf("want %q, have %q", want, have)
	// }
	//
	// if want, have := cID, test.GetConsumedCorrelationID(responseCTX); want != have {
	// 	t.Fatalf("want %q, have %q", want, have)
	// }

}

func checkClient(t *testing.T, session models.Session, client service.Service, source, golden string, description string) {
	_, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	ctx := context.Background()
	resp, err := client.GetAvailableAgents(ctx, session, mongoDBDatabase)
	logger.Log("resp", fmt.Sprintf("\n%#v", resp), "err", err)
	if err != nil {
		logger.Log("msg", "err is not nil")
		s, ok := status.FromError(err)
		if !ok {
			logger.Log("msg", status.Errorf(codes.Internal, "client call: unknown %v", err))
		}
		switch s.Code() {
		case codes.Unknown:

			logger.Log("msg", status.Errorf(codes.Internal, "client call: invalid argyemet %v", s), "errortype", ctx.Value(contextKey("errortype")))
		}
	}
}
