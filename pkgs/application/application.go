package application

import (
	"bpicori/raft/pkgs/dto"
	"bpicori/raft/pkgs/events"
	"context"
	"log/slog"
	"sync"
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
			setCommandEvent.Reply <- &dto.OkResponse{Ok: true}
		}
	}
}
