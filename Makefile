PHONY: install-tools install run srv1 srv2 srv3 clean

SERVERS = "localhost:8080,localhost:8081,localhost:8082"
PERSISTENT_PATH = ./ignore
DEBUG = DEBUG=false
GO = go run
SRC = cmd/main.go
TMUX_NEW_WINDOW = tmux new-window -n "Raft"
TMUX_SPLIT_WINDOW = tmux split-window
HEARTBEAT = 1000
TIMEOUT_MIN = 3000
TIMEOUT_MAX = 5000
PROTO_SRC = raft.proto
PROTO_OUT = .

install-tools:
	@echo "Installing protoc-gen-go..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "protoc-gen-go installed."

install:
	make install-tools
	go mod tidy

proto-build:
	@echo "Compiling protobuf files..."
	protoc --go_out=. dto/raft.proto
	@echo "Protobuf compilation complete."

build:
	@echo "Building the project..."
	mkdir -p bin
	make proto-build
	go build -o bin/raft cmd/main.go
	@echo "Project built."

run:
	make proto-build
	$(TMUX_NEW_WINDOW) "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8080 -persistent-path=$(PERSISTENT_PATH) -http-port=7070 -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)" && \
	$(TMUX_SPLIT_WINDOW) -v "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8081 -persistent-path=$(PERSISTENT_PATH) -http-port=7071 -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)" && \
	$(TMUX_SPLIT_WINDOW) -h "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8082 -persistent-path=$(PERSISTENT_PATH) -http-port=7072 -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)"

srv1:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8080 -persistent-path=$(PERSISTENT_PATH) -http-port=7070 -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

srv2:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8081 -persistent-path=$(PERSISTENT_PATH) -http-port=7071 -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

srv3:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8082 -persistent-path=$(PERSISTENT_PATH) -http-port=7072 -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

clean:
	rm -rf $(PERSISTENT_PATH)/*
