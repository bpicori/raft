#!/bin/bash

SERVERS="localhost:8080,localhost:8081,localhost:8082"
PERSISTENT_PATH="./ignore"

usage() {
  echo "Usage: $0 {run-tmux|clean}"
  exit 1
}

run() {
  tmux new-window -n "Raft" "go run cmd/main.go -servers=$SERVERS -current=localhost:8080 -persistent-path=$PERSISTENT_PATH" &&
    tmux split-window -v "go run cmd/main.go -servers=$SERVERS -current=localhost:8081 -persistent-path=$PERSISTENT_PATH" && tmux split-window -h "go run cmd/main.go -servers=$SERVERS -current=localhost:8082 -persistent-path=$PERSISTENT_PATH"
}

clean() {
  echo "Cleaning up..."
  rm -r ./ignore/*
}

if [ $# -eq 0 ]; then
  usage
fi

case "$1" in
run)
  run
  ;;
clean)
  clean
  ;;
*)
  usage
  ;;
esac
