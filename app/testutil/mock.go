package testutil

// testing.go
// These are export APIs for the sole purpose of providing
// mocks, test harnesses, helpers, etc.

import (
	"context"
	"encoding/json"
	//"fmt"
	"io/ioutil"
	//"os"
	"path/filepath"
	//"strconv"
	"time"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
	"gopkg.in/mgo.v2"
)

// Service Layer Mocking -------------------------------------------------------

// MockService acts as a mock of service.Service
type MockService struct {
	MockGetAvailableAgents func() ([]string, error)
	MockGetAgentIDFromRef  func() (int32, error)
	MockHeartBeat          func() (grpc_types.HeartBeatResponse_HeartBeatStatus, error)
	MockAddTask            func() (int32, error)
}

func NewMockService() service.Service {
	return MockService{}
}

func (fs MockService) Sum(ctx context.Context, a, b int) (int, error) {
	return 0, nil
}

func (fs MockService) Concat(ctx context.Context, a, b string) (string, error) {
	return "", nil
}

func (fs MockService) GetAvailableAgents(ctx context.Context, session models.Session, db string, limit int32) ([]string, error) {
	var strNil []string
	if fs.MockGetAvailableAgents != nil {
		return fs.MockGetAvailableAgents()
	}
	return strNil, nil
}

func (fs MockService) GetAgentIDFromRef(session models.Session, db string, refID string) (int32, error) {
	if fs.MockGetAgentIDFromRef != nil {
		return fs.MockGetAgentIDFromRef()
	}
	return 0, nil
}

func (fs MockService) HeartBeat(session models.Session, db string, agentID int32) (grpc_types.HeartBeatResponse_HeartBeatStatus, error) {
	if fs.MockHeartBeat != nil {
		return fs.MockHeartBeat()
	}
	return grpc_types.HeartBeatResponse_HEARTBEAT_SUCCESSFUL, nil
}

func (fs MockService) AddTask(session models.Session, db string, custID int32, agentIDs []int32) (int32, error) {
	if fs.MockAddTask != nil {
		return fs.MockAddTask()
	}
	return 1, nil
}

// -----------------------------------------------------------------------------

// MockSession satisfies Session and act as a mock of *mgo.session.
type MockSession struct{}

// NewMockSession mock NewSession.
func NewMockSession() models.Session {
	return MockSession{}
}

func (fs MockSession) SetSafe(*mgo.Safe) {}

func (fs MockSession) SetMode(consistency mgo.Mode, refresh bool) {}

func (fs MockSession) SetSyncTimeout(time.Duration) {}

func (fs MockSession) SetSocketTimeout(time.Duration) {}

// Copy mocks mgo.Session.Copy().
func (fs MockSession) Copy() models.Session {
	return MockSession{}
}

// Clone mocks mgo.Session.Clone().
func (fs MockSession) Clone() models.Session {
	return MockSession{}
}

// Close mocks mgo.Session.Close().
func (fs MockSession) Close() {}

func (fs MockSession) Refresh() {}

// DB mocks mgo.Session.DB().
func (fs MockSession) DB(name string) models.DataLayer {
	mockDatabase := MockDatabase{}
	return mockDatabase
}

func (fs MockSession) Ping() error {
	return nil
}

// MockDatabase satisfies DataLayer and act as a mock.
type MockDatabase struct{}

// MockCollection satisfies Collection and act as a mock.
type MockCollection struct{}

// Find mock.
func (fc MockCollection) Find(query interface{}) *mgo.Query {
	return nil
}

func (fc MockCollection) FindId(query interface{}) *mgo.Query {
	return nil
}

// Count mock.
func (fc MockCollection) Count() (n int, err error) {
	return 10, nil
}

// Insert mock.
func (fc MockCollection) Insert(docs ...interface{}) error {
	return nil
}

// Remove mock.
func (fc MockCollection) Remove(selector interface{}) error {
	return nil
}

// Update mock.
func (fc MockCollection) Update(selector interface{}, update interface{}) error {
	return nil
}

// UpdateId mock.
func (fc MockCollection) UpdateId(id interface{}, update interface{}) error {
	return nil
}

// EnsureIndex mock.
func (fc MockCollection) EnsureIndex(index mgo.Index) error {
	return nil
}

// Upsert mock.
func (fc MockCollection) Upsert(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error) {
	return nil, nil
}

// RemoveAll mock.
func (fc MockCollection) RemoveAll(selector interface{}) (info *mgo.ChangeInfo, err error) {
	return nil, nil
}

// C mocks mgo.Database(name).Collection(name).
func (db MockDatabase) C(name string) models.Collection {
	return MockCollection{}
}

// Mock service calls

//AgentExists mocks models.AgentExists().
func (db MockDatabase) AgentExists(agentID int32) (bool, error) {
	return true, nil
}

//GetAgents mocks models.GetAgents().
func (db MockDatabase) GetAgents(timestamp time.Time, limit int32) ([]models.Agent, error) {
	var agents []models.Agent
	source := filepath.Join(dataDir, "get_agents.json")

	src, err := ioutil.ReadFile(source)
	if err != nil {
		panic(err)
	}

	json.Unmarshal(src, &agents)

	return agents, nil
}

// AddTask mocks models.AddTask().
func (db MockDatabase) AddTask(custID int32, agentIDs []int32) (int32, error) {
	return 0, nil
}

func (db MockDatabase) GetNextSequence(name string) (int32, error) {
	return 1, nil
}

//GetAgentIDFromRef mocks models.GetAgents().
func (db MockDatabase) GetAgentIDFromRef(refID string) (int32, error) {
	return 0, nil
}

//HeartBeat mocks models.GetAgents().
func (db MockDatabase) HeartBeat(agentID int32) error {
	return nil
}

//DropDatabase mocks db.DropDatabase().
func (db MockDatabase) DropDatabase() error {
	return nil
}

// // GetScores mocks models.GetScores().
// func (db FakeDatabase) GetScores() ([]Score, error) {
// 	var Scores []Score
// 	scoreContent, _ := ioutil.ReadFile(
// 		"/go/src/github.com/thylong/regexrace/config/default_scores.json")
// 	json.Unmarshal(scoreContent, &Scores)

// 	return Scores, nil
// }

// // FindTopScores mocks models.FindTopScores().
// func (db FakeDatabase) FindTopScores() ([]Score, error) {
// 	var Scores []Score
// 	scoreContent, _ := ioutil.ReadFile(
// 		"/go/src/github.com/thylong/regexrace/config/default_scores.json")
// 	json.Unmarshal(scoreContent, &Scores)

// 	return Scores, nil
//
