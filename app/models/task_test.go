package models_test

// Basic property tests for task.go

import (
	"fmt"
	"testing"
	"testing/quick"
	"reflect"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/tests"
	"gopkg.in/mgo.v2/bson"
	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
)

//const (
//	mongoDBName = "test"
//)

func TestAddTaskCustIDStaysSame(t *testing.T) {
	moSession := tests.CreateTestMongoConnection(false, true)
	defer moSession.Refresh()
	defer moSession.Close()
	db := moSession.DB(mongoDBName)

	assertion := func(custID int32, agentIDs []int32) bool {
		fmt.Printf("Running 'TestAddTaskCustIDStaysSame' assert check: (custID=%d)\n",custID)
		if custID <= 0 {
			return true
		}

		_, err := db.AddTask(custID, agentIDs)
		tests.Ok(t, err)

		var task models.Task
		err = db.C("tasks").Find(bson.M{"custid": custID}).Select(bson.M{"custid": 1}).One(&task)
		tests.Ok(t, err)

		return custID == task.CustID
	}

	err := quick.Check(assertion, nil)
	tests.Ok(t, err)

	// Cleanup
	moSession.DB(mongoDBName).C("tasks").RemoveAll(nil)
}

func TestAddTaskAgentIDsStaysSame(t *testing.T) {
	moSession := tests.CreateTestMongoConnection(false, true)
	defer moSession.Refresh()
	defer moSession.Close()
	db := moSession.DB(mongoDBName)

	assertion := func(custID int32, agentIDs []int32) bool {
		fmt.Printf("Running 'TestAddTaskAgentIDsStaysSame' assert check: (agentIDs=%d)\n", agentIDs)
		if custID <= 0 {
			return true
		}

		_, err := db.AddTask(custID, agentIDs)
		tests.Ok(t, err)

		var task models.Task
		err = db.C("tasks").Find(bson.M{"agentids": agentIDs}).Select(bson.M{"agentids": 1}).One(&task)
		tests.Ok(t, err)

		return reflect.DeepEqual(agentIDs, task.AgentIDs)
	}

	err := quick.Check(assertion, nil)
	tests.Ok(t, err)

	// Cleanup
	moSession.DB(mongoDBName).C("tasks").RemoveAll(nil)
}

func TestAddTaskCustIDInvalid(t *testing.T) {
	moSession := tests.CreateTestMongoConnection(false, true)
	defer moSession.Refresh()
	defer moSession.Close()
	db := moSession.DB(mongoDBName)

	assertion := func(custID int32, agentIDs []int32) bool {
		fmt.Printf("Running 'TestAddTaskCustIDInvalid' assert check: (custID=%d)\n",custID)
		if custID > 0 {
			return true
		}

		taskID, err := db.AddTask(custID, agentIDs)

		return amerrors.Is(err, amerrors.ErrCustIDInvalid) && taskID == 0
	}

	err := quick.Check(assertion, nil)
	tests.Ok(t, err)

	// Cleanup
	moSession.DB(mongoDBName).C("tasks").RemoveAll(nil)
}

func TestAddTaskTaskIDIncrements(t *testing.T) {
	moSession := tests.CreateTestMongoConnection(false, true)
	defer moSession.Refresh()
	defer moSession.Close()
	db := moSession.DB(mongoDBName)

	assertion := func(custID int32, agentIDs []int32) bool {
		fmt.Printf("Running 'TestAddTaskTaskIDIncrements' assert check: (custID=%d)\n",custID)
		if custID <= 0 {
			return true
		}

		// What was the original taskid
		var counter models.Count
		errCount := db.C("counters").Find(bson.M{"_id": "taskid"}).Select(bson.M{"seq": 1}).One(&counter)
		tests.Ok(t, errCount)

		// AddTask
		taskID, err := db.AddTask(custID, agentIDs)
		tests.Ok(t, err)

		// Check DB
		var task models.Task
		err = db.C("tasks").Find(bson.M{"_id": taskID}).Select(bson.M{"_id": 1}).One(&task)
		tests.Ok(t, err)

		return counter.Seq + 1 == taskID && taskID == taskID
	}

	err := quick.Check(assertion, nil)
	tests.Ok(t, err)

	// Cleanup
	moSession.DB(mongoDBName).C("tasks").RemoveAll(nil)
}
