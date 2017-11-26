// Some notes on why the code is writtent the way it is


``` go


clientService := createTestClient(t, conn)


	resp, err := client.GetAvailableAgents(ctx, session, mongoDBDatabase) //	grpc.Header(&header),    // will retrieve header
	//grpc.Trailer(&trailer),  // will retrieve trailer)


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

	return amendpoint.Set{
		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
	}
}
```


Useful client code maybe

```go
func checkClient(t *testing.T, session models.Session, client grpc_types.AgentMgmtClient, source, golden string, description string) {
	_, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	var header, trailer metadata.MD
	ctx := context.Background()

	resp, err := client.GetAvailableAgents(
		ctx,
		&grpc_types.GetAvailableAgentsRequest{Limit: 10},
		grpc.Header(&header),   // will retrieve header
		grpc.Trailer(&trailer), // will retrieve trailer
	)

	//md, ok := metadata.FromIncomingContext(ctx)
	if resp != nil {
		logger.Log("resp", fmt.Sprintf("\n%#v", resp.AgentIds), "err", err, "header", fmt.Sprintf("\n%#v", header), "trailer", fmt.Sprintf("\n%#v", trailer))
	}
	if err != nil {
		logger.Log("msg", "err is not nil", "header", fmt.Sprintf("\n%#v", header), "trailer", fmt.Sprintf("\n%#v", trailer))
		if val, ok := trailer["errortype"]; ok {
			//do something here

			logger.Log("msg", "errortype found key")
			logger.Log("msg", val[0])
			errortype, err := strconv.Atoi(val[0])
			errortype2 := amerrors.ErrorType(errortype)
			if err != nil {
				logger.Log("msg", "failed conversitony")
			}
			switch errortype2 {
			case amerrors.ErrAgentIDNotFound:
				logger.Log("msg", "ErrAgentIDNotFoundError")

			}
		}
		s, ok := status.FromError(err)
		if !ok {
			logger.Log("msg", status.Errorf(codes.Internal, "client call: unknown %v", err))
		}
		switch s.Code() {
		case codes.Unknown:

			logger.Log("msg", status.Errorf(codes.Internal, "client call: invalid argyemet %v", s), "errortype", "hj")
		}
	}
}
```

useful create a server and a sprivifc gokit type client reequires endpoint as a service


```go
// createTestServer
func createTestServer(t *testing.T) {
	// Initialise mongo connection
	moSession := tests.CreateTestMongoConnection(*debug)
	defer moSession.Refresh()
	defer moSession.Close()

	// Create Service &  Endpoints (no logger, tracer, metrics etc)
	var (
		service   = service.NewService(nil, nil, nil, nil, nil)
		endpoints = amendpoint.NewEndpoint(service, nil, nil, nil, moSession, "test")
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

	return amendpoint.Set{
		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
	}
}


// in endpoints
// GetAvailableAgentsRequest implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) GetAvailableAgents(ctx context.Context, session models.Session, db string) ([]string, error) {
	resp, err := s.GetAvailableAgentsEndpoint(ctx, GetAvailableAgentsRequest{Limit: 10})
	if err != nil {
		var empty []string
		return empty, err
	}
	response := resp.(GetAvailableAgentsResponse)
	return response.AgentIds, err
}

// GetAgentIDFromRef implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) GetAgentIDFromRef(sess models.Session, db string, refID string) (int32, error) {
	resp, _ := s.GetAgentIDFromRefEndpoint(nil, GetAgentIDFromRefRequest{RefId: refID})
	response := resp.(GetAgentIDFromRefResponse)
	return response.AgentId, response.Err
}

// HeartBeat implements the service interface, so Set may be used as a
// service. This is primarily useful in the context of a client library.
func (s Set) HeartBeat(session models.Session, db string, agent models.Agent) (grpc_types.HeartBeatResponse_HeartBeatStatus, error) {
	resp, _ := s.HeartBeatEndpoint(nil, HeartBeatRequest{Agent: agent})
	response := resp.(HeartBeatResponse)
	return response.Status, response.Message
}


// -- grpc.go

// -- GetAvailableAgents() Client functions

// EncodeGRPCGetAvailableAgentsRequest is a transport/grpc.EncodeRequestFunc that converts a
// user-domain sum request to a gRPC sum request. Primarily useful in a client.
func EncodeGRPCGetAvailableAgentsRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(endpoint.GetAvailableAgentsRequest)
	return &grpc_types.GetAvailableAgentsRequest{Limit: int32(req.Limit)}, nil
}

// DecodeGRPCGetAvailableAgentsResponse is a transport/grpc.DecodeResponseFunc that converts a
// gRPC sum reply to a user-domain sum response. Primarily useful in a client.
func DecodeGRPCGetAvailableAgentsResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*grpc_types.GetAvailableAgentsResponse)
	return endpoint.GetAvailableAgentsResponse{AgentIds: reply.AgentIds}, nil //grpc.Errorf(codes.InvalidArgument, "Ouch!") //, Err: str2err(reply.Err)}, nil
}

```




// package main_test
//
// import (
// 	"context"
// 	"encoding/json"
// 	"flag"
// 	"fmt"
// 	"io/ioutil"
// 	"net"
// 	"path/filepath"
// 	"runtime"
// 	"strconv"
// 	"strings"
// 	"testing"
// 	"time"
//
// 	"github.com/go-kit/kit/endpoint"
// 	grpctransport "github.com/go-kit/kit/transport/grpc"
// 	amendpoint "github.com/newtonsystems/agent-mgmt/app/endpoint"
// 	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
// 	"github.com/newtonsystems/agent-mgmt/app/models"
// 	"github.com/newtonsystems/agent-mgmt/app/service"
// 	"github.com/newtonsystems/agent-mgmt/app/tests"
// 	"github.com/newtonsystems/agent-mgmt/app/transport"
// 	"github.com/newtonsystems/agent-mgmt/app/utils"
// 	"github.com/newtonsystems/grpc_types/go/grpc_types"
// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/codes"
// 	"google.golang.org/grpc/metadata"
// 	"google.golang.org/grpc/status"
// )
//
// var update = flag.Bool("update", false, "update golden files")
// var debug = flag.Bool("debug", false, "turn on mongo debug")
//
// var logger = utils.GetLogger()
//
// const (
// 	dataDir     = "./testdata"
// 	mongoDBName = "test"
// )
//
// type testRequest interface {
// }
//
// type entry struct {
// 	testName    string
// 	testArgs    testRequest
// 	testHasErr  error
// 	source      string
// 	golden      string
// 	description string
// }
//
// const (
// 	hostPort        string = ":8004"
// 	mongoDBDatabase string = "test"
// )
//
// var data = []entry{
// 	{
// 		"getavailableagents",
// 		&grpc_types.GetAvailableAgentsRequest{},
// 		nil,
// 		"getavailableagents.input",
// 		"getavailableagents.golden",
// 		"A basic test of service's GetAvailableAgents()",
// 	},
// }
//
// func createTestServer(t *testing.T) {
//
// 	// Initialise mongo connection
// 	moSession := tests.CreateTestMongoConnection(*debug)
// 	defer moSession.Refresh()
// 	defer moSession.Close()
//
// 	// Create Service &  Endpoints (no logger, tracer, metrics etc)
// 	var (
// 		service   = service.NewService(nil, nil, nil, nil, nil)
// 		endpoints = amendpoint.NewEndpoint(service, nil, nil, nil, moSession, "test")
// 	)
//
// 	// gRPC server
// 	go func() {
// 		ln, err := net.Listen("tcp", hostPort)
// 		if err != nil {
// 			t.Error(err)
// 			t.FailNow()
// 		}
//
// 		srv := transport.GRPCServer(endpoints, nil, nil)
// 		s := grpc.NewServer()
// 		grpc_types.RegisterAgentMgmtServer(s, srv)
// 		//defer s.GracefulStop()
//
// 		s.Serve(ln)
// 	}()
//
// }
//
// func createTestClient(t *testing.T, conn *grpc.ClientConn) service.Service {
// 	var getAvailableAgentsEndpoint endpoint.Endpoint
// 	{
// 		getAvailableAgentsEndpoint = grpctransport.NewClient(
// 			conn, "grpc_types.AgentMgmt", "GetAvailableAgents",
// 			transport.EncodeGRPCGetAvailableAgentsRequest,
// 			transport.DecodeGRPCGetAvailableAgentsResponse,
// 			grpc_types.GetAvailableAgentsResponse{},
// 		).Endpoint()
// 	}
// 	var getAgentIDFromRefEndpoint endpoint.Endpoint
// 	{
// 		getAgentIDFromRefEndpoint = grpctransport.NewClient(
// 			conn, "grpc_types.AgentMgmt", "GetAgentIDFromRef",
// 			transport.EncodeGRPCGetAvailableAgentsRequest,
// 			transport.DecodeGRPCGetAvailableAgentsResponse,
// 			grpc_types.GetAgentIDFromRefResponse{},
// 		).Endpoint()
// 	}
//
// 	return amendpoint.Set{
// 		GetAvailableAgentsEndpoint: getAvailableAgentsEndpoint,
// 		GetAgentIDFromRefEndpoint:  getAgentIDFromRefEndpoint,
// 	}
// }
//
// func TestGRPCClient(t *testing.T) {
// 	// Freeze Time
// 	service.NowFunc = func() time.Time {
// 		freezeTime := time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)
// 		logger.Log("level", "debug", "msg", "The time is "+freezeTime.Format("01/02/2006 03:04:05"))
// 		return freezeTime
// 	}
//
// 	// Initialise mongo connection
// 	moSession := tests.CreateTestMongoConnection(*debug)
// 	defer moSession.Refresh()
// 	defer moSession.Close()
//
// 	// Create Service &  Endpoints (no logger, tracer, metrics etc)
// 	var (
// 		service   = service.NewService(nil, nil, nil, nil, nil)
// 		endpoints = amendpoint.NewEndpoint(service, nil, nil, nil, moSession, "test")
// 	)
//
// 	service.GetAvailableAgents = func(ctx context.Context, session models.Session, db string) ([]string, error) {
// 		var agentIDs []string
// 		return agentIDs, amerrors.ErrAgentIDNotFoundError("")
// 	}
//
// 	// gRPC server
// 	go func() {
// 		ln, err := net.Listen("tcp", hostPort)
// 		if err != nil {
// 			t.Error(err)
// 			t.FailNow()
// 		}
//
// 		srv := transport.GRPCServer(endpoints, nil, nil)
// 		s := grpc.NewServer()
// 		grpc_types.RegisterAgentMgmtServer(s, srv)
// 		//defer s.GracefulStop()
//
// 		s.Serve(ln)
// 	}()
//
// 	// Connection to grpc server and create a client
//
// 	conn, err := grpc.Dial(hostPort, grpc.WithInsecure())
//
// 	defer conn.Close()
// 	if err != nil {
// 		t.Fatalf("unable to Dial: %+v", err)
// 		t.FailNow()
// 	}
//
// 	client := grpc_types.NewAgentMgmtClient(conn)
// 	for _, e := range data {
// 		source := filepath.Join(dataDir, e.source)
// 		golden := filepath.Join(dataDir, e.golden)
// 		t.Run(e.source, func(t *testing.T) {
// 			logger.Log("msg", "Running service test for "+e.testName)
// 			checkClientCall(t, client, moSession, source, golden, e.description, e.testName, e.testArgs, e.testHasErr)
// 		})
// 		cleanUp(moSession)
// 	}
//
// 	// var (
// 	// 	a   = "the answer to life the universe and everything"
// 	// 	b   = int64(42)
// 	// 	cID = "request-1"
// 	// 	ctx = test.SetCorrelationID(context.Background(), cID)
// 	// )
// 	//
// 	// responseCTX, v, err := client.Test(ctx, a, b)
// 	// if err != nil {
// 	// 	t.Fatalf("unable to Test: %+v", err)
// 	// }
// 	// if want, have := fmt.Sprintf("%s = %d", a, b), v; want != have {
// 	// 	t.Fatalf("want %q, have %q", want, have)
// 	// }
// 	//
// 	// if want, have := cID, test.GetConsumedCorrelationID(responseCTX); want != have {
// 	// 	t.Fatalf("want %q, have %q", want, have)
// 	// }
//
// }
//
// // cleanUp removes everyfrom the database including all collections
// func cleanUp(session models.Session) {
// 	session.DB(mongoDBName).DropDatabase()
// }
//
// // Convert to bytes for possible writing
// func runSrvTest(t *testing.T, client grpc_types.AgentMgmtClient, header, trailer metadata.MD, testName string, testArgs testRequest) ([]byte, error) {
// 	var res []byte
// 	var resErr error
// 	ctx := context.Background()
//
// 	switch testName {
// 	case "getavailableagents":
// 		request, ok := testArgs.(*grpc_types.GetAvailableAgentsRequest)
// 		if !ok {
// 			t.Error("Failed to convert request. This shouldnt happen ...")
// 			t.FailNow()
// 		}
// 		resp, err := client.GetAvailableAgents(
// 			ctx,
// 			request,
// 			grpc.Header(&header),
// 			grpc.Trailer(&trailer),
// 		)
// 		if err != nil {
// 			//res = []byte()
// 			resErr = err
// 			logger.Log("msg", fmt.Sprintf("\n%#v", resp))
//
// 		} else {
// 			res = []byte(strings.Join(resp.AgentIds, ", "))
// 			resErr = err
// 		}
//
// 	case "getagentidfromref":
// 		resp, err := client.GetAgentIDFromRef(
// 			ctx,
// 			&grpc_types.GetAgentIDFromRefRequest{RefId: "hsajdhjas"},
// 			grpc.Header(&header),
// 			grpc.Trailer(&trailer),
// 		)
// 		res = []byte(strconv.Itoa(int(resp.AgentId)))
// 		resErr = err
//
// 	case "heartbeat":
// 		resp, err := client.HeartBeat(
// 			ctx,
// 			&grpc_types.HeartBeatRequest{},
// 			grpc.Header(&header),
// 			grpc.Trailer(&trailer),
// 		)
// 		res = []byte(strconv.Itoa(int(resp.Status)))
// 		resErr = err
//
// 	}
//
// 	return res, resErr
// }
//
// // Unmarshal JSON From File
// func insertFixtureToDatabase(t *testing.T, session models.Session, testName, source string, src []byte) {
// 	logger.Log("msg", "insert input data into mongo")
// 	switch testName {
// 	case "getavailableagents":
// 		var agents []models.Agent
// 		json.Unmarshal(src, &agents)
//
// 		// Check we have found some input
// 		if len(agents) == 0 {
// 			var errMessage = "No input data found from " + source
// 			_, file, line, _ := runtime.Caller(1)
// 			logger.Log("info", "crit", "msg", errMessage)
// 			fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, errMessage)
// 			t.FailNow()
// 		}
//
// 		// Insert into mongo
// 		for _, agent := range agents {
// 			err1 := session.DB("test").C("agents").Insert(agent)
// 			if err1 != nil {
// 				logger.Log("msg", "Could not insert input into mongo", "err", err1)
// 				t.Error(err1)
// 			}
// 		}
//
// 	case "getagentidfromref":
// 		var phoneSessions []models.PhoneSession
// 		json.Unmarshal(src, &phoneSessions)
//
// 	case "heartbeat":
//
// 	}
// }
//
// func checkClientCall(t *testing.T, client grpc_types.AgentMgmtClient, session models.Session, source, golden, description, testName string, testArgs testRequest, testHasErr error) {
// 	// read input from file
// 	src, err := ioutil.ReadFile(source)
//
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
//
// 	// update mongo db with input data
// 	insertFixtureToDatabase(t, session, testName, source, src)
//
// 	// run service call
// 	var header, trailer metadata.MD
// 	res, err := runSrvTest(t, client, header, trailer, testName, testArgs)
//
// 	// is an error is expected? If so, we check it is the correct one
// 	if err != nil {
// 		if testHasErr != nil && err != testHasErr {
// 			t.Error(err)
// 			t.FailNow()
// 		}
// 	}
//
// 	// update golden files if necessary
// 	if *update {
// 		if werr := ioutil.WriteFile(golden, res, 0644); werr != nil {
// 			t.Error(err)
// 		}
// 		return
// 	}
//
// 	// get golden
// 	gld, err := ioutil.ReadFile(golden)
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
//
// 	// formatted source and golden must be the same
// 	if err := tests.Diff(source, golden, description, res, gld); err != nil {
// 		t.Error(err)
// 		return
// 	}
//
// }
//
// func checkClient(t *testing.T, session models.Session, client grpc_types.AgentMgmtClient, source, golden string, description string) {
// 	_, err := ioutil.ReadFile(source)
//
// 	if err != nil {
// 		t.Error(err)
// 		return
// 	}
//
// 	var header, trailer metadata.MD
// 	ctx := context.Background()
//
// 	resp, err := client.GetAvailableAgents(
// 		ctx,
// 		&grpc_types.GetAvailableAgentsRequest{Limit: 10},
// 		grpc.Header(&header),   // will retrieve header
// 		grpc.Trailer(&trailer), // will retrieve trailer
// 	)
//
// 	//md, ok := metadata.FromIncomingContext(ctx)
// 	if resp != nil {
// 		logger.Log("resp", fmt.Sprintf("\n%#v", resp.AgentIds), "err", err, "header", fmt.Sprintf("\n%#v", header), "trailer", fmt.Sprintf("\n%#v", trailer))
// 	}
// 	if err != nil {
// 		logger.Log("msg", "err is not nil", "header", fmt.Sprintf("\n%#v", header), "trailer", fmt.Sprintf("\n%#v", trailer))
// 		if val, ok := trailer["errortype"]; ok {
// 			//do something here
//
// 			logger.Log("msg", "errortype found key")
// 			logger.Log("msg", val[0])
// 			errortype, err := strconv.Atoi(val[0])
// 			errortype2 := amerrors.ErrorType(errortype)
// 			if err != nil {
// 				logger.Log("msg", "failed conversitony")
// 			}
// 			switch errortype2 {
// 			case amerrors.ErrAgentIDNotFound:
// 				logger.Log("msg", "ErrAgentIDNotFoundError")
//
// 			}
// 		}
// 		s, ok := status.FromError(err)
// 		if !ok {
// 			logger.Log("msg", status.Errorf(codes.Internal, "client call: unknown %v", err))
// 		}
// 		switch s.Code() {
// 		case codes.Unknown:
//
// 			logger.Log("msg", status.Errorf(codes.Internal, "client call: invalid argyemet %v", s), "errortype", "hj")
// 		}
// 	}
// }
