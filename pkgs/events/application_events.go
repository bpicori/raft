package events

import "bpicori/raft/pkgs/dto"

type SetCommandEvent struct {
	Payload *dto.SetCommandRequest
	Reply   chan *dto.GenericResponse
}

type GetCommandEvent struct {
	Payload *dto.GetCommandRequest
	Reply   chan *dto.GetCommandResponse
}

type SyncCommandEvent struct {
	LogEntry *dto.LogEntry
}

type NodeStatusEvent struct {
	Reply chan *dto.NodeStatusResponse
}

type IncrCommandEvent struct {
	Payload *dto.IncrCommandRequest
	Reply   chan *dto.IncrCommandResponse
}

type DecrCommandEvent struct {
	Payload *dto.DecrCommandRequest
	Reply   chan *dto.DecrCommandResponse
}

type RemoveCommandEvent struct {
	Payload *dto.RemoveCommandRequest
	Reply   chan *dto.GenericResponse
}

type LpushCommandEvent struct {
	Payload *dto.LpushCommandRequest
	Reply   chan *dto.LpushCommandResponse
}

type LpopCommandEvent struct {
	Payload *dto.LpopCommandRequest
	Reply   chan *dto.LpopCommandResponse
}

type LindexCommandEvent struct {
	Payload *dto.LindexCommandRequest
	Reply   chan *dto.LindexCommandResponse
}

type LlenCommandEvent struct {
	Payload *dto.LlenCommandRequest
	Reply   chan *dto.LlenCommandResponse
}

type KeysCommandEvent struct {
	Payload *dto.KeysCommandRequest
	Reply   chan *dto.KeysCommandResponse
}

type SaddCommandEvent struct {
	Payload *dto.SaddCommandRequest
	Reply   chan *dto.SaddCommandResponse
}

type SremCommandEvent struct {
	Payload *dto.SremCommandRequest
	Reply   chan *dto.SremCommandResponse
}

type SismemberCommandEvent struct {
	Payload *dto.SismemberCommandRequest
	Reply   chan *dto.SismemberCommandResponse
}

type SinterCommandEvent struct {
	Payload *dto.SinterCommandRequest
	Reply   chan *dto.SinterCommandResponse
}

type ScardCommandEvent struct {
	Payload *dto.ScardCommandRequest
	Reply   chan *dto.ScardCommandResponse
}

type HsetCommandEvent struct {
	Payload *dto.HsetCommandRequest
	Reply   chan *dto.HsetCommandResponse
}

type HgetCommandEvent struct {
	Payload *dto.HgetCommandRequest
	Reply   chan *dto.HgetCommandResponse
}

type HmgetCommandEvent struct {
	Payload *dto.HmgetCommandRequest
	Reply   chan *dto.HmgetCommandResponse
}

type HincrbyCommandEvent struct {
	Payload *dto.HincrbyCommandRequest
	Reply   chan *dto.HincrbyCommandResponse
}
