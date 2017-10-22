package models

// base.go
// Interface wrapper for mongo mgo
// gracefully ripped from: https://github.com/thylong/regexrace/blob/master/models/base.go

import (
	stdlog "log"
	"os"
	"strings"
	"time"

	//log "github.com/Sirupsen/logrus"
	"github.com/go-kit/kit/log"

	//"github.com/spf13/viper"
	mgo "gopkg.in/mgo.v2"
)

type mongokey string

// MongoKey contains the Mongo session for the Request.
const MongoKey mongokey = "mongo"

// ErrNotFound returned when an object is not found.
var ErrNotFound = mgo.ErrNotFound

// MongoCollection wraps a mgo.Collection to embed methods in models.
type MongoCollection struct {
	*mgo.Collection
}

// Collection is an interface to access to the collection struct.
type Collection interface {
	Find(query interface{}) *mgo.Query
	Count() (n int, err error)
	FindId(id interface{}) *mgo.Query
	Insert(docs ...interface{}) error
	Remove(selector interface{}) error
	Update(selector interface{}, update interface{}) error
	Upsert(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	EnsureIndex(index mgo.Index) error
	RemoveAll(selector interface{}) (info *mgo.ChangeInfo, err error)
}

// MongoDatabase wraps a mgo.Database to embed methods in models.
type MongoDatabase struct {
	*mgo.Database
}

// C shadows *mgo.DB to returns a DataLayer interface instead of *mgo.Database.
func (d MongoDatabase) C(name string) Collection {
	return &MongoCollection{Collection: d.Database.C(name)}
}

// DataLayer is an interface to access to the database struct
// (currently MongoDatabase).
type DataLayer interface {
	C(name string) Collection
	AgentExists(agentID int32) (bool, error)
	GetAgents(timestamp time.Time, limit int32) ([]Agent, error)
	GetAgentIDFromRef(refID string) (int32, error)
	HeartBeat(agentID int32) error
	DropDatabase() error
	//Remove()
	//GetQuestion(qid int) (Question, error)
	//GetScores() ([]Score, error)
	//FindTopScores() ([]Score, error)
}

// Session is an interface to access to the Session struct.
type Session interface {
	DB(name string) DataLayer
	SetSafe(safe *mgo.Safe)
	SetSyncTimeout(d time.Duration)
	SetSocketTimeout(d time.Duration)
	Close()
	Refresh()
	Copy() Session
}

// MongoSession is currently a Mongo session.
type MongoSession struct {
	*mgo.Session
}

// DB shadows *mgo.DB to returns a DataLayer interface instead of *mgo.Database.
func (s MongoSession) DB(name string) DataLayer {
	return &MongoDatabase{Database: s.Session.DB(name)}
}

// Copy mocks mgo.Session.Copy()
func (s MongoSession) Copy() Session {
	return MongoSession{s.Session.Copy()}
}
func (s MongoSession) Close() {
	s.Session.Close()
}

func (s MongoSession) Refresh() {
	s.Session.Refresh()
}

func (s MongoSession) SetSocketTimeout(d time.Duration) {
	s.Session.SetSocketTimeout(d)
}

// NewMongoSession returns a new Mongo Session.
func NewMongoSession(mongoDBDialInfo *mgo.DialInfo, logger log.Logger, debug bool) (Session, log.Logger) {
	// Initialise mongodb connection and logger
	// Create a session which maintains a pool of socket connections to our MongoDB.
	mongoLogger := log.With(logger, "connection", "mongo")
	mongoLogger.Log("hosts", strings.Join(mongoDBDialInfo.Addrs, ", "))
	mongoLogger.Log("db", mongoDBDialInfo.Database)

	mongoSession, err := mgo.DialWithInfo(mongoDBDialInfo)

	// Can't connect? - bail!
	if err != nil {
		mongoLogger.Log("exit", err)
		panic(err)
	}

	// Optional. Switch the session to a monotonic behavior.
	mongoSession.SetMode(mgo.Monotonic, true)
	mongoLogger.Log("msg", "successfully connected")

	if debug {
		mongoLogger.Log("msg", "Turning on mongo debug logging ...")
		mgo.SetDebug(true)
		var debugMongoLogger *stdlog.Logger
		debugMongoLogger = stdlog.New(os.Stderr, "", stdlog.LstdFlags)
		mgo.SetLogger(debugMongoLogger)
	}

	// Wrap mgo session in user defined interface/structs
	// This means we can mock db calls more easily
	session := MongoSession{mongoSession}
	session.SetSafe(&mgo.Safe{})
	session.SetSyncTimeout(3 * time.Second)
	session.SetSocketTimeout(3 * time.Second)

	return session, mongoLogger
}

// PrepareDB ensure presence of persistent and immutable data in the DB.
func PrepareDB(session Session, db string, logger log.Logger) {
	indexes := make(map[string]mgo.Index)
	indexes["agents"] = mgo.Index{
		Key:        []string{"agentid"},
		Unique:     true,
		DropDups:   true,
		Background: false,
	}
	indexes["phonesessions"] = mgo.Index{
		Key:        []string{"sessid"},
		Unique:     true,
		DropDups:   true,
		Background: false,
	}

	for collectionName, index := range indexes {
		err := session.DB(db).C(collectionName).EnsureIndex(index)
		if err != nil {
			panic("Cannot ensure index.")
		}
	}
	logger.Log("Prepared database indexes.")
}
