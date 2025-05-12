package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"log/slog"
)

func Get(eventManager *events.EventManager, getCommandEvent *events.GetCommandEvent) {
	slog.Debug("[APPLICATION] Received get command", "command", getCommandEvent.Payload)

	value, ok := hashMap.Load(getCommandEvent.Payload.Key)
	if !ok {
		getCommandEvent.Reply <- &dto.GetCommandResponse{Value: ""}
		return
	}

	getCommandEvent.Reply <- &dto.GetCommandResponse{Value: value.(string)}
}
