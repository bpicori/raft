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
	LpushCommand
	LpopCommand
	LindexCommand
	LlenCommand
	KeysCommand
	SaddCommand
	SremCommand
	SismemberCommand
	SinterCommand
	ScardCommand
	HsetCommand
	HgetCommand
	HmgetCommand
	HincrbyCommand
)

const (
	SetOp CommandType = iota
	DeleteOp
	IncrementOp
	DecrementOp
	LpushOp
	LpopOp
	LindexOp
	LlenOp
	SaddOp
	SremOp
	SismemberOp
	SinterOp
	ScardOp
	HsetOp
	HgetOp
	HmgetOp
	HincrbyOp
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
	case LpushCommand:
		return "LpushCommand"
	case LpopCommand:
		return "LpopCommand"
	case LindexCommand:
		return "LindexCommand"
	case LlenCommand:
		return "LlenCommand"
	case KeysCommand:
		return "KeysCommand"
	case SaddCommand:
		return "SaddCommand"
	case SremCommand:
		return "SremCommand"
	case SismemberCommand:
		return "SismemberCommand"
	case SinterCommand:
		return "SinterCommand"
	case ScardCommand:
		return "ScardCommand"
	case HsetCommand:
		return "HsetCommand"
	case HgetCommand:
		return "HgetCommand"
	case HmgetCommand:
		return "HmgetCommand"
	case HincrbyCommand:
		return "HincrbyCommand"
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
	case LpushOp:
		return "LPUSH"
	case LpopOp:
		return "LPOP"
	case LindexOp:
		return "LINDEX"
	case LlenOp:
		return "LLEN"
	case SaddOp:
		return "SADD"
	case SremOp:
		return "SREM"
	case SismemberOp:
		return "SISMEMBER"
	case SinterOp:
		return "SINTER"
	case ScardOp:
		return "SCARD"
	case HsetOp:
		return "HSET"
	case HgetOp:
		return "HGET"
	case HmgetOp:
		return "HMGET"
	case HincrbyOp:
		return "HINCRBY"
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
	case "LpushCommand":
		return LpushCommand, nil
	case "LpopCommand":
		return LpopCommand, nil
	case "LindexCommand":
		return LindexCommand, nil
	case "LlenCommand":
		return LlenCommand, nil
	case "KeysCommand":
		return KeysCommand, nil
	case "SaddCommand":
		return SaddCommand, nil
	case "SremCommand":
		return SremCommand, nil
	case "SismemberCommand":
		return SismemberCommand, nil
	case "SinterCommand":
		return SinterCommand, nil
	case "ScardCommand":
		return ScardCommand, nil
	case "HsetCommand":
		return HsetCommand, nil
	case "HgetCommand":
		return HgetCommand, nil
	case "HmgetCommand":
		return HmgetCommand, nil
	case "HincrbyCommand":
		return HincrbyCommand, nil
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
	case "LPUSH":
		return LpushOp
	case "LPOP":
		return LpopOp
	case "LINDEX":
		return LindexOp
	case "LLEN":
		return LlenOp
	case "SADD":
		return SaddOp
	case "SREM":
		return SremOp
	case "SISMEMBER":
		return SismemberOp
	case "SINTER":
		return SinterOp
	case "SCARD":
		return ScardOp
	case "HSET":
		return HsetOp
	case "HGET":
		return HgetOp
	case "HMGET":
		return HmgetOp
	case "HINCRBY":
		return HincrbyOp
	default:
		return SetOp
	}
}
