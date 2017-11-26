package models_test

// Basic property + table driven tests for agent.go

import (
	"fmt"
	"testing"

	"github.com/newtonsystems/agent-mgmt/app/models"
	tu "github.com/newtonsystems/agent-mgmt/app/testutil"
)

func TestIndexSessIDUnique(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	originalCount, _ := db.C("phonesessions").Count()
	tu.Equals(t, 0, originalCount)

	// Insert agent into db
	db.C("phonesessions").Insert(&models.PhoneSession{SessID: 10, AgentID: 1, RefID: "ref8933"})
	count, _ := db.C("phonesessions").Count()
	tu.Equals(t, 1, count)

	// Insert agent with the same agent ID
	db.C("phonesessions").Insert(&models.PhoneSession{SessID: 10, AgentID: 2, RefID: "ref8934"})
	count, _ = db.C("phonesessions").Count()
	tu.Equals(t, 1, count)

}

func TestGetAgentIDFromRef(t *testing.T) {
	testCases := []struct {
		description     string
		inserts         []tu.TestModelInsert
		refID           string
		expectedAgentID int32
		expectedErr     tu.TestAMErrorType
	}{
		{
			"get_agent_id_from_ref",
			[]tu.TestModelInsert{
				&models.PhoneSession{SessID: 2, AgentID: 10, RefID: "ref8934"},
			},
			"ref8934",
			10,
			nil,
		},
		{
			"get_agent_id_from_ref_wrong_ref",
			[]tu.TestModelInsert{
				&models.PhoneSession{SessID: 2, AgentID: 10, RefID: "ref8934"},
			},
			"ref8935",
			0,
			nil,
		},
	}

	// Initialise mongo connection
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Run table driven tests
	for _, tc := range testCases {
		if *tu.Verbose {
			fmt.Printf("Running 'TestGetAgents' (" + tc.description + ")")
		}

		t.Run(tc.description, func(t *testing.T) {
			defer tu.CleanAllCollectionsTestMongo(session)
			tu.InsertCollectionToDB(t, db, "phonesessions", tc.inserts)

			agentID, _ := db.GetAgentIDFromRef(tc.refID)
			tu.Equals(t, tc.expectedAgentID, agentID)
		})

	}

}
