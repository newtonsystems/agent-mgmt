package tests

// Test service layer
// gracefully ripped from https://github.com/hashicorp/hcl/blob/master/hcl/printer/printer_test.go

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/go-kit/kit/metrics"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	//"github.com/newtonsystems/agent-mgmt/app/utils"
)

//var logger = utils.GetLogger()

var update = flag.Bool("update", false, "update golden files")

var debug = flag.Bool("debug", false, "update golden files")

type entry struct {
	srvTestName, srvTestArgs, source, golden, description string
	srvTestErr                                            error
}

//const (
//	dataDir = "./testdata"
//)

// Use go test -update to create/update the respective golden files.
var data = []entry{
	{
		"getavailableagents",
		"",
		"getavailableagents.input",
		"getavailableagents.golden",
		"A basic test of service's GetAvailableAgents()",
		nil,
	},
	{
		"getavailableagents",
		"",
		"getavailableagents_oldheartbeat.input",
		"getavailableagents_oldheartbeat.golden",
		"A test to ensure heartbeats older than one minute are not included as available agents by service's GetAvailableAgents()",
		nil,
	},
	{
		"getavailableagents",
		"",
		"getavailableagents_futureheartbeat.input",
		"getavailableagents_futureheartbeat.golden",
		"A test to ensure heartbeats newer than one minute are included as available agents by service's GetAvailableAgents()  (We accept future timestamps)",
		nil,
	},
	{
		"getavailableagents",
		"",
		"getavailableagents_minuteagoexactly.input",
		"getavailableagents_minuteagoexactly.golden",
		"A test to ensure a heartbeat exactly a minute ago is included as an available agent by service's GetAvailableAgents()",
		nil,
	},
	{
		"getavailableagents",
		"",
		"getavailableagents_limit_results_10.input",
		"getavailableagents_limit_results_10.golden",
		"A test to check there is a limit to the available agent ids returned by service's GetAvailableAgents()",
		nil,
	},
	{
		"getagentidfromref",
		"ref001a",
		"getagentidfromref.input",
		"getagentidfromref.golden",
		"A basic test of service's GetAgentIDFromRef()",
		nil,
	},
	{
		"getagentidfromref",
		"",
		"getagentidfromref_empty.input",
		"getagentidfromref_empty.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is empty returned by service's GetAgentIDFromRef()",
		amerrors.ErrAgentIDNotFoundError(""),
	},
	{
		"getagentidfromref",
		"refwrong",
		"getagentidfromref_wrongref.input",
		"getagentidfromref_wrongref.golden",
		"A test to check that we get an ErrAgentIDNotFound error when refID is incorrect returned by service's GetAgentIDFromRef()",
		amerrors.ErrAgentIDNotFoundError(""),
	},
	{
		"heartbeat",
		"20",
		"heartbeat.input",
		"heartbeat.golden",
		"A basic test of service's HeartBeat()",
		nil,
	},
}

func clearAgentsCollection(sess models.Session) {
	var i interface{}
	sess.DB("test").C("agents").RemoveAll(i)
	sess.DB("test").C("phonesessions").RemoveAll(i)
}

func insertAgentsIntoMongoFromInput(t *testing.T, session models.Session, srcFile []byte, source string) {
	var agents []models.Agent

	// Unmarshal JSON From File
	json.Unmarshal(srcFile, &agents)

	// Check we have found some input
	if len(agents) == 0 {
		var errMessage = "No input data found from " + source
		_, file, line, _ := runtime.Caller(1)
		logger.Log("info", "crit", "msg", errMessage)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, errMessage)
		t.FailNow()
	}

	// Insert into mongo
	for _, agent := range agents {
		err1 := session.DB("test").C("agents").Insert(agent)
		if err1 != nil {
			logger.Log("msg", "Could not insert input into mongo", "err", err1)
			t.Error(err1)
		}
	}

}

func tGetAvailableAgents(t *testing.T, source string, s service.Service, session models.Session, src []byte) ([]byte, error) {
	var res []byte

	insertAgentsIntoMongoFromInput(t, session, src, source)

	agentIDs, err := s.GetAvailableAgents(context.Background(), session, "test")

	// Convert to bytes for possible writing
	resString := strings.Join(agentIDs, ", ")
	res = []byte(resString)

	return res, err
}

func tGetAgentIDFromRef(t *testing.T, source string, s service.Service, srvTestArgs string, session models.Session, src []byte) ([]byte, error) {
	var phoneSessions []models.PhoneSession
	json.Unmarshal(src, &phoneSessions)

	if len(phoneSessions) == 0 {
		var errMessage = "No input data found from " + source
		logger.Log("info", "crit", "msg", errMessage)
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, errMessage)
		t.FailNow()
	}

	// Insert into mongo
	for _, phoneSess := range phoneSessions {
		fmt.Printf(fmt.Sprintf("\n%#v", phoneSess))
		err := session.DB("test").C("phonesessions").Insert(phoneSess)
		if err != nil {
			logger.Log("msg", "Could not insert input into mongo", "err", err)
			t.Error(err)
			t.FailNow()
		}
	}

	agentID, err := s.GetAgentIDFromRef(session, "test", srvTestArgs)
	res := []byte(strconv.Itoa(int(agentID)))

	return res, err
}

func tHeartBeat(t *testing.T, source string, s service.Service, srvTestArgs string, session models.Session, src []byte) ([]byte, error) {
	var res []byte

	insertAgentsIntoMongoFromInput(t, session, src, source)

	agentID, err := strconv.Atoi(srvTestArgs)

	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	agent := models.Agent{AgentID: int32(agentID)}
	status, err := s.HeartBeat(session, "test", agent)
	res = []byte(strconv.Itoa(int(status)))

	return res, err
}

func TestFiles(t *testing.T) {

	// Initialise mongo connection
	moSession := CreateTestMongoConnection(*debug)
	defer moSession.Refresh()
	defer moSession.Close()

	service.NowFunc = func() time.Time {
		freezeTime := time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)
		logger.Log("level", "debug", "msg", "The time is "+freezeTime.Format("01/02/2006 03:04:05"))
		return freezeTime
	}

	// TODO: Create a Mock Version or fix this
	// (Not a priority at the moment)

	var ints, chars, refs, beats metrics.Counter
	{
		// Business-level metrics.
		ints = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "addsvc",
			Name:      "integers_summed",
			Help:      "Total count of integers summed via the Sum method.",
		}, []string{})
		chars = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "addsvc",
			Name:      "characters_concatenated",
			Help:      "Total count of characters concatenated via the Concat method.",
		}, []string{})
		refs = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "agentmgmt",
			Name:      "references_used",
			Help:      "Total count of references used to get agent ID via the GetAgentIDFromRef method.",
		}, []string{})
		beats = kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
			Namespace: "example",
			Subsystem: "agentmgmt",
			Name:      "total_heartbeat_counts",
			Help:      "Total count of heartbeats service call from the HeartBeat method.",
		}, []string{})
	}

	// Create new service
	s := service.NewService(logger, ints, chars, refs, beats)

	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			check(t, s, moSession, e.srvTestName, e.srvTestArgs, source, golden, e.srvTestErr, e.description)
		})
		clearAgentsCollection(moSession)
	}
}

func check(t *testing.T, srv service.Service, session models.Session, srvTestCase, srvTestArgs, source, golden string, srvTestErr error, description string) {
	src, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	var res []byte
	var srvError error
	if srvTestCase == "getavailableagents" {
		res, srvError = tGetAvailableAgents(t, source, srv, session, src)
	} else if srvTestCase == "getagentidfromref" {
		res, srvError = tGetAgentIDFromRef(t, source, srv, srvTestArgs, session, src)
	} else if srvTestCase == "heartbeat" {
		res, srvError = tHeartBeat(t, source, srv, srvTestArgs, session, src)
	} else {
		t.Error("test service call name '" + srvTestCase + "' is unknown")
		return
	}

	if srvError != nil {
		if srvTestErr != nil && srvError != srvTestErr {
			t.Error(srvError)
			return
		}
	}

	// update golden files if necessary
	if *update {
		if err := ioutil.WriteFile(golden, res, 0644); err != nil {
			t.Error(err)
		}
		return
	}

	// get golden
	gld, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Error(err)
		return
	}

	// formatted source and golden must be the same
	if err := diff(source, golden, description, res, gld); err != nil {
		t.Error(err)
		return
	}
}
