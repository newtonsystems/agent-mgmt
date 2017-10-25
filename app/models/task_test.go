package models_test

// Basic property tests for task.go

import (
	"fmt"
	"testing"
	"testing/quick"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/tests"
	"gopkg.in/mgo.v2/bson"
)

// func TestAddingExample(t *testing.T) {
// 	if add(3, 2) != 5 {
// 		t.Error("3 plus 2 is 5")
// 	}
// }
//
// func TestAddingZeroMakesNoDifference(t *testing.T) {
// 	assertion := func(x int) bool {
// 		return add(x, 0) == x
// 	}
//
// 	if err := quick.Check(assertion, nil); err != nil {
// 		t.Error(err)
// 	}
// }
//
// func TestAssociativity(t *testing.T) {
// 	assertion := func(x, y, z int) bool {
// 		return add(add(x, y), z) == add(add(z, y), x)
// 	}
//
// 	if err := quick.Check(assertion, nil); err != nil {
// 		t.Error(err)
// 	}
// }

func TestCustIDStaysSame(t *testing.T) {
	moSession := tests.CreateTestMongoConnection(*debug, true)
	defer moSession.Refresh()
	defer moSession.Close()

	var i interface{}
	moSession.DB(mongoDBName).C("tasks").RemoveAll(i)

	db := moSession.DB(mongoDBName)

	assertion := func(custID int64, agentIDs []int32) bool {
		if custID <= 0 {
			return true
		}
		var task models.Task
		fmt.Printf("Testing with custid: %d\n", custID)
		a := []int32{2, 3, 4}
		_, err := db.AddTask(custID, a)

		if err != nil {
			fmt.Printf(err.Error())
		}
		err = db.C("tasks").Find(bson.M{"custID": custID}).Select(bson.M{"custid": 1}).One(&task)
		//fmt.Printf("Testing with custid: %d\n   %d", task.CustID, taskID)
		//if err != nil {
		//	fmt.Printf(err.Error() + "\n")
		//}
		return custID == task.CustID
	}

	if err := quick.Check(assertion, nil); err != nil {
		t.Error(err)
		t.FailNow()
	}
}
