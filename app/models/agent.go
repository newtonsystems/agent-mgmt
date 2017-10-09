package models

// agent.go
// Agent Model / Mongo Calls

import (
	"time"

	//"github.com/go-kit/kit/log"

	//"github.com/newtonsystems/agent-mgmt/app/models"
	"github.com/newtonsystems/agent-mgmt/app/utils"
	"gopkg.in/mgo.v2/bson"
)

var logger = utils.GetLogger()

type Agent struct {
	AgentID       int32     `bson:"agentid" json:"agentid"`
	LastHeartBeat time.Time `bson:"lastheartbeat" json:"lastheartbeat"`
}

// Mongo Calls

// GetAgents returns all Agents within a certain heartbeat
func (db *MongoDatabase) GetAgents(timestamp time.Time) ([]Agent, error) {
	var agents []Agent

	err := db.C("agents").Find(bson.M{"lastheartbeat": bson.M{"$gt": timestamp}}).Limit(10).All(&agents)

	if err != nil {
		return agents, err
	}
	return agents, nil
}
