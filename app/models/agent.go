package models

// agent.go
// Agent Model / Mongo Calls

import (
	"errors"
	"time"

	//"github.com/go-kit/kit/log"

	agentmgmterrors "github.com/newtonsystems/agent-mgmt/app/errors"

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
func (db *MongoDatabase) HeartBeat(agentID int32) error {
	selector := bson.M{"agentid": agentID}
	update := bson.M{"$set": bson.M{"lastheartbeat": NowFunc()}}
	err := db.C("agents").Update(selector, update)

	return err
}

// GetAgents returns all Agents within a certain heartbeat
func (db *MongoDatabase) GetAgents(timestamp time.Time) ([]Agent, error) {
	var agents []Agent

	err := db.C("agents").Find(bson.M{"lastheartbeat": bson.M{"$gt": timestamp}}).Limit(10).All(&agents)

	return agents, agentmgmterrors.ErrAgentIDNotFoundError("This fhjksahfk sh sa")
	return agents, errors.New("This is an test error")

	if err != nil {
		return agents, err
	}
	return agents, nil
}
