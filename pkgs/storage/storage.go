package storage

import (
	"fmt"
	"log/slog"
	"os"

	"bpicori/raft/pkgs/dto"

	"google.golang.org/protobuf/proto"
)

type Storage interface {
	PersistStateMachine(state *dto.StateMachineState) error
	LoadStateMachine() (*dto.StateMachineState, error)
}

type StorageImpl struct {
	ServerId             string
	StateMachineFilePath string
}

func NewStorage(serverId string, stateMachineFilePath string) Storage {
	return &StorageImpl{
		ServerId:             serverId,
		StateMachineFilePath: stateMachineFilePath,
	}
}

func (s *StorageImpl) PersistStateMachine(state *dto.StateMachineState) error {
	fileName := fmt.Sprintf("%s.pb", s.ServerId)
	filePath := fmt.Sprintf("%s/%s", s.StateMachineFilePath, fileName)

	slog.Debug("[STORAGE] Saving state to file", "path", filePath)

	data, err := proto.Marshal(state)
	if err != nil {
		slog.Error("[STORAGE] Error marshaling state to proto", "error", err)
		return err
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		slog.Error("[STORAGE] Error writing state to file", "error", err)
		return err
	}

	slog.Debug("[STORAGE] State saved to file", "path", filePath)
	return nil
}

func (s *StorageImpl) LoadStateMachine() (*dto.StateMachineState, error) {
	fileName := fmt.Sprintf("%s.pb", s.ServerId)
	filePath := fmt.Sprintf("%s/%s", s.StateMachineFilePath, fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// file does not exist, first time running the application
			return &dto.StateMachineState{}, nil
		}
		return nil, err
	}

	var state dto.StateMachineState
	err = proto.Unmarshal(data, &state)
	if err != nil {
		slog.Error("[STORAGE] Error unmarshaling state from proto", "error", err)
		return nil, err
	}

	slog.Debug("[STORAGE] State loaded from file", "path", filePath)
	return &state, nil
}
