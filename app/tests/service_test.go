package tests

// Test service layer
// gracefully ripped from https://github.com/hashicorp/hcl/blob/master/hcl/printer/printer_test.go

import (
	"bytes"
	"context"
	"flag"
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
	//"gopkg.in/mgo.v2/bson"
)

var update = flag.Bool("update", false, "update golden files")
var verbose = flag.Bool("verbose", false, "turn on more verbose output")
var debug = flag.Bool("debug", false, "update golden files")

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
	//dataDir     = "./testdata"
	mongoDBName = "test"
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
				FailNowAt(t, errConvert.Error())
			}
			limit = int32(limitInt)
		}

		agentIDs, err := s.GetAvailableAgents(ctx, session, mongoDBName, limit)

		// Style: this doesnt feel go like
		if err == nil {
			res = []byte(strings.Join(agentIDs, ", "))
		}
		resErr = err

	case "getagentidfromref":
		agentID, err := s.GetAgentIDFromRef(session, mongoDBName, testArgs[0])

		if *verbose {
			fmt.Printf("Response: " + fmt.Sprintf("%#v", agentID) + "\n")
		}

		res = []byte(strconv.Itoa(int(agentID)))
		resErr = err

	case "heartbeat":
		agentID, errConvert := strconv.Atoi(testArgs[0])

		if errConvert != nil {
			FailNowAt(t, errConvert.Error())
		}

		status, err := s.HeartBeat(session, mongoDBName, int32(agentID))

		res = []byte(strconv.Itoa(int(status)))
		resErr = err

	case "addtask":
		custID, errConvert := strconv.Atoi(testArgs[0])

		if errConvert != nil {
			FailNowAt(t, errConvert.Error())
		}

		var agentIDs []int32
		ids := strings.Split(testArgs[1], ",")

		for _, item := range ids {
			agentID, errConvert := strconv.Atoi(item)

			if errConvert != nil {
				FailNowAt(t, errConvert.Error())
			}
			agentIDs = append(agentIDs, int32(agentID))
		}

		taskID, err := s.AddTask(session, mongoDBName, int32(custID), agentIDs)

		res = []byte(strconv.Itoa(int(taskID)))
		resErr = err

	}

	return res, resErr
}

// cleanUp removes everyfrom the database including all collections
func cleanUp(session models.Session) {
	session.DB(mongoDBName).DropDatabase()
}

// cleanUpCollection removes all items from a collection
func cleanUpCollection(session models.Session, testName string) {
	var i interface{}
	var collection string
	switch testName {
	case "getavailableagents":
		fallthrough
	case "heartbeat":
		collection = "agents"
	case "getagentidfromref":
		collection = "phonesessions"
	case "addtask":
		collection = "tasks"
	}

	//session.DB(mongoDBName).C("counters").UpdateId("taskid", bson.M{"$set": bson.M{"seq": 1}})
	session.DB(mongoDBName).C(collection).RemoveAll(i)
}

func TestFiles(t *testing.T) {

	// Initialise mongo connection
	moSession := CreateTestMongoConnection(*debug, true)
	defer moSession.Refresh()
	defer moSession.Close()

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
			check(t, moSession, s, source, golden, e.compare, e.description, e.testName, e.testArgs, e.testHasErr)
		})
		cleanUpCollection(moSession, e.testName)
	}
	cleanUp(moSession)
}

func check(t *testing.T, session models.Session, srv service.Service, source, golden, compare, description, testName string, testArgs []string, testHasErr amerrors.ErrorType) {
	src, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	// update mongo db with input data
	InsertFixturesToDB(t, session, testName, source, src, verbose)

	// run service call
	res, err := runSrvTest(t, session, srv, testName, testArgs)

	// is an error is expected? If so, we check it is the correct one
	if err != nil {
		if *verbose {
			fmt.Printf("Error in response found: " + err.Error())
			fmt.Printf("Expected error found: " + fmt.Sprintf("%#v", amerrors.Is(err, testHasErr)))
		}
		if testHasErr != 0 && !amerrors.Is(err, testHasErr) {
			FailNowAt(t, "Expected error type:"+amerrors.StrName(testHasErr)+" however got: "+fmt.Sprintf("%#v", err))
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
	if err := Diff(compare, golden, description, res, gld); err != nil {
		t.Error(err)
		return
	}
}
