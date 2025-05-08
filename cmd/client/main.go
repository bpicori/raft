package main

import (
	"bpicori/raft/pkgs/config"
	"bpicori/raft/pkgs/logger"
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type flagValues struct {
	servers string
}

func init() {
	logger.LogSetup()
}

func main() {
	flagValues, operation, args := parseFlags()
	cfg := buildServerConfig(flagValues.servers)

	if operation == "" {
		runInteractiveMode(cfg)
		return
	}

	executeCommand(cfg, operation, args)
}

func runInteractiveMode(cfg *config.Config) {
	fmt.Println("--------------------------------")
	fmt.Println("Raft CLI")
	fmt.Println("--------------------------------")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		if input == "help" {
			printHelp()
			continue
		}

		args := strings.Fields(input)
		if len(args) == 0 {
			continue
		}

		operation := args[0]
		operationArgs := args[1:]

		handleCommandExecution(cfg, operation, operationArgs)
	}
}

func executeCommand(cfg *config.Config, operation string, args []string) {
	switch operation {
	case "status":
		GetClusterStatus(cfg)
	case "set":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Key and value are required for set operation\n")
			showUsage()
			os.Exit(1)
		}
		key := args[0]
		value := args[1]

		SetCommand(cfg, key, value)
	case "rm":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Key is required for rm operation\n")
			showUsage()
			os.Exit(1)
		}
		key := args[0]
		fmt.Printf("Removing key '%s' (not implemented)\n", key)
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", operation)
		showUsage()
		os.Exit(1)
	}
}

func handleCommandExecution(cfg *config.Config, operation string, args []string) {
	// handle panics
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Command execution error: %v\n", r)
		}
	}()

	// Don't exit the program on errors in interactive mode
	switch operation {
	case "status":
		GetClusterStatus(cfg)
	case "set":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Key and value are required for set operation\n")
			return
		}
		key := args[0]
		value := args[1]
		SetCommand(cfg, key, value)
	case "rm":
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Key is required for rm operation\n")
			return
		}
		key := args[0]
		fmt.Printf("Removing key '%s' (not implemented)\n", key)
	case "clear", "cls":
		clearScreen()
	case "help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s (type 'help' for available commands)\n", operation)
	}
}

func parseFlags() (flagValues, string, []string) {
	servers := flag.String("servers", "", "Comma-separated list of servers in format host:port")

	// Define custom usage function
	flag.Usage = showUsage

	flag.Parse()

	if *servers == "" {
		fmt.Fprintf(os.Stderr, "Servers list is required\n")
		showUsage()
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		return flagValues{
			servers: *servers,
		}, "", nil
	}

	operation := args[0]
	operationArgs := args[1:]

	return flagValues{
		servers: *servers,
	}, operation, operationArgs
}

func buildServerConfig(serversStr string) *config.Config {
	serverList := strings.Split(serversStr, ",")
	if len(serverList) == 0 {
		slog.Error("No servers provided")
		os.Exit(1)
	}

	serverConfigs := make(map[string]config.ServerConfig)
	for _, addr := range serverList {
		id := addr
		serverConfigs[id] = config.ServerConfig{
			ID:   id,
			Addr: addr,
		}
	}

	return &config.Config{
		Servers: serverConfigs,
	}
}
