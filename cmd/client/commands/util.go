package commands

import (
	"flag"
	"fmt"
	"os"
)

func ClearScreen() {
	fmt.Print("\033[H\033[2J")
}

func PrintHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  status            - Get the cluster status")
	fmt.Println("  set <key> <value> - Add a key to the storage")
	fmt.Println("  rm <key>          - Remove a key from the storage")
	fmt.Println("  clear/cls         - Clear the screen")
	fmt.Println("  help              - Show this help message")
	fmt.Println("  exit/quit         - Exit the CLI")
	fmt.Println()
}

func ShowUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s -servers=host1:port1,host2:port2,... [operation] [arguments...]\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Operations:\n")
	fmt.Fprintf(os.Stderr, "  status            - Get the cluster status\n")
	fmt.Fprintf(os.Stderr, "  set <key> <value> - Add a key to the storage\n")
	fmt.Fprintf(os.Stderr, "  rm <key>          - Remove a key from the storage\n")
	fmt.Fprintf(os.Stderr, "If no operation is provided, the CLI will start in interactive mode.\n")
	flag.PrintDefaults()
}
