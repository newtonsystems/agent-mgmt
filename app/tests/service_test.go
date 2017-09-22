package tests

// Test service layer
// gracefully ripped from https://github.com/hashicorp/hcl/blob/master/hcl/printer/printer_test.go

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/metrics"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
)

var update = flag.Bool("update", false, "update golden files")

type entry struct {
	source, golden string
}

// Use go test -update to create/update the respective golden files.
var data = []entry{
	{"getavailableagents.input", "getavailableagents.golden"},
	{"getavailableagents_oldheartbeat.input", "getavailableagents_oldheartbeat.golden"},
	{"getavailableagents_futureheartbeat.input", "getavailableagents_futureheartbeat.golden"},
	{"getavailableagents_minuteagoexactly.input", "getavailableagents_minuteagoexactly.golden"},
}

func TestFiles(t *testing.T) {
	// Initialise mongo connection
	moSession := CreateTestMongoConnection()
	defer moSession.Close()

	var ints, chars metrics.Counter
	// Create new service
	s := service.NewService(logger, ints, chars)

	for _, e := range data {
		source := filepath.Join(dataDir, e.source)
		golden := filepath.Join(dataDir, e.golden)
		t.Run(e.source, func(t *testing.T) {
			check(t, s, moSession, source, golden)
		})
	}
}

func check(t *testing.T, s service.Service, session models.Session, source, golden string) {
	src, err := ioutil.ReadFile(source)
	if err != nil {
		t.Error(err)
		return
	}

	res, err := s.GetAvailableAgents(context.Background(), session, "fakedb")
	if err != nil {
		t.Error(err)
		return
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
	if err := diff(source, golden, res, gld); err != nil {
		t.Error(err)
		return
	}
}

// diff compares a and b.
func diff(aname, bname string, a, b []byte) error {
	var buf bytes.Buffer // holding long error message

	// compare lengths
	if len(a) != len(b) {
		fmt.Fprintf(&buf, "\nlength changed: len(%s) = %d, len(%s) = %d", aname, len(a), bname, len(b))
	}

	// compare contents
	line := 1
	offs := 1
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
		return errors.New(buf.String())
	}
	return nil
}

// // format parses src, prints the corresponding AST, verifies the resulting
// // src is syntactically correct, and returns the resulting src or an error
// // if any.
// func format(src []byte) ([]byte, error) {
// 	formatted, err := Format(src)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// make sure formatted output is syntactically correct
// 	if _, err := parser.Parse(formatted); err != nil {
// 		return nil, fmt.Errorf("parse: %s\n%s", err, formatted)
// 	}

// 	return formatted, nil
// }

// lineAt returns the line in text starting at offset offs.
func lineAt(text []byte, offs int) []byte {
	i := offs
	for i < len(text) && text[i] != '\n' {
		i++
	}
	return text[offs:i]
}
