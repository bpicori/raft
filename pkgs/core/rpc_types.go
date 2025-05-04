package core

import "fmt"

// RaftRPCType defines the type of Raft RPC message using an iota enum pattern.
type RaftRPCType int

const (
	UnknownRPCType RaftRPCType = iota
	VoteRequest
	VoteResponse
	AppendEntriesReqType
	AppendEntriesRespType
	ClusterStateType
)

// String method for RaftRPCType for logging/debugging
func (rt RaftRPCType) String() string {
	switch rt {
	case VoteRequest:
		return "VoteRequest"
	case VoteResponse:
		return "VoteResponse"
	case AppendEntriesReqType:
		return "AppendEntriesReq"
	case AppendEntriesRespType:
		return "AppendEntriesResp"
	case ClusterStateType:
		return "ClusterState"
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
	case "AppendEntriesReq":
		return AppendEntriesReqType, nil
	case "AppendEntriesResp":
		return AppendEntriesRespType, nil
	case "ClusterState":
		return ClusterStateType, nil
	default:
		return UnknownRPCType, fmt.Errorf("unknown RPC type string: %s", rpcTypeStr)
	}
}
