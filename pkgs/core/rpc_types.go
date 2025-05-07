package core

import "fmt"

// RaftRPCType defines the type of Raft RPC message using an iota enum pattern.
type RaftRPCType int

const (
	UnknownRPCType RaftRPCType = iota
	VoteRequest
	VoteResponse
	LogRequest
	LogResponse
	NodeStatus
	AddCommand
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
	case AddCommand:
		return "AddCommand"
	default:
		return "Unknown"
	}
}

// mapStringToRPCType converts the protobuf string type to our internal enum.
func mapStringToRPCType(rpcTypeStr string) (RaftRPCType, error) {
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
	case "AddCommand":
		return AddCommand, nil
	default:
		return UnknownRPCType, fmt.Errorf("unknown RPC type string: %s", rpcTypeStr)
	}
}
