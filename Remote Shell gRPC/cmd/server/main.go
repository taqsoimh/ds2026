// Package main is the entry point for the remote shell server.
package main

import (
	"flag"
	"log"
	"os"
	"time"
	"gopkg.in/yaml.v3"
	"remote-shell-rpc/internal/server"
	"remote-shell-rpc/pkg/logger"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	host := flag.String("host", "0.0.0.0", "Server host")
	port := flag.Int("port", 50051, "Server port")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Create logger
	logCfg := logger.Config{
		Level:  logger.Level(*logLevel),
		Format: "text",
		Output: os.Stdout,
	}
	log := logger.New(logCfg)

	// Load configuration
	cfg := server.DefaultConfig()

	if *configPath != "" {
		loadedCfg, err := loadConfig(*configPath)
		if err != nil {
			log.Error("Failed to load config", "error", err.Error())
			os.Exit(1)
		}
		cfg = loadedCfg
	}

	// Override with command line flags
	if *host != "0.0.0.0" {
		cfg.Host = *host
	}
	if *port != 50051 {
		cfg.Port = *port
	}

	// Create and start server
	srv := server.New(cfg, log)

	log.Info("Starting Remote Shell RPC Server",
		"host", cfg.Host,
		"port", cfg.Port,
		"max_connections", cfg.MaxConnections,
	)

	if err := srv.Start(); err != nil {
		log.Error("Server failed", "error", err.Error())
		os.Exit(1)
	}
}

// loadConfig loads configuration from a YAML file
func loadConfig(path string) (server.Config, error) {
	cfg := server.DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	var fileCfg struct {
		Server struct {
			Host           string `yaml:"host"`
			Port           int    `yaml:"port"`
			MaxConnections int    `yaml:"max_connections"`
		} `yaml:"server"`
		Executor struct {
			Timeout string `yaml:"timeout"`
			Shell   string `yaml:"shell"`
		} `yaml:"executor"`
		Logging struct {
			Level  string `yaml:"level"`
			Format string `yaml:"format"`
		} `yaml:"logging"`
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
	if fileCfg.Server.MaxConnections != 0 {
		cfg.MaxConnections = fileCfg.Server.MaxConnections
	}
	if fileCfg.Executor.Timeout != "" {
		if timeout, err := time.ParseDuration(fileCfg.Executor.Timeout); err == nil {
			cfg.CommandTimeout = timeout
		}
	}
	if fileCfg.Executor.Shell != "" {
		cfg.Shell = fileCfg.Executor.Shell
	}

	return cfg, nil
}

func init() {
	// Suppress default log output
	log.SetOutput(os.Stderr)
}
