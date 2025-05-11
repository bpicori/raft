package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

func Start(eventManager *events.EventManager, ctx context.Context, wg *sync.WaitGroup) {
	slog.Info("[APPLICATION] Starting application")

	for {
		select {
		case <-ctx.Done():
			slog.Info("[APPLICATION] Context done, shutting down application")
			wg.Done()
			return
		case setCommandEvent := <-eventManager.SetCommandRequestChan:
			slog.Info("[APPLICATION] Received set command", "command", setCommandEvent.Payload)
			// TODO: implement set command

			uuid := uuid.New().String()
			ch := make(chan bool)

			appendLogEntryEvent := events.AppendLogEntryEvent{
				Command: &dto.Command{
					Operation: dto.CommandOperation_SET,
					Args: &dto.Command_SetCommand{
						SetCommand: &dto.SetCommand{
							Key:   setCommandEvent.Payload.Key,
							Value: setCommandEvent.Payload.Value,
						},
					},
				},
				Uuid:  uuid,
				Reply: ch,
			}
			eventManager.AppendLogEntryChan <- appendLogEntryEvent

			setCommandEvent.Reply <- &dto.OkResponse{Ok: true}
		}
	}
}
