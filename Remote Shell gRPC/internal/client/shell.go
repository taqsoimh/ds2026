package client

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	pb "remote-shell-rpc/proto"
)

// ShellConfig holds configuration for the interactive shell
type ShellConfig struct {
	Prompt      string
	HistorySize int
}

// DefaultShellConfig returns the default shell configuration
func DefaultShellConfig() ShellConfig {
	return ShellConfig{
		Prompt:      "remote> ",
		HistorySize: 100,
	}
}

// Shell represents an interactive shell interface
type Shell struct {
	client  *Client
	config  ShellConfig
	history []string
	running bool
}

// NewShell creates a new interactive shell
func NewShell(client *Client, cfg ShellConfig) *Shell {
	return &Shell{
		client:  client,
		config:  cfg,
		history: make([]string, 0, cfg.HistorySize),
		running: false,
	}
}

// Run starts the interactive shell loop
func (s *Shell) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	s.running = true

	s.printWelcome()

	for s.running {
		// Print prompt
		fmt.Print(s.config.Prompt)

		// Read input
		input, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" {
				fmt.Println("\nGoodbye!")
				break
			}
			return fmt.Errorf("failed to read input: %w", err)
		}

		// Trim whitespace
		input = strings.TrimSpace(input)

		// Skip empty input
		if input == "" {
			continue
		}

		// Add to history
		s.addToHistory(input)

		// Handle command
		if err := s.handleCommand(ctx, input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}

	return nil
}

// Stop stops the interactive shell
func (s *Shell) Stop() {
	s.running = false
}

// handleCommand processes a command
func (s *Shell) handleCommand(ctx context.Context, input string) error {
	// Handle local commands
	switch strings.ToLower(input) {
	case "exit", "quit":
		fmt.Println("Goodbye!")
		s.running = false
		return nil

	case "clear":
		// Clear screen
		fmt.Print("\033[2J\033[H")
		return nil

	case "help":
		s.printHelp()
		return nil

	case "history":
		s.printHistory()
		return nil

	case "status":
		s.printStatus()
		return nil
	}

	// Execute remote command with streaming
	return s.executeRemoteCommand(ctx, input)
}

// executeRemoteCommand executes a command on the remote server
func (s *Shell) executeRemoteCommand(ctx context.Context, command string) error {
	outputHandler := func(output *pb.CommandOutput) {
		if output.IsComplete {
			// Command completed
			if output.ExitCode != 0 {
				fmt.Fprintf(os.Stderr, "[Exit code: %d]\n", output.ExitCode)
			}
			return
		}

		// Print output
		if output.Type == pb.CommandOutput_STDERR {
			fmt.Fprint(os.Stderr, string(output.Data))
		} else {
			fmt.Print(string(output.Data))
		}
	}

	return s.client.ExecuteCommandStream(ctx, command, 30, outputHandler)
}

// addToHistory adds a command to the history
func (s *Shell) addToHistory(cmd string) {
	if len(s.history) >= s.config.HistorySize {
		s.history = s.history[1:]
	}
	s.history = append(s.history, cmd)
}

// printWelcome prints the welcome message
func (s *Shell) printWelcome() {
	fmt.Println("╔════════════════════════════════════════════════════╗")
	fmt.Println("║       Remote Shell RPC Client - Group 15           ║")
	fmt.Println("║────────────────────────────────────────────────────║")
	fmt.Println("║  Type 'help' for available commands                ║")
	fmt.Println("║  Type 'exit' or 'quit' to disconnect               ║")
	fmt.Println("╚════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Session ID: %s\n", s.client.GetSessionID())
	fmt.Println()
}

// printHelp prints the help message
func (s *Shell) printHelp() {
	fmt.Println("\nAvailable Commands:")
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println("  help     - Show this help message")
	fmt.Println("  exit     - Disconnect and exit")
	fmt.Println("  quit     - Same as exit")
	fmt.Println("  clear    - Clear the screen")
	fmt.Println("  history  - Show command history")
	fmt.Println("  status   - Show connection status")
	fmt.Println()
	fmt.Println("All other commands are executed on the remote server.")
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println()
}

// printHistory prints the command history
func (s *Shell) printHistory() {
	fmt.Println("\nCommand History:")
	fmt.Println("───────────────────────────────────────────────────")
	for i, cmd := range s.history {
		fmt.Printf("  %3d  %s\n", i+1, cmd)
	}
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println()
}

// printStatus prints the connection status
func (s *Shell) printStatus() {
	fmt.Println("\nConnection Status:")
	fmt.Println("───────────────────────────────────────────────────")
	if s.client.IsConnected() {
		fmt.Println("  Connected: Yes")
	} else {
		fmt.Println("  Connected: No")
	}
	if s.client.HasSession() {
		fmt.Printf("  Session ID: %s\n", s.client.GetSessionID())
	} else {
		fmt.Println("  Session ID: None")
	}
	fmt.Println("───────────────────────────────────────────────────")
	fmt.Println()
}
