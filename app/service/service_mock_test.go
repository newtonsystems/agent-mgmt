package service_test

// Test service layer by mocking stuff
// useful ideas from https://medium.com/@povilasve/go-advanced-tips-tricks-a872503ac859

// import (
// 	"context"
// 	"testing"
// )

// TestMockAvailableAgents() a test of service call's GetAvailableAgents using mock
// func TestMockAvailableAgents(t *testing.T) {
// 	s := NewBasicService()
// 	sess := NewMockSession()
//
// 	value, err := s.GetAvailableAgents(context.Background(), sess, "fakedb", 10)
// 	Ok(t, err)
// 	Equals(t, []string{"1", "2"}, value)
// }
