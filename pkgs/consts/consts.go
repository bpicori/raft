package consts

import "fmt"

// RaftRPCType defines the type of Raft RPC message using an iota enum pattern.
type RaftRPCType int

// Role defines the role of a server in the cluster using an iota enum pattern.
type Role int

// CommandType defines the type of command operations using an iota enum pattern.
type CommandType int

const (
	Follower Role = iota
	Candidate
	Leader
)

const (
	UnknownRPCType RaftRPCType = iota
	VoteRequest
	VoteResponse
	LogRequest
	LogResponse
	NodeStatus
	SetCommand
	GetCommand
	GenericResponse
	IncrCommand
	DecrCommand
	RemoveCommand
)

const (
	SetOp CommandType = iota
	DeleteOp
	IncrementOp
	DecrementOp
)


// String method for RaftRPCType for logging/debugging
func (rt RaftRPCType) String() string {
	switch rt {
	case VoteRequest:
		return "VoteRequest"
	case VoteResponse:
		return "VoteResponse"
	case LogRequest:
		return "LogRequest"
	case LogResponse:
		return "LogResponse"
	case NodeStatus:
		return "NodeStatus"
	case SetCommand:
		return "SetCommand"
	case GetCommand:
		return "GetCommand"
	case GenericResponse:
		return "GenericResponse"
	case IncrCommand:
		return "IncrCommand"
	case DecrCommand:
		return "DecrCommand"
	case RemoveCommand:
		return "RemoveCommand"
	default:
		return "Unknown"
	}
}

// String method for CommandType for logging/debugging
func (ct CommandType) String() string {
	switch ct {
	case SetOp:
		return "SET"
	case DeleteOp:
		return "DELETE"
	case IncrementOp:
		return "INCREMENT"
	case DecrementOp:
		return "DECREMENT"
	default:
		return "Unknown"
	}
}

// MapStringToRPCType converts the protobuf string type to our internal enum.
func MapStringToRPCType(rpcTypeStr string) (RaftRPCType, error) {
	switch rpcTypeStr {
	case "VoteRequest":
		return VoteRequest, nil
	case "VoteResponse":
		return VoteResponse, nil
	case "LogRequest":
		return LogRequest, nil
	case "LogResponse":
		return LogResponse, nil
	case "NodeStatus":
		return NodeStatus, nil
	case "SetCommand":
		return SetCommand, nil
	case "GetCommand":
		return GetCommand, nil
	case "IncrCommand":
		return IncrCommand, nil
	case "DecrCommand":
		return DecrCommand, nil
	case "RemoveCommand":
		return RemoveCommand, nil
	case "GenericResponse":
		return GenericResponse, nil
	default:
		return UnknownRPCType, fmt.Errorf("unknown RPC type string: %s", rpcTypeStr)
	}
}

// MapRoleToString converts the role enum to a string.
func MapRoleToString(role Role) string {
	switch role {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// MapCommandTypeToString converts our internal CommandType to the proto string value.
func MapCommandTypeToString(ct CommandType) string {
	return ct.String()
}

// MapStringToCommandType converts the proto string to our internal CommandType.
func MapStringToCommandType(opString string) CommandType {
	switch opString {
	case "SET":
		return SetOp
	case "DELETE":
		return DeleteOp
	case "INCREMENT":
		return IncrementOp
	case "DECREMENT":
		return DecrementOp
	default:
		return SetOp
	}
}
