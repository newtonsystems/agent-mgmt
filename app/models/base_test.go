package models_test

import (
	"flag"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/tests"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	mgo "gopkg.in/mgo.v2"
)

type collectionModel interface{}

type entry struct {
	collection  string             // Collection Name
	model       [2]collectionModel // The model to be inserted into mongo
	isDupError  bool               // Is the inserted models cause a duplicate error
	description string             // A useful description of what the test intends to accomplish
}

var debug = flag.Bool("debug", false, "turn on mongo debug")

var logger = utils.GetLogger()

const (
	mongoDBName = "test"
)

var data = []entry{
	{
		"agents",
		[2]collectionModel{
			&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
			&models.Agent{AgentID: 10, LastHeartBeat: time.Now().Add(-time.Minute)},
		},
		true,
		"Check the uniqueness of agents collection (agent_id unique)",
	},
	{
		"phonesessions",
		[2]collectionModel{
			&models.PhoneSession{SessID: 10, AgentID: 1, RefID: "ref01a"},
			&models.PhoneSession{SessID: 10, AgentID: 2, RefID: "ref01b"},
		},
		true,
		"Check the uniqueness of phonesessions collection (sess_id unique)",
	},
	{
		"phonesessions",
		[2]collectionModel{
			&models.PhoneSession{SessID: 10, AgentID: 1, RefID: "ref01a"},
			&models.PhoneSession{SessID: 11, AgentID: 1, RefID: "ref01b"},
		},
		false,
		"Check no duplication error of phonesessions collection if same agent_id is inserted.",
	},
	{
		"phonesessions",
		[2]collectionModel{
			&models.PhoneSession{SessID: 10, AgentID: 1, RefID: "ref01a"},
			&models.PhoneSession{SessID: 11, AgentID: 2, RefID: "ref01a"},
		},
		false,
		"Check no duplication error of phonesessions collection if same ref_id is inserted.",
	},
}

// cleanUp removes everyfrom the database including all collections
func cleanUp(session models.Session) {
	session.DB(mongoDBName).DropDatabase()
}

// cleanUpCollection removes all items from a collection
func cleanUpCollection(session models.Session, collection string) {
	var i interface{}
	session.DB(mongoDBName).C(collection).RemoveAll(i)
}

func isUnique(t *testing.T, expectError bool, collection string, err error) {
	if expectError && !mgo.IsDup(err) {
		t.Error(err)
		tests.FailNowAt(t, "Failed to find duplication error. Collection "+collection+" is not unique.")
	}
}

func hasCountChanged(t *testing.T, expectError bool, collection string, prevCount, count int) {
	if expectError && prevCount != count {
		tests.FailNowAt(t, "Expect count for collection "+collection+" to have NOT changed. (count: "+strconv.Itoa(count)+" prevCount: "+strconv.Itoa(prevCount)+")")
	}
	if !expectError && prevCount == count {
		tests.FailNowAt(t, "Expect count for collection "+collection+" to have changed. (count: "+strconv.Itoa(count)+" prevCount: "+strconv.Itoa(prevCount)+")")
	}
}

func TestCheckCollectionsUniqueness(t *testing.T) {
	// initialise mongo connection
	moSession := tests.CreateTestMongoConnection(*debug, true)
	defer moSession.Refresh()
	defer moSession.Close()

	// Run through tests
	for _, e := range data {
		t.Run(e.collection, func(t *testing.T) {
			logger.Log("msg", "Running service uniqueness test for collection "+e.collection)

			// insert models (twice) into mongo
			switch e.collection {
			case "agents":
				var prevCount = 0
				for i := 0; i <= 1; i++ {
					model, ok := e.model[i].(*models.Agent)
					if !ok {
						tests.FailNowAt(
							t,
							"Failed to convert/decode to "+fmt.Sprintf("%#v", e.model[i])+". This shouldnt happen ...",
						)
					}
					err := moSession.DB(mongoDBName).C(e.collection).Insert(model)

					// On second attempt check for duplicate error and check collection
					// size has not increased
					count, _ := moSession.DB(mongoDBName).C(e.collection).Count()
					if i == 1 {
						isUnique(t, e.isDupError, e.collection, err)
						hasCountChanged(t, e.isDupError, e.collection, prevCount, count)
					}

					prevCount = count
				}
			case "phonesessions":
				var prevCount = 0
				for i := 0; i <= 1; i++ {
					model, ok := e.model[i].(*models.PhoneSession)
					if !ok {
						tests.FailNowAt(
							t,
							"Failed to convert/decode to "+fmt.Sprintf("%#v", e.model[i])+". This shouldnt happen ...",
						)
					}
					err := moSession.DB(mongoDBName).C(e.collection).Insert(model)

					// On second attempt check for duplicate error and check collection
					// size has not increased
					count, _ := moSession.DB(mongoDBName).C(e.collection).Count()
					if i == 1 {
						isUnique(t, e.isDupError, e.collection, err)
						hasCountChanged(t, e.isDupError, e.collection, prevCount, count)
					}

					prevCount = count
				}

			}
			// Clean up collection after every test
			cleanUpCollection(moSession, e.collection)
		})
	}

	// Clean up database
	cleanUp(moSession)

}

// Future
// Property tests http://www.quii.co.uk/Property-based%20testing%20in%20Go
// agent agent_id is unique
// phonesession can not be inserted if agent_id doesnt exist
// phonesession sess_id is unique
