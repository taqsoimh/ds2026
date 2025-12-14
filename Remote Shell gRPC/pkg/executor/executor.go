package executor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Common errors
var (
	ErrCommandTimeout  = errors.New("command execution timeout")
	ErrCommandKilled   = errors.New("command was killed")
	ErrInvalidCommand  = errors.New("invalid command")
	ErrEmptyCommand    = errors.New("empty command")
	ErrCommandNotFound = errors.New("command not found")
)

// OutputType represents the type of command output
type OutputType int

const (
	Stdout OutputType = iota
	Stderr
)

// Output represents a piece of command output
type Output struct {
	Type       OutputType
	Data       []byte
	IsComplete bool
	ExitCode   int
}

// Result represents the complete result of a command execution
type Result struct {
	Output        string
	Error         string
	ExitCode      int
	ExecutionTime time.Duration
}

// Config holds executor configuration
type Config struct {
	Shell          string
	DefaultTimeout time.Duration
	WorkingDir     string
	Environment    []string
}

// DefaultConfig returns the default executor configuration
func DefaultConfig() Config {
	return Config{
		Shell:          "/bin/bash",
		DefaultTimeout: 30 * time.Second,
		WorkingDir:     "",
		Environment:    nil,
	}
}

// Executor handles shell command execution
type Executor struct {
	config Config
	mu     sync.RWMutex
}

// New creates a new Executor with the given configuration
func New(cfg Config) *Executor {
	if cfg.Shell == "" {
		cfg.Shell = "/bin/bash"
	}
	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Second
	}
	return &Executor{
		config: cfg,
	}
}

// SetWorkingDir sets the working directory for command execution
func (e *Executor) SetWorkingDir(dir string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.WorkingDir = dir
}

// GetWorkingDir returns the current working directory
func (e *Executor) GetWorkingDir() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config.WorkingDir
}

// SetEnvironment sets the environment variables for command execution
func (e *Executor) SetEnvironment(env []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.Environment = env
}

// AddEnvironment adds environment variables to the existing ones
func (e *Executor) AddEnvironment(env ...string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config.Environment = append(e.config.Environment, env...)
}

// Execute runs a command and returns the complete result
func (e *Executor) Execute(ctx context.Context, command string) (*Result, error) {
	if err := validateCommand(command); err != nil {
		return nil, err
	}

	start := time.Now()

	e.mu.RLock()
	shell := e.config.Shell
	workingDir := e.config.WorkingDir
	environment := e.config.Environment
	e.mu.RUnlock()

	cmd := exec.CommandContext(ctx, shell, "-c", command)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	if len(environment) > 0 {
		cmd.Env = environment
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	executionTime := time.Since(start)

	result := &Result{
		Output:        stdout.String(),
		Error:         stderr.String(),
		ExecutionTime: executionTime,
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result, ErrCommandTimeout
		}
		if ctx.Err() == context.Canceled {
			return result, ErrCommandKilled
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}

		if errors.Is(err, exec.ErrNotFound) {
			return result, ErrCommandNotFound
		}

		return result, fmt.Errorf("command execution failed: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}

// ExecuteStream runs a command and streams the output
func (e *Executor) ExecuteStream(ctx context.Context, command string) (<-chan Output, error) {
	if err := validateCommand(command); err != nil {
		return nil, err
	}

	e.mu.RLock()
	shell := e.config.Shell
	workingDir := e.config.WorkingDir
	environment := e.config.Environment
	e.mu.RUnlock()

	cmd := exec.CommandContext(ctx, shell, "-c", command)
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	if len(environment) > 0 {
		cmd.Env = environment
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	outputCh := make(chan Output, 100)

	go func() {
		defer close(outputCh)

		var wg sync.WaitGroup
		wg.Add(2)

		// Read stdout
		go func() {
			defer wg.Done()
			readOutput(ctx, stdout, Stdout, outputCh)
		}()

		// Read stderr
		go func() {
			defer wg.Done()
			readOutput(ctx, stderr, Stderr, outputCh)
		}()

		wg.Wait()

		// Wait for command to complete
		exitCode := 0
		if err := cmd.Wait(); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			}
		}

		// Send completion signal
		select {
		case outputCh <- Output{IsComplete: true, ExitCode: exitCode}:
		case <-ctx.Done():
		}
	}()

	return outputCh, nil
}

// readOutput reads from a reader and sends output to the channel
func readOutput(ctx context.Context, reader io.Reader, outputType OutputType, ch chan<- Output) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line size

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			data := append(scanner.Bytes(), '\n')
			ch <- Output{
				Type: outputType,
				Data: data,
			}
		}
	}
}

// validateCommand checks if a command is valid
func validateCommand(command string) error {
	command = strings.TrimSpace(command)
	if command == "" {
		return ErrEmptyCommand
	}
	return nil
}

// IsDangerousCommand checks if a command might be dangerous
// This is a simple check and can be extended based on requirements
func IsDangerousCommand(command string) bool {
	dangerous := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=/dev/zero",
		":(){ :|:& };:",
		"> /dev/sda",
		"chmod -R 777 /",
	}

	cmdLower := strings.ToLower(command)
	for _, d := range dangerous {
		if strings.Contains(cmdLower, strings.ToLower(d)) {
			return true
		}
	}
	return false
}
