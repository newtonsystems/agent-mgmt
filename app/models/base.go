package models

// base.go
// Interface wrapper for mongo mgo
// gracefully ripped from: https://github.com/thylong/regexrace/blob/master/models/base.go

import (
	"fmt"
	stdlog "log"
	"os"
	"strings"
	"time"

	//log "github.com/Sirupsen/logrus"
	"github.com/go-kit/kit/log"

	//"github.com/spf13/viper"
	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
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
	UpdateId(id interface{}, update interface{}) error
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
	AddTask(custID int32, agentIDs []int32) (int32, error)
	AgentExists(agentID int32) (bool, error)
	GetAgents(timestamp time.Time, limit int32) ([]Agent, error)
	GetAgentIDFromRef(refID string) (int32, error)
	HeartBeat(agentID int32) error
	DropDatabase() error
	GetNextSequence(name string) (int32, error)
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
	SetMode(consistency mgo.Mode, refresh bool)
	SetSocketTimeout(d time.Duration)
	Close()
	Clone() Session
	Refresh()
	Copy() Session
	Ping() error
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

// Clone mocks mgo.Session.Clone()
func (s MongoSession) Clone() Session {
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

func (s MongoSession) SetMode(consistency mgo.Mode, refresh bool) {
	s.Session.SetMode(consistency, refresh)
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
	mongoSession.SetMode(mgo.Strong, true)
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

type Count struct {
	ID  string `bson:"_id"`
	Seq int32  `bson:"seq"`
}

// GetNextSequence returns the next sequence for 'name'
func (db *MongoDatabase) GetNextSequence(name string) (int32, error) {
	var doc Count
	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"seq": 1}},
		Upsert:    true,
		ReturnNew: true,
	}

	//println(name)
	count, err := db.C("counters").Find(bson.M{"_id": name}).Count()

	if err != nil {
		logger.Log("level", "error", "msg", "Failed in get count of sequence name: "+name)
		return 0, err
	}

	if count == 0 {
		return 0, amerrors.ErrCounterNotFoundError("failed to find an counter counters(_id=" + name + ")")
	}

	_, err = db.C("counters").Find(bson.M{"_id": name}).Apply(change, &doc)
	//fmt.Println(doc)
	if err != nil {
		panic("Creation of next sequence failed for " + name + " error: " + err.Error())
	}

	return doc.Seq, nil
}

// PrepareDB ensure presence of persistent and immutable data in the DB.
func PrepareDB(session Session, db string, logger log.Logger) {
	sessCopy := session.Copy()
	defer sessCopy.Close()

	logger.Log("level", "info", "tag", "#beforeprepare", "msg", "stats: "+fmt.Sprintf("%#v", mgo.GetStats()))

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
		err := sessCopy.DB(db).C(collectionName).EnsureIndex(index)
		if err != nil {
			panic("Cannot ensure index for " + collectionName + " error: " + err.Error())
		}
	}
	logger.Log("level", "info", "msg", "Prepared database indexes.")

	logger.Log("level", "info", "msg", "Setting up counters ...")
	logger.Log("level", "debug", "msg", "Setting up taskid")

	err := sessCopy.DB(db).C("counters").Insert(bson.M{
		"_id": "taskid",
		"seq": 1,
	})

	if err != nil {
		logger.Log("level", "error", "msg", "Failed to set the taskid to its initial value")
		panic(err)
	}

	logger.Log("level", "info", "tag", "#prepared", "msg", "stats: "+fmt.Sprintf("%#v", mgo.GetStats()))
}
