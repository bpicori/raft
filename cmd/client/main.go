package main

import (
	"bpicori/raft/pkgs/logger"
	"fmt"
)



func init() {
  logger.LogSetup()
}

func main(){
  fmt.Println("Hello, World!")
}
