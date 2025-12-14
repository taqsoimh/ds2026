package client

import (
	"context"
	"fmt"
	"io"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "remote-shell-rpc/proto"

	"remote-shell-rpc/pkg/logger"
)

// Config holds client configuration
type Config struct {
	Host    string        `yaml:"host"`
	Port    int           `yaml:"port"`
	Timeout time.Duration `yaml:"timeout"`
}

// DefaultConfig returns the default client configuration
func DefaultConfig() Config {
	return Config{
		Host:    "localhost",
		Port:    50051,
		Timeout: 10 * time.Second,
	}
}

// Client represents a gRPC shell client
type Client struct {
	config    Config
	conn      *grpc.ClientConn
	client    pb.ShellServiceClient
	sessionID string
	logger    *logger.Logger
}

// New creates a new Client with the given configuration
func New(cfg Config, log *logger.Logger) *Client {
	if log == nil {
		log = logger.Default()
	}
	return &Client{
		config: cfg,
		logger: log.WithComponent("client"),
	}
}

// Connect establishes a connection to the server
func (c *Client) Connect(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	c.logger.Info("Connecting to server", "address", address)

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}

	c.conn = conn
	c.client = pb.NewShellServiceClient(conn)

	c.logger.Info("Connected to server", "address", address)
	return nil
}

// Disconnect closes the connection to the server
func (c *Client) Disconnect() error {
	if c.sessionID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := c.client.CloseSession(ctx, &pb.CloseSessionRequest{
			SessionId: c.sessionID,
		})
		if err != nil {
			c.logger.Warn("Failed to close session", "error", err.Error())
		}
		c.sessionID = ""
	}

	if c.conn != nil {
		c.logger.Info("Disconnecting from server")
		return c.conn.Close()
	}
	return nil
}

// CreateSession creates a new shell session
func (c *Client) CreateSession(ctx context.Context, clientID string) error {
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	resp, err := c.client.CreateSession(ctx, &pb.CreateSessionRequest{
		ClientId: clientID,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	c.sessionID = resp.SessionId
	c.logger.Info("Session created",
		"session_id", c.sessionID,
		"working_dir", resp.WorkingDirectory,
	)

	return nil
}

// GetSessionID returns the current session ID
func (c *Client) GetSessionID() string {
	return c.sessionID
}

// ExecuteCommand executes a command and returns the result
func (c *Client) ExecuteCommand(ctx context.Context, command string, timeout int) (*pb.CommandResponse, error) {
	if c.sessionID == "" {
		return nil, fmt.Errorf("no active session")
	}

	resp, err := c.client.ExecuteCommand(ctx, &pb.CommandRequest{
		SessionId:      c.sessionID,
		Command:        command,
		TimeoutSeconds: int32(timeout),
	})
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	return resp, nil
}

// ExecuteCommandStream executes a command and streams the output
func (c *Client) ExecuteCommandStream(ctx context.Context, command string, timeout int, outputHandler func(output *pb.CommandOutput)) error {
	if c.sessionID == "" {
		return fmt.Errorf("no active session")
	}

	stream, err := c.client.ExecuteCommandStream(ctx, &pb.CommandRequest{
		SessionId:      c.sessionID,
		Command:        command,
		TimeoutSeconds: int32(timeout),
	})
	if err != nil {
		return fmt.Errorf("failed to start command stream: %w", err)
	}

	for {
		output, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream error: %w", err)
		}

		if outputHandler != nil {
			outputHandler(output)
		}
	}

	return nil
}

// IsConnected returns true if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil
}

// HasSession returns true if the client has an active session
func (c *Client) HasSession() bool {
	return c.sessionID != ""
}
