package consts

import "fmt"

// RaftRPCType defines the type of Raft RPC message using an iota enum pattern.
type RaftRPCType int

// Role defines the role of a server in the cluster using an iota enum pattern.
type Role int

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
	OkResponse
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
	case OkResponse:
		return "OkResponse"
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
	case "OkResponse":
		return OkResponse, nil
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
