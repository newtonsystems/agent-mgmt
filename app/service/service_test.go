package service_test

// Test service layer
// gracefully ripped from https://github.com/hashicorp/hcl/blob/master/hcl/printer/printer_test.go

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	tu "github.com/newtonsystems/agent-mgmt/app/testutil"
	"github.com/newtonsystems/agent-mgmt/app/utils"
)

var logger = utils.GetLogger()

type entry struct {
	testName    string             // An identifier test name e.g. getavailableagents
	testArgs    []string           // A list of args for the service call
	testHasErr  amerrors.ErrorType // The error expected by service call. Nil if no error is expected by the rpc call
	source      string             // The source file that contains data to be inserted into mongo
	compare     string             // A description of what we compare against the golden
	golden      string             // The golden file
	description string             // A useful description of what the test intends to accomplish
}

const (
	dataDir = "../testutil/testdata"
)

// Use go test -update to create/update the respective golden files.
var data = []entry{
	{
		"getavailableagents",
		[]string{""},
		0,
		"getavailableagents.input",
		"response agent IDs",
		"getavailableagents.golden",
		"A basic test of service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		[]string{""},
		0,
		"getavailableagents_oldheartbeat.input",
		"response agent IDs",
		"getavailableagents_oldheartbeat.golden",
		"A test to ensure heartbeats older than one minute are not included as available agents by service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		[]string{""},
		0,
		"getavailableagents_futureheartbeat.input",
		"response agent IDs",
		"getavailableagents_futureheartbeat.golden",
		"A test to ensure heartbeats newer than one minute are included as available agents by service's GetAvailableAgents()  (We accept future timestamps)",
	},
	{
		"getavailableagents",
		[]string{""},
		0,
		"getavailableagents_minuteagoexactly.input",
		"response agent IDs",
		"getavailableagents_minuteagoexactly.golden",
		"A test to ensure a heartbeat exactly a minute ago is included as an available agent by service's GetAvailableAgents()",
	},
	{
		"getavailableagents",
		[]string{"10"},
		0,
		"getavailableagents_limit_results_10.input",
		"response agent IDs",
		"getavailableagents_limit_results_10.golden",
		"A test to check there is a limit to the available agent ids returned by service's GetAvailableAgents()",
	},
	{
		"getagentidfromref",
		[]string{"ref001a"},
		0,
		"getagentidfromref.input",
		"response agent ID",
		"getagentidfromref.golden",
		"A basic test of service's GetAgentIDFromRef()",
	},
	{
		"getagentidfromref",
		[]string{""},
		amerrors.ErrAgentIDNotFound,
		"getagentidfromref_empty.input",
		"response agent ID",
		"getagentidfromref_empty.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is empty and no phonesession exists returned by service's GetAgentIDFromRef()",
	},
	{
		"getagentidfromref",
		[]string{""},
		0,
		"getagentidfromref.input",
		"response agent ID",
		"getagentidfromref_emptyexists.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is empty and no phonesession exists returned by service's GetAgentIDFromRef()",
	},
	{
		"getagentidfromref",
		[]string{"refwrong"},
		amerrors.ErrAgentIDNotFound,
		"getagentidfromref_wrongref.input",
		"response agent ID",
		"getagentidfromref_wrongref.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is incorrect returned by service's GetAgentIDFromRef()",
	},
	{
		"heartbeat",
		[]string{"20"},
		amerrors.ErrAgentNotFound,
		"heartbeat.input",
		"response status",
		"heartbeat.golden",
		"A basic test of service's HeartBeat()",
	},
	{
		"addtask",
		[]string{"1", "1,2,3"},
		0,
		"addtask.input",
		"response taskID",
		"addtask.golden",
		"A basic test of service's AddTask()",
	},
	{
		"addtask",
		[]string{"0", "1,2,3"},
		amerrors.ErrCustIDInvalid,
		"addtask.input",
		"response taskID",
		"addtask_custid0.golden",
		"A test to check invalid custid of 0 for service's AddTask()",
	},
}

// runSrvTest runs a specifc test based off testName we convert to bytes for possible writing
func runSrvTest(t *testing.T, session models.Session, s service.Service, testName string, testArgs []string) ([]byte, error) {
	var res []byte
	var resErr error
	ctx := context.Background()

	switch testName {
	case "getavailableagents":
		var limit int32
		limit = 0
		if testArgs[0] != "" {
			limitInt, errConvert := strconv.Atoi(testArgs[0])
			if errConvert != nil {
				tu.FailNowAt(t, errConvert.Error())
			}
			limit = int32(limitInt)
		}

		agentIDs, err := s.GetAvailableAgents(ctx, session, tu.MongoDBName, limit)

		// Style: this doesnt feel go like
		if err == nil {
			res = []byte(strings.Join(agentIDs, ", "))
		}
		resErr = err

	case "getagentidfromref":
		agentID, err := s.GetAgentIDFromRef(session, tu.MongoDBName, testArgs[0])

		if *tu.Verbose {
			fmt.Printf("Response: " + fmt.Sprintf("%#v", agentID) + "\n")
		}

		res = []byte(strconv.Itoa(int(agentID)))
		resErr = err

	case "heartbeat":
		agentID, errConvert := strconv.Atoi(testArgs[0])

		if errConvert != nil {
			tu.FailNowAt(t, errConvert.Error())
		}

		status, err := s.HeartBeat(session, tu.MongoDBName, int32(agentID))

		res = []byte(strconv.Itoa(int(status)))
		resErr = err

	case "addtask":
		custID, errConvert := strconv.Atoi(testArgs[0])

		if errConvert != nil {
			tu.FailNowAt(t, errConvert.Error())
		}

		var agentIDs []int32
		ids := strings.Split(testArgs[1], ",")

		for _, item := range ids {
			agentID, errConvert := strconv.Atoi(item)

			if errConvert != nil {
				tu.FailNowAt(t, errConvert.Error())
			}
			agentIDs = append(agentIDs, int32(agentID))
		}

		taskID, err := s.AddTask(session, tu.MongoDBName, int32(custID), agentIDs)

		res = []byte(strconv.Itoa(int(taskID)))
		resErr = err

	}

	return res, resErr
}

func TestGoldenFiles(t *testing.T) {
	// Initialise mongo connection
	session, _ := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Freeze time
	service.NowFunc = func() time.Time {
		freezeTime := time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)
		logger.Log("level", "debug", "msg", "The time is "+freezeTime.Format("01/02/2006 03:04:05"))
		return freezeTime
	}

	// Create new service
	s := service.NewService(logger, nil)

	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			if *tu.Verbose {
				fmt.Printf("Running 'TestGoldenFiles': " + e.testName + " (" + e.description + ")")
			}

			defer tu.CleanAllCollectionsTestMongo(session)
			check(t, session, s, source, golden, e.compare, e.description, e.testName, e.testArgs, e.testHasErr)
		})
	}

}

func check(t *testing.T, session models.Session, srv service.Service, source, golden, compare, description, testName string, testArgs []string, testHasErr amerrors.ErrorType) {

	// Read file
	src, err := ioutil.ReadFile(source)
	if err != nil {
		tu.FailNowAt(t, err.Error())
	}

	// Update mongo db with input data
	tu.InsertFixturesToDB(t, session, testName, src)

	// Run service call
	res, err := runSrvTest(t, session, srv, testName, testArgs)

	// is an error is expected? If so, we check it is the correct one
	if err != nil {
		if *tu.Verbose {
			fmt.Printf("Error in response found: " + err.Error())
			fmt.Printf("Expected error found: " + fmt.Sprintf("%#v", amerrors.Is(err, testHasErr)))
		}
		if testHasErr != 0 && !amerrors.Is(err, testHasErr) {
			tu.FailNowAt(t, "Expected error type:"+amerrors.StrName(testHasErr)+" however got: "+fmt.Sprintf("%#v", err))
		}
	}

	// Update golden files if necessary
	if *tu.Update {
		if werr := ioutil.WriteFile(golden, res, 0644); werr != nil {
			t.Error(err)
		}
		return
	}

	// Get golden file
	gld, err := ioutil.ReadFile(golden)

	// TODO: Want to remove eol from file length (this is a crap method needs bettering)
	gld = bytes.Trim(gld, "\n\t")
	if err != nil {
		t.Error(err)
		return
	}

	// Formatted source and golden must be the same
	if err := tu.Diff(compare, golden, description, res, gld); err != nil {
		tu.FailNowAt(t, err.Error())
	}

}
