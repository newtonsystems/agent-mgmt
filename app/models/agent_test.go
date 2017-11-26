package models_test

// Basic property + table driven tests for agent.go

import (
	"fmt"
	"testing"
	"time"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	tu "github.com/newtonsystems/agent-mgmt/app/testutil"
	"gopkg.in/mgo.v2/bson"
)

func TestIndexAgentIDUnique(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	originalCount, _ := db.C("agents").Count()
	tu.Equals(t, 0, originalCount)

	// Insert agent into db
	db.C("agents").Insert(&models.Agent{AgentID: 10, LastHeartBeat: time.Now()})
	count, _ := db.C("agents").Count()
	tu.Equals(t, 1, count)

	// Insert agent with the same agent ID
	db.C("agents").Insert(&models.Agent{AgentID: 10, LastHeartBeat: time.Now()})
	count, _ = db.C("agents").Count()
	tu.Equals(t, 1, count)

}

func TestAgentExists(t *testing.T) {
	testCases := []struct {
		description    string
		inserts        []tu.TestModelInsert
		agentID        int32
		expectedExists bool
		expectedErr    tu.TestAMErrorType
	}{
		{
			"insert_one_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
			},
			10,
			true,
			nil,
		},
		{
			"insert_one_doesnt_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
			},
			11,
			false,
			amerrors.ErrAgentNotFound,
		},
		{
			"insert_two_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Now().Add(-time.Minute)},
			},
			10,
			true,
			nil,
		},
		{
			"insert_three_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Now().Add(-time.Minute)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Now().Add(-time.Minute)},
			},
			10,
			true,
			nil,
		},
		{
			"insert_three_doesnt_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Now().Add(-time.Minute)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Now().Add(-time.Minute)},
			},
			13,
			false,
			amerrors.ErrAgentNotFound,
		},
		{
			"insert_three_exists_middle_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Now().Add(-time.Minute)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Now().Add(-time.Minute)},
			},
			11,
			true,
			nil,
		},
		{
			"insert_three_exists_end_exists",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Now()},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Now().Add(-time.Minute)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Now().Add(-time.Minute)},
			},
			12,
			true,
			nil,
		},
	}

	// Initialise mongo connection
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Run table driven tests
	for _, tc := range testCases {
		if *tu.Verbose {
			fmt.Printf(tu.ColourBlue + "Running 'TestAgentExists' (" + tc.description + ")" + tu.ColourReset + "\n")
		}

		t.Run(tc.description, func(t *testing.T) {
			defer tu.CleanAllCollectionsTestMongo(session)
			tu.InsertCollectionToDB(t, db, "agents", tc.inserts)

			exists, err := db.AgentExists(tc.agentID)

			tu.Equals(t, tc.expectedExists, exists)
			tu.IsAmError(t, tc.expectedErr, err)
		})

	}

}

func TestHeartBeat(t *testing.T) {
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// Insert agent into db
	originalTime := time.Now()
	db.C("agents").Insert(&models.Agent{AgentID: 10, LastHeartBeat: originalTime})
	count, _ := db.C("agents").Count()
	tu.Equals(t, 1, count)

	// Update heartbeat with an agent that doesnt exist
	err := db.HeartBeat(11)
	tu.IsAmError(t, amerrors.ErrAgentNotFound, err)

	// Update heartbeat on correct agent then check the timestamp has changed
	err = db.HeartBeat(10)
	tu.Ok(t, err)

	var agent models.Agent
	_ = db.C("counters").Find(bson.M{"agentid": 10}).One(&agent)
	tu.NotEquals(t, originalTime, agent.LastHeartBeat)
}

func TestGetAgents(t *testing.T) {
	testCases := []struct {
		description    string
		inserts        []tu.TestModelInsert
		timestamp      time.Time
		limit          int32
		expectedAgents []models.Agent
		expectedErr    tu.TestAMErrorType
	}{
		{
			"get_agents",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			100,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_only_one",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			100,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			100,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_limit_1",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			1,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_limit_2",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			2,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_limit_3",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			3,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_limit_4",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			4,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_limit_5",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 50, 29, 0, time.UTC),
			5,
			[]models.Agent{
				models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_some_too_old",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 51, 29, 0, time.UTC),
			100,
			[]models.Agent{
				models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			nil,
		},
		{
			"get_agents_multiple_all_too_old",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 54, 29, 0, time.UTC),
			100,
			[]models.Agent{},
			nil,
		},
		{
			"get_agents_multiple_all_too_old_limit_1",
			[]tu.TestModelInsert{
				&models.Agent{AgentID: 10, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)},
				&models.Agent{AgentID: 11, LastHeartBeat: time.Date(2017, time.September, 21, 17, 50, 33, 0, time.UTC)},
				&models.Agent{AgentID: 12, LastHeartBeat: time.Date(2017, time.September, 21, 17, 51, 31, 0, time.UTC)},
				&models.Agent{AgentID: 13, LastHeartBeat: time.Date(2017, time.September, 21, 17, 52, 31, 0, time.UTC)},
			},
			time.Date(2017, time.September, 21, 17, 54, 29, 0, time.UTC),
			1,
			[]models.Agent{},
			nil,
		},
	}

	// Initialise mongo connection
	session, db := tu.NewTestMongoConnection(*tu.Debug, *tu.OutsideConn)
	defer tu.CleanUpTestMongoConnection(t, session)

	// freeze time (Wow! cool!)
	service.NowFunc = func() time.Time {
		freezeTime := time.Date(2017, time.September, 21, 17, 50, 31, 0, time.UTC)
		if *tu.Verbose {
			fmt.Printf("The time is " + freezeTime.Format("01/02/2006 17:04:05"))
		}
		return freezeTime
	}

	// Run table driven tests
	for _, tc := range testCases {
		if *tu.Verbose {
			fmt.Printf("Running 'TestGetAgents' (" + tc.description + ")")
		}

		t.Run(tc.description, func(t *testing.T) {
			defer tu.CleanAllCollectionsTestMongo(session)
			tu.InsertCollectionToDB(t, db, "agents", tc.inserts)

			agents, err := db.GetAgents(tc.timestamp, tc.limit)
			tu.IsAmError(t, tc.expectedErr, err)

			// Check lengths are the same
			tu.Equals(t, len(tc.expectedAgents), len(agents))

			// Compare expected agents vs actual agents
			for i, agent := range tc.expectedAgents {
				tu.Equals(t, tc.expectedAgents[i].AgentID, agent.AgentID)
				tu.TimeEquals(t, tc.expectedAgents[i].LastHeartBeat, agent.LastHeartBeat)
			}

		})

	}

}
