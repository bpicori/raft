package main

import (
	"bpicori/raft"
	"fmt"
	"os"
)

func init() {
	os.Setenv("RAFT_SERVERS", "localhost:8080,localhost:8081,localhost:8082")
	os.Setenv("CURRENT_SERVER", "localhost:8080")
}

func main() {

	_, err := raft.NewServer()
	if err != nil {
		panic(err)
	}

	// server2 := raft.NewServer(2)
	// server3 := raft.NewServer(3)

	for {
		// Block forever
	}
	fmt.Println("END")
}
