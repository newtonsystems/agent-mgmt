package tests

// Test service layer
// useful ideas from https://medium.com/@povilasve/go-advanced-tips-tricks-a872503ac859

import (
	"context"
	//"fmt"
	"testing"

	"github.com/newtonsystems/agent-mgmt/app/service"
)

// GetAvailableAgents()
func TestAvailableAgentsReturnIDs(t *testing.T) {
	s := service.NewBasicService()
	sess := NewMockSession()

	value, err := s.GetAvailableAgents(context.Background(), sess, "fakedb")
	Ok(t, err)
	Equals(t, []string{"1", "2"}, value)
}

//func TestMongoConnection(t *testing.T) {
//	sess := CreateTestMongoConnection()
//	fmt.Printf("%s", sess)
//	defer sess.Close()
//}
