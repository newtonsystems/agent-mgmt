package tests

// testing.go
// These are export APIs for the sole purpose of providing
// mocks, test harnesses, helpers, etc.

import (
	"context"
	"encoding/json"
	stdlog "log"
	"os"
	//"fmt"
	"io/ioutil"
	//"os"
	"path/filepath"
	//"strconv"
	"time"

	"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/service"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"github.com/newtonsystems/grpc_types/go/grpc_types"
	"gopkg.in/mgo.v2"
)

var logger = utils.GetLogger()

const (
	dataDir = "./testdata"
)

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

// Service Layer Mocking -------------------------------------------------------

// MockService acts as a mock of service.Service
type MockService struct {
	MockGetAvailableAgents func() ([]string, error)
	MockGetAgentIDFromRef  func() (int32, error)
	MockHeartBeat          func() (grpc_types.HeartBeatResponse_HeartBeatStatus, error)
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

// -----------------------------------------------------------------------------

// MockSession satisfies Session and act as a mock of *mgo.session.
type MockSession struct{}

// NewMockSession mock NewSession.
func NewMockSession() models.Session {
	return MockSession{}
}

func (fs MockSession) SetSafe(*mgo.Safe) {}

func (fs MockSession) SetSyncTimeout(time.Duration) {}

func (fs MockSession) SetSocketTimeout(time.Duration) {}

// Copy mocks mgo.Session.Copy().
func (fs MockSession) Copy() models.Session {
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
func (db MockDatabase) AddTask(custID int64, agentIDs []int32) (int32, error) {
	return 0, nil
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

// Create a real mongo connection for tests
// set to "test" database
func CreateTestMongoConnection(debug bool, prepare bool) models.Session {
	// Initialise mongodb connection and logger
	// Create a session which maintains a pool of socket connections to our MongoDB.
	var mongoExternalHost = envString("MONGO_EXTERNAL_SERVICE_HOST", "192.168.99.100") + ":" + envString("MONGO_EXTERNAL_SERVICE_PORT", "31017")

	mongoHosts := []string{
		mongoExternalHost,
	}
	mongoDBDialInfo := &mgo.DialInfo{
		Addrs:    mongoHosts,
		Timeout:  20 * time.Second,
		Database: "test",
	}

	if debug {
		mgo.SetDebug(true)
		var debugMongoLogger *stdlog.Logger
		debugMongoLogger = stdlog.New(os.Stderr, "", stdlog.LstdFlags)
		mgo.SetLogger(debugMongoLogger)
	}

	mongoSession, err := mgo.DialWithInfo(mongoDBDialInfo)

	// Can't connect? - bail!
	if err != nil {
		panic(err)
	}

	// Optional. Switch the session to a monotonic behavior.
	//mongoSession.SetMode(mgo.Monotonic, true)

	// Wrap mgo session in user defined interface/structs
	// This means we can mock db calls more easily
	session := models.MongoSession{mongoSession}
	session.SetSafe(&mgo.Safe{})
	session.SetSyncTimeout(7 * time.Second)
	session.SetSocketTimeout(10 * time.Second)

	// Prepare database
	if prepare {
	}
	models.PrepareDB(session, "test", logger)
	//	}
	return session
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
