package main_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"bytes"
	"time"
	"testing"

	"github.com/go-kit/kit/endpoint"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	amendpoint "github.com/newtonsystems/agent-mgmt/app/endpoint"
	//amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/agent-mgmt/app/tests"
	"github.com/newtonsystems/agent-mgmt/app/transport"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var update = flag.Bool("update", false, "update golden files")
var verbose = flag.Bool("verbose", false, "turn on more verbose output")
var debug = flag.Bool("debug", false, "turn on mongo debug")

var logger = utils.GetLogger()

const (
	dataDir = "./testdata"
	mongoDBName = "test"
)

type testRequest interface {
}

type entry struct {
	testName    string
	testArgs    testRequest
	testHasErr  error
	source      string
	compare     string
	golden      string
	description string
}

const (
	hostPort        string = ":8004"
	mongoDBDatabase string = "test"
)

var data = []entry{
	{
		"getavailableagents",
		&grpc_types.GetAvailableAgentsRequest{},
		nil,
		"getavailableagents.input",
		"response agent IDs",
		"getavailableagents.golden",
		"A basic test of service's GetAvailableAgents()",
	},
}

// cleanUp removes everyfrom the database including all collections
func cleanUp(session models.Session) {
	session.DB(mongoDBName).DropDatabase()
}


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

func TestGRPCClient(t *testing.T) {

	// Freeze Time
	service.NowFunc = func() time.Time {
		freezeTime := time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)
		logger.Log("level", "debug", "msg", "The time is "+freezeTime.Format("01/02/2006 03:04:05"))
		return freezeTime
	}

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
		defer s.GracefulStop()

		s.Serve(ln)
	}()


	// Connection to grpc server and create a client
	conn, err := grpc.Dial(hostPort, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		t.Fatalf("unable to Dial: %+v", err)
		t.FailNow()
	}

	client := grpc_types.NewAgentMgmtClient(conn)

	// Run through tests
	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			logger.Log("msg", "Running service test for "+e.testName)
			checkAPICall(t, client, moSession, source, golden, e.compare, e.description, e.testName, e.testArgs, e.testHasErr)
		})
		cleanUp(moSession)
	}

}

// runSrvTest runs a specifc test based off testName
// we convert to bytes for possible writing
func runSrvTest(t *testing.T, client grpc_types.AgentMgmtClient, header, trailer metadata.MD, testName string, testArgs testRequest) ([]byte, error) {
	var res []byte
	var resErr error
	ctx := context.Background()

	switch testName {
	case "getavailableagents":
		request, ok := testArgs.(*grpc_types.GetAvailableAgentsRequest)
		if !ok {
			tests.FailNowAt(t, "Failed to convert/decode request. This shouldnt happen ...")
		}
		resp, err := client.GetAvailableAgents(
			ctx,
			request,
			grpc.Header(&header),
			grpc.Trailer(&trailer),
		)
		if err != nil {
			//res = []byte()
			resErr = err
			logger.Log("msg", fmt.Sprintf("\n%#v", resp))

		} else {
			res = []byte(strings.Join(resp.AgentIds, ", "))
			resErr = err
		}

	case "getagentidfromref":
		resp, err := client.GetAgentIDFromRef(
			ctx,
			&grpc_types.GetAgentIDFromRefRequest{RefId: "hsajdhjas"},
			grpc.Header(&header),
			grpc.Trailer(&trailer),
		)
		res = []byte(strconv.Itoa(int(resp.AgentId)))
		resErr = err

	case "heartbeat":
		resp, err := client.HeartBeat(
			ctx,
			&grpc_types.HeartBeatRequest{},
			grpc.Header(&header),
			grpc.Trailer(&trailer),
		)
		res = []byte(strconv.Itoa(int(resp.Status)))
		resErr = err

	}

	return res, resErr
}

// Unmarshal JSON From File
func insertFixtureToDatabase(t *testing.T, session models.Session, testName, source string, src []byte) {
	var errMessage = "No JSON data found when unmarshalled data from " + source

	switch testName {
	case "getavailableagents":
	fallthrough
	case "heartbeat":
		var agents []models.Agent
		json.Unmarshal(src, &agents)

		// Check we have found some input
		if len(agents) == 0 {
			tests.FailNowAt(t, errMessage)
		}

		// Insert agents into mongo
		for _, agent := range agents {
			if *verbose {
				fmt.Printf("Inserting "+fmt.Sprintf("%#v", agent)+" into collection 'agents'\n")
			}
			err := session.DB("test").C("agents").Insert(agent)
			if err != nil {
				t.Error(err)
				tests.FailNowAt(t, "Could not insert " + fmt.Sprintf("%#v", agent) + " into mongo")
			}
		}

	case "getagentidfromref":
		var phoneSessions []models.PhoneSession
		json.Unmarshal(src, &phoneSessions)

		// Check we have found some input
		if len(phoneSessions) == 0 {
			tests.FailNowAt(t, errMessage)
		}

		// Insert phonesessions into mongo
		for _, phoneSess := range phoneSessions {
			if *verbose {
				fmt.Printf("Inserting "+fmt.Sprintf("%#v", phoneSess)+" into collection 'phonesessions'\n")
			}
			err := session.DB("test").C("phonesessions").Insert(phoneSess)
			if err != nil {
				t.Error(err)
				tests.FailNowAt(t, "Could not insert " + fmt.Sprintf("%#v", phoneSess) + " into mongo")
			}
		}

	}
}

func checkAPICall(t *testing.T, client grpc_types.AgentMgmtClient, session models.Session, source, golden, compare, description, testName string, testArgs testRequest, testHasErr error) {
	// read input from file
	src, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	// update mongo db with input data
	insertFixtureToDatabase(t, session, testName, source, src)

	// run service call
	var header, trailer metadata.MD
	res, err := runSrvTest(t, client, header, trailer, testName, testArgs)

	// is an error is expected? If so, we check it is the correct one
	if err != nil {
		if testHasErr != nil && service.UnWrapError(err, trailer) != testHasErr {
			t.Error(err)
			t.FailNow()
		}
	}

	// update golden files if necessary
	if *update {
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
	if err := tests.Diff(compare, golden, description, res, gld); err != nil {
		t.Error(err)
		return
	}

}
