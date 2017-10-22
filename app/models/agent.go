package models

// agent.go
// Agent Model / Mongo Calls

import (
	// "errors"

	"strconv"
	"time"

	//"github.com/go-kit/kit/log"

	amerrors "github.com/newtonsystems/agent-mgmt/app/errors"

	"github.com/newtonsystems/agent-mgmt/app/utils"
	"gopkg.in/mgo.v2/bson"
)

var logger = utils.GetLogger()

type nowFuncT func() time.Time

var NowFunc nowFuncT

func init() {
	NowFunc = func() time.Time {
		return time.Now()
	}
}

type Agent struct {
	AgentID       int32     `bson:"agentid" json:"agentid"`
	LastHeartBeat time.Time `bson:"lastheartbeat" json:"lastheartbeat"`
}

// Mongo Calls

// HeartBeat updates LastHeartBeat with current time now
func (db *MongoDatabase) AgentExists(agentID int32) (bool, error) {
	count, err := db.C("agents").Find(bson.M{"agentid": agentID}).Count()

	if err != nil {
		return false, err
	}

	if count == 0 {
		return false, amerrors.ErrAgentNotFoundError("failed to find an Agent(AgentID=" + strconv.Itoa(int(agentID)) + ")")
	}

	return true, nil
}

// HeartBeat updates LastHeartBeat with current time now
func (db *MongoDatabase) HeartBeat(agentID int32) error {
	selector := bson.M{"agentid": agentID}
	update := bson.M{"$set": bson.M{"lastheartbeat": NowFunc()}}
	err := db.C("agents").Update(selector, update)

	return err
}

// GetAgents returns all Agents within a certain heartbeat
func (db *MongoDatabase) GetAgents(timestamp time.Time, limit int32) ([]Agent, error) {
	var agents []Agent

	err := db.C("agents").Find(bson.M{"lastheartbeat": bson.M{"$gt": timestamp}}).Limit(int(limit)).All(&agents)

	//return agents, amerrors.ErrAgentIDNotFoundError("This fhjksahfk sh sa")
	//return agents, errors.New("This is an test error")

	if err != nil {
		return agents, err
	}
	return agents, nil
}
