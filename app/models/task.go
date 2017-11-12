package models

// agent.go
// Agent Model / Mongo Calls

import (
	"time"
	"strconv"
		amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
)

// Task - models for phone task note bson uses int32 a lot
type Task struct {
	TaskID   int32     `bson:"_id" json:"_id"`
	CustID   int32     `bson:"custid" json:"custid"`
	AgentIDs []int32   `bson:"agentids" json:"agentids"`
	AddedAt  time.Time `bson:"addedat" json:"addedat"`
}

// Mongo Calls

// AddTask add a task to mongo and returns the newly created Task's id if successful
func (db *MongoDatabase) AddTask(custID int32, agentIDs []int32) (int32, error) {
	if custID <= 0 {
		return 0, amerrors.ErrCustIDInvalidError("Invalid Cust ID: " + strconv.Itoa(int(custID)))
	}

	taskID := GetNextSequence(*db, "taskid")
	//logger.Log("level", "debug", "msg", "Created new sequence "+strconv.Itoa(int(taskID)))

	err := db.C("tasks").Insert(&Task{
		TaskID: taskID,
		CustID: custID,
		AgentIDs: agentIDs,
		AddedAt: NowFunc(),
	})

	if err != nil {
		return 0, err
	}
	return taskID, nil
}
