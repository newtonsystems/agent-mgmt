package models

// session.go
// Session Model / Mongo Calls

import (
	"fmt"

	"gopkg.in/mgo.v2/bson"
)

// NOTE: MixedCaps (Go public!)
type PhoneSession struct {
	SessID  int32  `bson:"sessid" json:"sessid"`
	AgentID int32  `bson:"agentid" json:"agentid"`
	RefID   string `bson:"refid" json:"refid"`
}

//	AgentID int32 `bson:"agentid" json:"agentid"` // Agent ID (make foreign key)
//	//PhoneID int32  `bson:"phoneid" json:"phoneid"` // Phone ID (make foreign key)
//	refID string `bson:"refid" json:"refid"` // Ref ID
//	//LastHeartBeat time.Time `bson:"lastheartbeat" json:"lastheartbeat"`

// Mongo Calls

// GetAgentIDFromRef returns the Agent ID from a Reference
func (db *MongoDatabase) GetAgentIDFromRef(refID string) (int32, error) {
	var pSess PhoneSession

	err := db.C("phonesessions").Find(bson.M{"refid": refID}).Select(bson.M{"agentid": 1}).One(&pSess)

	logger.Log("level", "debug", "msg", "Found agent ID: "+fmt.Sprintf("%#v", pSess.AgentID))

	if err != nil {
		// Is returning 0 really the best way to handle this?
		// What is the best way to handle error (raise something?)
		return 0, err
	}

	return pSess.AgentID, nil
}
