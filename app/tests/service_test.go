package tests

// Test service layer
// gracefully ripped from https://github.com/hashicorp/hcl/blob/master/hcl/printer/printer_test.go

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"

	"github.com/go-kit/kit/metrics"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	//"github.com/newtonsystems/agent-mgmt/app/utils"
)

//var logger = utils.GetLogger()

var update = flag.Bool("update", false, "update golden files")

var debug = flag.Bool("debug", false, "update golden files")

type entry struct {
	source, golden, description string
}

//const (
//	dataDir = "./testdata"
//)

// Use go test -update to create/update the respective golden files.
var data = []entry{
	{
		"getavailableagents.input",
		"getavailableagents.golden",
		"A basic test of service's GetAvailableAgents()",
	},
	{
		"getavailableagents_oldheartbeat.input",
		"getavailableagents_oldheartbeat.golden",
		"A test to ensure heartbeats older than one minute are not included as available agents by service's GetAvailableAgents()",
	},
	{
		"getavailableagents_futureheartbeat.input",
		"getavailableagents_futureheartbeat.golden",
		"A test to ensure heartbeats newer than one minute are included as available agents by service's GetAvailableAgents()  (We accept future timestamps)",
	},
	{
		"getavailableagents_minuteagoexactly.input",
		"getavailableagents_minuteagoexactly.golden",
		"A test to ensure a heartbeat exactly a minute ago is included as an available agent by service's GetAvailableAgents()",
	},
	{
		"getavailableagents_limit_results_10.input",
		"getavailableagents_limit_results_10.golden",
		"A test to check there is a limit to the available agent ids returned by service's GetAvailableAgents()",
	},
}

func clearAgentsCollection(sess models.Session) {
	var i interface{}
	sess.DB("test").C("agents").RemoveAll(i)
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
	var ints, chars metrics.Counter
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
	}

	// Create new service
	s := service.NewService(logger, ints, chars)

	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			check(t, s, moSession, source, golden, e.description)
		})
		clearAgentsCollection(moSession)
	}
}

func check(t *testing.T, s service.Service, session models.Session, source, golden, description string) {
	src, err := ioutil.ReadFile(source)

	if err != nil {
		t.Error(err)
		return
	}

	var agents []models.Agent
	json.Unmarshal(src, &agents)

	if len(agents) == 0 {
		var errMessage = "No input found from " + source
		logger.Log("info", "crit", "msg", errMessage)
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, errMessage)
		t.FailNow()
	}

	// Insert into mongo
	for _, agent := range agents {
		err1 := session.DB("test").C("agents").Insert(agent)
		if err1 != nil {
			logger.Log("msg", "Could not insert input into", "err", err)
			t.Error(err)
		}
	}

	res_s, err := s.GetAvailableAgents(context.Background(), session, "test")
	if err != nil {
		t.Error(err)
		return
	}

	// Convert to bytes for possible writing
	resString := strings.Join(res_s, ", ")
	res := []byte(resString)

	// // update golden files if necessary
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

// diff compares a and b.
func diff(aname, bname, desc string, a, b []byte) error {
	var buf bytes.Buffer // holding long error message

	// compare lengths
	if len(a) != len(b) {
		fmt.Fprintf(&buf, "\nlength changed: len(%s) = %d, len(%s) = %d", aname, len(a), bname, len(b))
	}

	// compare contents
	line := 1
	offs := 0
	for i := 0; i < len(a) && i < len(b); i++ {
		ch := a[i]
		if ch != b[i] {
			fmt.Fprintf(&buf, "\n%s:%d:%d: %q", aname, line, i-offs+1, lineAt(a, offs))
			fmt.Fprintf(&buf, "\n%s:%d:%d: %q", bname, line, i-offs+1, lineAt(b, offs))
			fmt.Fprintf(&buf, "\n\n")
			break
		}
		if ch == '\n' {
			line++
			offs = i + 1
		}
	}

	if buf.Len() > 0 {
		fmt.Fprintf(&buf, "\n%s\n", desc)
		return errors.New(buf.String())
	}
	return nil
}

// lineAt returns the line in text starting at offset offs.
func lineAt(text []byte, offs int) []byte {
	i := offs
	for i < len(text) && text[i] != '\n' {
		i++
	}
	return text[offs:i]
}
