SERVERS = "localhost:8080,localhost:8081,localhost:8082"
PERSISTENT_PATH = ./ignore
DEBUG = DEBUG=true
GO = go run
SRC = cmd/main.go
TMUX_NEW_WINDOW = tmux new-window -n "Raft"
TMUX_SPLIT_WINDOW = tmux split-window

run:
	$(TMUX_NEW_WINDOW) "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8080 -persistent-path=$(PERSISTENT_PATH) -http-port=7070" && \
	$(TMUX_SPLIT_WINDOW) -v "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8081 -persistent-path=$(PERSISTENT_PATH) -http-port=7071" && \
	$(TMUX_SPLIT_WINDOW) -h "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8082 -persistent-path=$(PERSISTENT_PATH) -http-port=7072"


clean:
	rm -rf $(PERSISTENT_PATH)/*
