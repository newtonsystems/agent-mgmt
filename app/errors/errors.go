package errors

import "fmt"

// ErrorType provides a coarse category for BoulderErrors
type ErrorType int

const (
	InternalServer ErrorType = iota
	_
	ErrAgentIDNotFound
)

// AgentMgmtError represents internal Boulder errors
type AgentMgmtError struct {
	Type   ErrorType
	Detail string
}

func (be *AgentMgmtError) Error() string {
	return be.Detail
}

// New is a convenience function for creating a new AgentMgmtError
func New(errType ErrorType, msg string, args ...interface{}) error {
	return &AgentMgmtError{
		Type:   errType,
		Detail: fmt.Sprintf(msg, args...),
	}
}

// StrName is a convenience function for getting the string constant name
func StrName(errType ErrorType) string {
	switch errType {
	case InternalServer:
		return "InternalServer"
	case ErrAgentIDNotFound:
		return "ErrAgentIDNotFound"
	}
	return fmt.Sprintf("%v", errType)
}

// Is is a convenience function for testing the internal type of an AgentMgmtError
func Is(err error, errType ErrorType) bool {
	bErr, ok := err.(*AgentMgmtError)
	if !ok {
		return false
	}
	return bErr.Type == errType
}

func InternalServerError(msg string, args ...interface{}) error {
	return New(InternalServer, msg, args...)
}

// ErrAgentIDNotFoundError returns when we cant find an agent ID
func ErrAgentIDNotFoundError(msg string, args ...interface{}) error {
	return New(ErrAgentIDNotFound, msg, args...)
}
