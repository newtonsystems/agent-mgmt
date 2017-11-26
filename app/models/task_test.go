package models_test

// Basic property tests for task.go

import (
	"fmt"
	"reflect"
	"testing"
	"testing/quick"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	tu "github.com/newtonsystems/agent-mgmt/app/testutil"
	"gopkg.in/mgo.v2/bson"
)

func TestAddTaskCustIDStaysSame(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	//
	// Cust ID doesnt change when adding a task (invariant property)
	//
	assertion := func(custID int32, agentIDs []int32) bool {
		if *tu.Verbose {
			fmt.Printf("Running 'TestAddTaskCustIDStaysSame' (when adding tasks cust id should not change) assert check: (custID=%d)\n", custID)
		}
		if custID <= 0 {
			return true
		}

		_, err := db.AddTask(custID, agentIDs)
		tu.Ok(t, err)

		var task models.Task
		err = db.C("tasks").Find(bson.M{"custid": custID}).Select(bson.M{"custid": 1}).One(&task)
		tu.Ok(t, err)

		return custID == task.CustID
	}

	err := quick.Check(assertion, nil)
	tu.Ok(t, err)

}

func TestAddTaskAgentIDsStaysSame(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	//
	// Agent ID doesnt change when adding a task (invariant property)
	//
	assertion := func(custID int32, agentIDs []int32) bool {
		if *tu.Verbose {
			fmt.Printf("Running 'TestAddTaskAgentIDsStaysSame' (when adding tasks agent id should not change) assert check: (agentIDs=%d)\n", agentIDs)
		}
		if custID <= 0 {
			return true
		}

		_, err := db.AddTask(custID, agentIDs)
		tu.Ok(t, err)

		var task models.Task
		err = db.C("tasks").Find(bson.M{"agentids": agentIDs}).Select(bson.M{"agentids": 1}).One(&task)
		tu.Ok(t, err)

		return reflect.DeepEqual(agentIDs, task.AgentIDs)
	}

	err := quick.Check(assertion, nil)
	tu.Ok(t, err)

}

func TestAddTaskCustIDInvalid(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	//
	// Check invalid cust ID are caught with ErrCustIDInvalidError
	//
	// Invalid cust id are considered custID <= 0
	//
	assertion := func(custID int32, agentIDs []int32) bool {
		if *tu.Verbose {
			fmt.Printf("Running 'TestAddTaskCustIDInvalid' assert check: (custID=%d)\n", custID)
		}
		if custID > 0 {
			return true
		}

		taskID, err := db.AddTask(custID, agentIDs)

		return amerrors.Is(err, amerrors.ErrCustIDInvalid) && taskID == 0
	}

	err := quick.Check(assertion, nil)
	tu.Ok(t, err)

}

func TestAddTaskTaskIDIncrements(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	assertion := func(custID int32, agentIDs []int32) bool {
		if *tu.Verbose {
			fmt.Printf("Running 'TestAddTaskTaskIDIncrements' (when adding tasks the taskID should increment the seq) assert check: (custID=%d)\n", custID)
		}
		if custID <= 0 {
			return true
		}

		// What was the original taskid
		var counter models.Count
		errCount := db.C("counters").Find(bson.M{"_id": "taskid"}).Select(bson.M{"seq": 1}).One(&counter)
		tu.Ok(t, errCount)

		// AddTask
		taskID, err := db.AddTask(custID, agentIDs)
		tu.Ok(t, err)

		// Check DB
		var task models.Task
		err = db.C("tasks").Find(bson.M{"_id": taskID}).Select(bson.M{"_id": 1}).One(&task)
		tu.Ok(t, err)

		return counter.Seq+1 == taskID && taskID == taskID
	}

	err := quick.Check(assertion, nil)
	tu.Ok(t, err)

}

func TestAddTask(t *testing.T) {

	testCases := []struct {
		description    string
		custID         int32
		agentIDs       []int32
		expectedTaskID int32
		expectedErr    tu.TestAMErrorType
	}{
		{"cust_id_should_produce_error", 0, []int32{1, 2, 3}, 0, amerrors.ErrCustIDInvalid},
		{"cust_id_1", 1, []int32{100, 2, 3}, 2, nil},
		{"cust_id_100", 100, []int32{1, 2, 3}, 3, nil},
		{"cust_id_1000", 1000, []int32{1, 2, 3}, 4, nil},
	}

	// Initialise mongo connection
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Run table driven tests
	for _, tc := range testCases {
		if *tu.Verbose {
			fmt.Printf("Running 'TestAddTask' (" + tc.description + ")")
		}

		t.Run(tc.description, func(t *testing.T) {
			// NOTE: We dont clean up after every test (so seq increases)
			taskID, err := db.AddTask(tc.custID, tc.agentIDs)
			tu.Equals(t, tc.expectedTaskID, taskID)
			tu.IsAmError(t, tc.expectedErr, err)
		})
	}

}
