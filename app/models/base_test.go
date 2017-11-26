package models_test

import (
	"fmt"
	"testing"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	tu "github.com/newtonsystems/agent-mgmt/app/testutil"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"gopkg.in/mgo.v2/bson"
)

var logger = utils.GetLogger()

// Future
// Property tests http://www.quii.co.uk/Property-based%20testing%20in%20Go
// agent agent_id is unique
// phonesession can not be inserted if agent_id doesnt exist
// phonesession sess_id is unique

func TestGetNextSequence(t *testing.T) {

	testCases := []struct {
		description string
		name        string
		expectedSeq int32
		expectedErr tu.TestAMErrorType
	}{
		{"next_seq_1", "genericid", 2, nil},
		{"next_seq_2", "genericid", 3, nil},
		{"next_seq_3", "genericid", 4, nil},
		{"wrong_seq", "wrongid", 0, amerrors.ErrCounterNotFound},
	}

	// Initialise mongo connection
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Create a sequence
	err := db.C("counters").Insert(bson.M{
		"_id": "genericid",
		"seq": 1,
	})

	if err != nil {
		logger.Log("level", "error", "msg", "Failed to set up counter for testcases")
		panic(err)
	}

	//
	// Run table driven tests
	//
	for _, tc := range testCases {
		if *tu.Verbose {
			fmt.Printf("Running 'TestGetNextSequence' (" + tc.description + ")")
		}

		t.Run(tc.description, func(t *testing.T) {
			seqID, err := db.GetNextSequence(tc.name)
			tu.Equals(t, tc.expectedSeq, seqID)
			tu.IsAmError(t, tc.expectedErr, err)
		})
	}

}

func TestPrepareDB(t *testing.T) {

	testCases := []struct {
		description string
		name        string
		expectedSeq int32
	}{
		{"prepare_1", "taskid", 1},
		{"wrong_seq", "wrongid", 0},
	}

	// Initialise mongo connection
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)
	// TODO: add option to make preparing db optionally
	tu.DropTestMongoDB(t, session)

	//
	// Run table driven tests
	//
	for _, tc := range testCases {
		if *tu.Verbose {
			fmt.Printf("Running 'TestGetNextSequence' (" + tc.description + ")")
		}

		t.Run(tc.description, func(t *testing.T) {
			models.PrepareDB(session, tu.MongoDBName, logger)
			defer tu.DropTestMongoDB(t, session)

			var doc models.Count
			_ = db.C("counters").FindId(tc.name).One(&doc)

			tu.Equals(t, tc.expectedSeq, doc.Seq)
		})
	}
}
