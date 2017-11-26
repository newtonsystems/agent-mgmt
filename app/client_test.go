package main_test

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	amendpoint "github.com/newtonsystems/agent-mgmt/app/endpoint"
	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	tu "github.com/newtonsystems/agent-mgmt/app/testutil"
	"github.com/newtonsystems/agent-mgmt/app/transport"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	mgo "gopkg.in/mgo.v2"
)

var logger = utils.GetLogger()

const (
	dataDir = "./testdata"
)

// An interface so we can encode different grpc requests
type testRequest interface {
}

type entry struct {
	testName    string             // An identifier test name e.g. getavailableagents
	testReq     testRequest        // The grpc request e.g. &grpc_types.GetAvailableAgentsRequest{Limit: 10},
	testHasErr  amerrors.ErrorType // The error expected by service call. Nil if no error is expected by the rpc call
	source      string             // The source file that contains data to be inserted into mongo
	compare     string             // A description of what we compare against the golden
	golden      string             // The golden file
	description string             // A useful description of what the test intends to accomplish
}

const (
	hostPort        string = ":8004"
	mongoDBDatabase string = "test"
)

var data = []entry{
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{},
		0,
		"getavailableagents.input",
		"response agent IDs",
		"getavailableagents.golden",
		"A basic test of service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{},
		0,
		"getavailableagents_oldheartbeat.input",
		"response agent IDs",
		"getavailableagents_oldheartbeat.golden",
		"A test to ensure heartbeats older than one minute are not included as available agents by service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{},
		0,
		"getavailableagents_futureheartbeat.input",
		"response agent IDs",
		"getavailableagents_futureheartbeat.golden",
		"A test to ensure heartbeats newer than one minute are included as available agents by service's GetAvailableAgents()  (We accept future timestamps)",
	},
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{},
		0,
		"getavailableagents_minuteagoexactly.input",
		"response agent IDs",
		"getavailableagents_minuteagoexactly.golden",
		"A test to ensure a heartbeat exactly a minute ago is included as an available agent by service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{Limit: 10},
		0,
		"getavailableagents_limit_results_10.input",
		"response agent IDs",
		"getavailableagents_limit_results_10.golden",
		"A test to check there is a limit to the available agent ids returned by service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{Limit: 3},
		0,
		"getavailableagents_limit.input",
		"response agent IDs",
		"getavailableagents_limit.golden",
		"A test to check 'Limit' request argument works for service's GetAvailableAgents()",
	},
	{
		"getagentidfromref",
		&grpc_types.GetAgentIDFromRefRequest{RefId: "ref001a"},
		0,
		"getagentidfromref.input",
		"response agent ID",
		"getagentidfromref.golden",
		"A basic test of service's GetAgentIDFromRef()",
	},
	{
		"getagentidfromref",
		&grpc_types.GetAgentIDFromRefRequest{RefId: ""},
		amerrors.ErrAgentIDNotFound,
		"getagentidfromref_empty.input",
		"response agent ID",
		"getagentidfromref_empty.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is empty returned by service's GetAgentIDFromRef()",
	},
	{
		"getagentidfromref",
		&grpc_types.GetAgentIDFromRefRequest{RefId: "refwrong"},
		amerrors.ErrAgentIDNotFound,
		"getagentidfromref_wrongref.input",
		"response agent ID",
		"getagentidfromref_wrongref.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is incorrect returned by service's GetAgentIDFromRef()",
	},
	{
		"heartbeat",
		&grpc_types.HeartBeatRequest{AgentId: 3},
		0,
		"heartbeat.input",
		"response heartbeat status",
		"heartbeat.golden",
		"A basic test of service's HeartBeat()",
	},
	{
		"heartbeat",
		&grpc_types.HeartBeatRequest{AgentId: 32},
		amerrors.ErrAgentIDNotFound,
		"heartbeat.input",
		"response heartbeat status",
		"heartbeat_wrongagentid.golden",
		"A test to check correct error if the agent does not exist given agent id provided to service's HeartBeat()",
	},
	{
		"heartbeat",
		&grpc_types.HeartBeatRequest{},
		amerrors.ErrAgentIDNotFound,
		"heartbeat.input",
		"response heartbeat status",
		"heartbeat_noagentidinrequest.golden",
		"A test to check correct error if no agent id is provided to service's HeartBeat()",
	},
	{
		"addtask",
		&grpc_types.AddTaskRequest{CustId: 1, CallIds: []int32{1, 2, 3}},
		0,
		"addtask.input",
		"response taskid",
		"addtask.golden",
		"A basic test of service's AddTask()",
	},
	{
		"addtask",
		&grpc_types.AddTaskRequest{},
		amerrors.ErrCustIDInvalid,
		"addtask.input",
		"response taskid",
		"addtask_empty.golden",
		"A test to check of invalid custid of 0 (via empty request) for service's AddTask()",
	},
	{
		"addtask",
		&grpc_types.AddTaskRequest{CustId: 0},
		amerrors.ErrCustIDInvalid,
		"addtask.input",
		"response taskid",
		"addtask_custid0.golden",
		"A test to check of invalid custid of 0 for service's AddTask()",
	},
}

type entryQueryError struct {
	testName    string      // An identifier test name e.g. getavailableagents
	testReq     testRequest // The grpc request e.g. &grpc_types.GetAvailableAgentsRequest{Limit: 10},
	description string      // A useful description of what the test intends to accomplish
}

var dataMockErrors = []entryQueryError{
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{},
		"A basic QueryError test of service's GetAvailableAgents()",
	},
	{
		"getagentidfromref",
		&grpc_types.GetAgentIDFromRefRequest{},
		"A basic QueryError test of service's GetAgentIDFromRef()",
	},
	{
		"addtask",
		&grpc_types.AddTaskRequest{},
		"A basic QueryError test of service's AddTask()",
	},
}

// runSrvTest runs a specifc test based off testName we convert to bytes for possible writing
func runSrvTest(t *testing.T, client grpc_types.AgentMgmtClient, header, trailer *metadata.MD, testName string, testReq testRequest) ([]byte, error) {
	var res []byte
	var resErr error
	ctx := context.Background()

	switch testName {
	case "getavailableagents":
		request, ok := testReq.(*grpc_types.GetAvailableAgentsRequest)
		if !ok {
			tu.FailNowAt(t, "Failed to convert/decode request. This shouldnt happen ...")
		}
		resp, err := client.GetAvailableAgents(
			ctx,
			request,
			grpc.Header(header),
			grpc.Trailer(trailer),
		)
		// Style: this doesnt feel go like
		if err == nil {
			res = []byte(strings.Join(resp.AgentIds, ", "))
		}
		resErr = err

	case "getagentidfromref":
		request, ok := testReq.(*grpc_types.GetAgentIDFromRefRequest)
		if !ok {
			tu.FailNowAt(t, "Failed to convert/decode request. This shouldnt happen ...")
		}
		resp, err := client.GetAgentIDFromRef(
			ctx,
			request,
			grpc.Header(header),
			grpc.Trailer(trailer),
		)
		if *tu.Verbose {
			fmt.Printf("Response: " + fmt.Sprintf("%#v", resp) + "\n")
		}
		// Style: this doesnt feel go like
		if err == nil {
			res = []byte(strconv.Itoa(int(resp.AgentId)))
		}
		resErr = err

	case "heartbeat":
		request, ok := testReq.(*grpc_types.HeartBeatRequest)
		if !ok {
			tu.FailNowAt(t, "Failed to convert/decode request. This shouldnt happen ...")
		}
		resp, err := client.HeartBeat(
			ctx,
			request,
			grpc.Header(header),
			grpc.Trailer(trailer),
		)
		// Style: this doesnt feel go like
		if err == nil {
			res = []byte(strconv.Itoa(int(resp.Status)))
		}
		resErr = err

	case "addtask":
		request, ok := testReq.(*grpc_types.AddTaskRequest)
		if !ok {
			tu.FailNowAt(t, "Failed to convert/decode request. This shouldnt happen ...")
		}
		resp, err := client.AddTask(
			ctx,
			request,
			grpc.Header(header),
			grpc.Trailer(trailer),
		)
		// Style: this doesnt feel go like
		if err == nil {
			res = []byte(strconv.Itoa(int(resp.TaskId)))
		}
		resErr = err
	}

	return res, resErr
}

func checkAPICall(t *testing.T, client grpc_types.AgentMgmtClient, session models.Session, source, golden, compare, description, testName string, testReq testRequest, testHasErr amerrors.ErrorType) {
	// read input from file
	src, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	// update mongo db with input data
	tu.InsertFixturesToDB(t, session, testName, src)

	// run service call
	var header, trailer metadata.MD
	res, err := runSrvTest(t, client, &header, &trailer, testName, testReq)

	// is an error is expected? If so, we check it is the correct one
	if err != nil {
		if *tu.Verbose {
			fmt.Printf("Error in response found: " + fmt.Sprintf("%#v", service.UnWrapError(err, trailer)) + "\n")
			fmt.Printf("Expected error found: " + fmt.Sprintf("%#v", amerrors.Is(service.UnWrapError(err, trailer), testHasErr)) + "\n")
		}
		// If expecting an error and it is not the one we thought, fail
		if testHasErr != 0 && !amerrors.Is(service.UnWrapError(err, trailer), testHasErr) {
			t.Error(err)
			tu.FailNowAt(t, "Expected error type:"+amerrors.StrName(testHasErr)+" however got: "+fmt.Sprintf("%#v", service.UnWrapError(err, trailer)))
		}
		// If not expecting an error , fail
		if testHasErr == 0 {
			t.Error(err)
			tu.FailNowAt(t, "Was not expecting an error. Error: "+err.Error())
		}
	}

	// update golden files if necessary
	if *tu.Update {
		if werr := ioutil.WriteFile(golden, res, 0644); werr != nil {
			t.Error(err)
		}
		return
	}

	// get golden
	gld, err := ioutil.ReadFile(golden)
	// TODO: want to remove eol from file length (this is a crap method needs bettering)
	gld = bytes.Trim(gld, "\n\t")
	if err != nil {
		t.Error(err)
		return
	}

	// formatted source and golden must be the same
	if err := tu.Diff(compare, golden, description, res, gld); err != nil {
		t.Error(err)
		return
	}
}

func TestGRPCServerClient(t *testing.T) {

	// Freeze Time
	service.NowFunc = func() time.Time {
		freezeTime := time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)
		logger.Log("level", "debug", "msg", "The time is "+freezeTime.Format("01/02/2006 03:04:05"))
		return freezeTime
	}

	// Initialise mongo connection
	session, _ := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Create Service &  Endpoints (no logger, tracer, metrics etc)
	var (
		service   = service.NewService(nil, nil)
		endpoints = amendpoint.NewEndpoint(service, nil, nil, nil, session, "test")
	)

	// gRPC server
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	srv := transport.GRPCServer(endpoints, nil, nil)
	s := grpc.NewServer()
	grpc_types.RegisterAgentMgmtServer(s, srv)
	go s.Serve(ln)
	defer s.GracefulStop()

	// Connection to grpc server and create a client
	conn, err := grpc.Dial(hostPort, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		t.Fatalf("unable to Dial: %+v", err)
		t.FailNow()
	}

	client := grpc_types.NewAgentMgmtClient(conn)

	//
	// Run through tests
	//
	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			logger.Log("msg", "TestGRPCServerClient: Running service test for "+e.testName)
			defer tu.CleanAllCollectionsTestMongo(session)

			checkAPICall(t, client, session, source, golden, e.compare, e.description, e.testName, e.testReq, e.testHasErr)
		})
	}
}

// TestGRPCQueryError tests the server against query errors
func TestGRPCQueryError(t *testing.T) {
	// Initialise mongo connection
	session, _ := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Create Service &  Endpoints (no logger, tracer, metrics etc)
	// https://husobee.github.io/golang/testing/unit-test/2015/06/08/golang-unit-testing.html
	var (
		svc = tu.MockService{
			MockGetAvailableAgents: func() ([]string, error) {
				var agentIDs []string
				return agentIDs, &mgo.QueryError{Code: 1}
			},
			MockGetAgentIDFromRef: func() (int32, error) {
				return 0, &mgo.QueryError{Code: 1}
			},
			MockHeartBeat: func() (grpc_types.HeartBeatResponse_HeartBeatStatus, error) {
				return grpc_types.HeartBeatResponse_HEARTBEAT_FAILED, &mgo.QueryError{Code: 1}
			},
			MockAddTask: func() (int32, error) {
				return 0, &mgo.QueryError{Code: 1}
			},
		}
		endpoints = amendpoint.NewEndpoint(svc, nil, nil, nil, session, "test")
	)

	// gRPC server
	ln, err := net.Listen("tcp", hostPort)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	srv := transport.GRPCServer(endpoints, nil, nil)
	s := grpc.NewServer()
	grpc_types.RegisterAgentMgmtServer(s, srv)
	go s.Serve(ln)
	defer s.GracefulStop()

	// Connection to grpc server and create a client
	conn, err := grpc.Dial(hostPort, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		t.Fatalf("unable to Dial: %+v", err)
		t.FailNow()
	}

	client := grpc_types.NewAgentMgmtClient(conn)

	// Run through tests
	for _, e := range dataMockErrors {
		t.Run(e.description, func(t *testing.T) {
			logger.Log("msg", "Running service test (mocking QueryError) for "+e.testName)

			// run service call
			var header, trailer metadata.MD
			_, err := runSrvTest(t, client, &header, &trailer, e.testName, e.testReq)

			// is an error is expected? If so, we check it is the correct one{
			s, _ := status.FromError(err)
			if s.Code() != codes.Unknown {
				t.Error(err)
				t.FailNow()
			}

		})
	}
}
