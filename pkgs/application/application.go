package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"log/slog"
	"sync"
	"time"

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
			slog.Debug("[APPLICATION] Received set command", "command", setCommandEvent.Payload)

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

			go func() {
				select {
				case <-ch:
					slog.Debug("[APPLICATION] Received response from append log entry", "uuid", uuid)
					setCommandEvent.Reply <- &dto.OkResponse{Ok: true}
				case <-time.After(5 * time.Second):
					slog.Error("[APPLICATION] No response from append log entry", "uuid", uuid)
					setCommandEvent.Reply <- &dto.OkResponse{Ok: false}
				}
			}()
		}
	}
}
