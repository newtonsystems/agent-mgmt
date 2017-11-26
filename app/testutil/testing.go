package testutil

// testing.go
// These are export APIs for the sole purpose of providing
// mocks, test harnesses, helpers, etc.

import (
	"flag"
	stdlog "log"
	"os"
	"testing"
	//"fmt"

	//"os"

	//"strconv"
	"time"

	tmodels "github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var logger = utils.GetLogger()

const (
	dataDir     = "./testdata"
	MongoDBName = "test" // The database table to use for tests
)

// Update retuhfdjh
var Update = flag.Bool("update", false, "update golden files")

// Verbose dfdsfgd
var Verbose = flag.Bool("verbose", false, "turn on more verbose output")

// Debug turns on Mongo debug
var Debug = flag.Bool("debug", false, "update golden files")

// OutsideConn connect to a remote cluster from (outside of k8s cluster)
var OutsideConn = flag.Bool("conn.local", true, "If connecting from outside of cluster")

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

type isMasterResult struct {
	IsMaster  bool
	Secondary bool
	Primary   string
	Hosts     []string
	Passives  []string
	Tags      bson.D
	Msg       string
}

// CleanUpTestMongoConnection clean ups mongo connecition - drop test databse and closes session
func CleanUpTestMongoConnection(t *testing.T, session tmodels.Session) {
	logger.Log("level", "info", "msg", "Cleaning up test mongo connection ... ")
	defer session.Close()
	err := session.DB(MongoDBName).DropDatabase()

	if err != nil {
		FailNowAt(t, "Failed to remove test database")
	}

}

// DropTestMongoDB clean ups mongo connecition - drop test databse and closes session
func DropTestMongoDB(t *testing.T, session tmodels.Session) {
	logger.Log("level", "debug", "msg", "Drop test mongo db ... ")

	err := session.DB(MongoDBName).DropDatabase()

	if err != nil {
		FailNowAt(t, "Failed to drop test database")
	}

}

// CleanAllCollectionsTestMongo cleans all collections from test database
func CleanAllCollectionsTestMongo(session tmodels.Session) {
	logger.Log("level", "debug", "msg", "Cleaning up all collections from test mongo database ... ")
	var i interface{}
	var err error
	_, err = session.DB(MongoDBName).C("agents").RemoveAll(i)

	if err != nil {
		panic(err)
	}

	session.DB(MongoDBName).C("phonesessions").RemoveAll(i)

	if err != nil {
		panic(err)
	}

	session.DB(MongoDBName).C("tasks").RemoveAll(i)

	if err != nil {
		panic(err)
	}

}

// NewTestMongoConnection set to "test" database
func NewTestMongoConnection(debug bool, localConn bool) (tmodels.Session, tmodels.DataLayer) {
	// Initialise mongodb connection and logger
	// Create a session which maintains a pool of socket connections to our MongoDB.
	if debug {
		mgo.SetDebug(true)
		var debugMongoLogger *stdlog.Logger
		debugMongoLogger = stdlog.New(os.Stderr, "", stdlog.LstdFlags)
		mgo.SetLogger(debugMongoLogger)
	}

	var mongoSession *mgo.Session

	if localConn {
		var mongo0Host = envString("MONGO_0_SERVICE_HOST", "192.168.99.100") + ":" + envString("MONGO_0_SERVICE_PORT", "31070")
		var mongo1Host = envString("MONGO_1_SERVICE_HOST", "192.168.99.100") + ":" + envString("MONGO_1_SERVICE_PORT", "31071")
		var mongo2Host = envString("MONGO_2_SERVICE_HOST", "192.168.99.100") + ":" + envString("MONGO_2_SERVICE_PORT", "31072")

		mongoHosts := []string{
			mongo0Host,
			mongo1Host,
			mongo2Host,
		}

		//for index := range mongoHosts {
		//println(mongoHosts[index])
		mongoDBDialInfo := &mgo.DialInfo{
			Addrs:    mongoHosts,
			Timeout:  10 * time.Second,
			Database: MongoDBName,
			Direct:   true,
		}

		var err error
		mongoSession, err = mgo.DialWithInfo(mongoDBDialInfo)

		// Can't connect? - bail!
		if err != nil {
			panic(err)
		}

		// Optional. Switch the session to a monotonic behavior.
		mongoSession.SetMode(mgo.Strong, false)
		//
		// 	var result isMasterResult
		// 	err = mongoSession.Run("ismaster", &result)
		//
		// 	if err != nil {
		// 		mongoSession.Close()
		// 		println("Failed to run command ismaster (error: " + err.Error() + " )")
		// 		panic(err)
		// 	}
		// 	println(&result)
		//
		// 	if !result.IsMaster {
		// 		//mongoSession.Close()
		// 		println("Failed to connect to master ... ")
		// 		continue
		// 	}
		//
		// 	println("Connected to master ...")
		// 	break
		// }

	} else {
		mongoHosts := []string{
			"mongo-0.mongo:27017",
			"mongo-1.mongo:27017",
			"mongo-2.mongo:27017",
		}
		mongoDBDialInfo := &mgo.DialInfo{
			Addrs:    mongoHosts,
			Timeout:  10 * time.Second,
			Database: "test",
		}

		var err error
		mongoSession, err = mgo.DialWithInfo(mongoDBDialInfo)

		// Can't connect? - bail!
		if err != nil {
			panic(err)
		}

		// Optional. Switch the session to a monotonic behavior.
		mongoSession.SetMode(mgo.Monotonic, true)

	}

	// Optional. Add stats
	mgo.SetStats(true)

	// Wrap mgo session in user defined interface/structs
	// This means we can mock db calls more easily
	session := tmodels.MongoSession{mongoSession}

	session.SetSafe(&mgo.Safe{WMode: "majority", J: true})
	session.EnsureSafe(&mgo.Safe{WMode: "majority", J: true})
	session.SetSyncTimeout(10 * time.Second)
	session.SetSocketTimeout(10 * time.Second)

	// Drop database - to have a clean start
	err := session.DB(MongoDBName).DropDatabase()

	if err != nil {
		logger.Log("level", "error", "msg", "Failed to drop test database before trying to prepare")
		panic(err)
	}

	// Prepare database
	tmodels.PrepareDB(session, MongoDBName, logger)

	db := session.DB(MongoDBName)

	return session, db
}

// // GetScores mocks tmodels.GetScores().
// func (db FakeDatabase) GetScores() ([]Score, error) {
// 	var Scores []Score
// 	scoreContent, _ := ioutil.ReadFile(
// 		"/go/src/github.com/thylong/regexrace/config/default_scores.json")
// 	json.Unmarshal(scoreContent, &Scores)

// 	return Scores, nil
// }

// // FindTopScores mocks tmodels.FindTopScores().
// func (db FakeDatabase) FindTopScores() ([]Score, error) {
// 	var Scores []Score
// 	scoreContent, _ := ioutil.ReadFile(
// 		"/go/src/github.com/thylong/regexrace/config/default_scores.json")
// 	json.Unmarshal(scoreContent, &Scores)

// 	return Scores, nil
//
