package tests

// utils.go
// A collection of useful tiny testing functions
// disgracefully ripped from https://github.com/benbjohnson/testing

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/newtonsystems/agent-mgmt/app/models"
)

// InsertFixturesToDB Unmarshal JSON From File
func InsertFixturesToDB(t *testing.T, session models.Session, testName, source string, src []byte, verbose *bool) {
	var errMessage = "No JSON data found when unmarshalled data from " + source

	switch testName {
	case "getavailableagents":
		fallthrough
	case "heartbeat":
		var agents []models.Agent
		json.Unmarshal(src, &agents)

		// Check we have found some input
		if len(agents) == 0 {
			FailNowAt(t, errMessage)
		}

		// Insert agents into mongo
		for _, agent := range agents {
			if *verbose {
				fmt.Printf("Inserting " + fmt.Sprintf("%#v", agent) + " into collection 'agents'\n")
			}
			err := session.DB("test").C("agents").Insert(agent)
			if err != nil {
				t.Error(err)
				FailNowAt(t, "Could not insert "+fmt.Sprintf("%#v", agent)+" into mongo")
			}
		}

	case "getagentidfromref":
		var phoneSessions []models.PhoneSession
		json.Unmarshal(src, &phoneSessions)

		// Check we have found some input
		if len(phoneSessions) == 0 {
			FailNowAt(t, errMessage)
		}

		// Insert phonesessions into mongo
		for _, phoneSess := range phoneSessions {
			if *verbose {
				fmt.Printf("Inserting " + fmt.Sprintf("%#v", phoneSess) + " into collection 'phonesessions'\n")
			}
			err := session.DB("test").C("phonesessions").Insert(phoneSess)
			if err != nil {
				t.Error(err)
				FailNowAt(t, "Could not insert "+fmt.Sprintf("%#v", phoneSess)+" into mongo")
			}
		}

	}
}

// FailNowAt is a helper function to display more information on a Fail Now
func FailNowAt(t *testing.T, msg string) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, msg)
	t.FailNow()
}

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// Ok fails the test if an err is not nil.
func Ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// Diff compares a and b.
func Diff(aname, bname, desc string, a, b []byte) error {
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