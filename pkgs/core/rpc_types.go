package core

import "fmt"

// RaftRPCType defines the type of Raft RPC message using an iota enum pattern.
type RaftRPCType int

const (
	UnknownRPCType RaftRPCType = iota
	RequestVoteReqType
	RequestVoteRespType
	AppendEntriesReqType
	AppendEntriesRespType
	ClusterStateType
)

// String method for RaftRPCType for logging/debugging
func (rt RaftRPCType) String() string {
	switch rt {
	case RequestVoteReqType:
		return "RequestVoteReq"
	case RequestVoteRespType:
		return "RequestVoteResp"
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
	case "RequestVoteReq":
		return RequestVoteReqType, nil
	case "RequestVoteResp":
		return RequestVoteRespType, nil
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
