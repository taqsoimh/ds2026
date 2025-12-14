// Package main is the entry point for the remote shell client.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
	"gopkg.in/yaml.v3"
	"remote-shell-rpc/internal/client"
	"remote-shell-rpc/pkg/logger"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	host := flag.String("host", "localhost", "Server host")
	port := flag.Int("port", 50051, "Server port")
	clientID := flag.String("client-id", "", "Client ID (auto-generated if empty)")
	logLevel := flag.String("log-level", "warn", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Create logger
	logCfg := logger.Config{
		Level:  logger.Level(*logLevel),
		Format: "text",
		Output: os.Stderr,
	}
	log := logger.New(logCfg)

	// Load configuration
	cfg := client.DefaultConfig()

	if *configPath != "" {
		loadedCfg, err := loadConfig(*configPath)
		if err != nil {
			log.Error("Failed to load config", "error", err.Error())
			os.Exit(1)
		}
		cfg = loadedCfg
	}

	// Override with command line flags
	if *host != "localhost" {
		cfg.Host = *host
	}
	if *port != 50051 {
		cfg.Port = *port
	}

	// Generate client ID if not provided
	cID := *clientID
	if cID == "" {
		cID = fmt.Sprintf("client-%d", time.Now().UnixNano())
	}

	// Create client
	c := client.New(cfg, log)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt signal, disconnecting...")
		cancel()
	}()

	// Connect to server
	fmt.Printf("Connecting to %s:%d...\n", cfg.Host, cfg.Port)
	if err := c.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer c.Disconnect()

	// Create session
	if err := c.CreateSession(ctx, cID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create session: %v\n", err)
		os.Exit(1)
	}

	// Create and run interactive shell
	shellCfg := client.DefaultShellConfig()
	shell := client.NewShell(c, shellCfg)

	if err := shell.Run(ctx); err != nil {
		if ctx.Err() == nil {
			fmt.Fprintf(os.Stderr, "Shell error: %v\n", err)
			os.Exit(1)
		}
	}
}

// loadConfig loads configuration from a YAML file
func loadConfig(path string) (client.Config, error) {
	cfg := client.DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	var fileCfg struct {
		Server struct {
			Host    string `yaml:"host"`
			Port    int    `yaml:"port"`
			Timeout string `yaml:"timeout"`
		} `yaml:"server"`
		Shell struct {
			Prompt      string `yaml:"prompt"`
			HistorySize int    `yaml:"history_size"`
		} `yaml:"shell"`
	}

	if err := yaml.Unmarshal(data, &fileCfg); err != nil {
		return cfg, err
	}

	if fileCfg.Server.Host != "" {
		cfg.Host = fileCfg.Server.Host
	}
	if fileCfg.Server.Port != 0 {
		cfg.Port = fileCfg.Server.Port
	}
	if fileCfg.Server.Timeout != "" {
		if timeout, err := time.ParseDuration(fileCfg.Server.Timeout); err == nil {
			cfg.Timeout = timeout
		}
	}

	return cfg, nil
}
