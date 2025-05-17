PHONY: install-tools install run srv1 srv2 srv3 srv4 srv5 clean client test test-verbose test-coverage test-coverage-html

SERVERS = "localhost:8080,localhost:8081,localhost:8082,localhost:8083,localhost:8084"
PERSISTENT_PATH = ./ignore
DEBUG = DEBUG=true
GO = go run
SRC = cmd/server/main.go
SRC_CLIENT = cmd/client/*.go
TMUX_NEW_WINDOW = tmux new-window -n "Raft"
TMUX_SPLIT_WINDOW = tmux split-window
HEARTBEAT = 1000
TIMEOUT_MIN = 3000
TIMEOUT_MAX = 5000
PROTO_SRC = raft.proto
PROTO_OUT = .
CMD_ARGS = $(filter-out $@,$(MAKECMDGOALS))

# Define test flags to prevent flag redefinition errors
TEST_FLAGS = -timeout-min=3000 -timeout-max=5000 -heartbeat=1000

install-tools:
	@echo "Installing protoc-gen-go..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@echo "protoc-gen-go installed."

install:
	make install-tools
	go mod tidy

proto-build:
	@echo "Compiling protobuf files..."
	protoc --go_out=. pkgs/dto/raft.proto
	@echo "Protobuf compilation complete."

build:
	@echo "Building the project..."
	mkdir -p bin
	make proto-build
	go build -o bin/raft cmd/server/main.go
	go build -o bin/client cmd/client/main.go
	@echo "Project built."

run:
	make proto-build
	$(TMUX_NEW_WINDOW) "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8080 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)" && \
	$(TMUX_SPLIT_WINDOW) -v "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8081 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)" && \
	$(TMUX_SPLIT_WINDOW) -h "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8082 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)" && \
	$(TMUX_SPLIT_WINDOW) -v "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8083 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)" && \
	$(TMUX_SPLIT_WINDOW) -h "$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8084 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)"

srv1:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8080 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

srv2:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8081 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

srv3:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8082 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

srv4:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8083 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

srv5:
	$(DEBUG) $(GO) $(SRC) -servers=$(SERVERS) -current=localhost:8084 -persistent-path=$(PERSISTENT_PATH) -timeout-min=$(TIMEOUT_MIN) -timeout-max=$(TIMEOUT_MAX) -heartbeat=$(HEARTBEAT)

client:
	$(DEBUG) $(GO) $(SRC_CLIENT) -servers=$(SERVERS) $(CMD_ARGS)

clean:
	rm -rf $(PERSISTENT_PATH)/*

kill:
	kill -15 $(shell lsof -t -i:8080)
	kill -15 $(shell lsof -t -i:8081)
	kill -15 $(shell lsof -t -i:8082)
	kill -15 $(shell lsof -t -i:8083)
	kill -15 $(shell lsof -t -i:8084)

# Run all tests (ignoring errors)
test:
	@echo "Running all tests..."
	-go test -race ./...

# Run all tests with verbose output (ignoring errors)
test-verbose:
	@echo "Running all tests with verbose output..."
	-go test -v -race ./...

test-raft:
	@echo "Running raft tests..."
	-go test -race ./pkgs/raft/...

# This allows passing arguments to the make command without errors
%:
	@:
