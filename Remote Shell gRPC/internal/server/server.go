package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	pb "remote-shell-rpc/proto"

	"remote-shell-rpc/pkg/executor"
	"remote-shell-rpc/pkg/logger"
	"remote-shell-rpc/pkg/session"
)

// Config holds server configuration
type Config struct {
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	MaxConnections int           `yaml:"max_connections"`
	CommandTimeout time.Duration `yaml:"command_timeout"`
	Shell          string        `yaml:"shell"`
}

// DefaultConfig returns the default server configuration
func DefaultConfig() Config {
	return Config{
		Host:           "0.0.0.0",
		Port:           50051,
		MaxConnections: 100,
		CommandTimeout: 30 * time.Second,
		Shell:          "/bin/bash",
	}
}

// Server represents the gRPC shell server
type Server struct {
	pb.UnimplementedShellServiceServer
	config         Config
	sessionManager *session.Manager
	logger         *logger.Logger
	grpcServer     *grpc.Server
}

// New creates a new Server with the given configuration
func New(cfg Config, log *logger.Logger) *Server {
	if log == nil {
		log = logger.Default()
	}

	sessionCfg := session.ManagerConfig{
		MaxSessions: cfg.MaxConnections,
	}

	return &Server{
		config:         cfg,
		sessionManager: session.NewManager(sessionCfg),
		logger:         log.WithComponent("server"),
	}
}

// Start starts the gRPC server
func (s *Server) Start() error {
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	// Create gRPC server with interceptors
	s.grpcServer = grpc.NewServer(
		grpc.UnaryInterceptor(s.unaryInterceptor),
		grpc.StreamInterceptor(s.streamInterceptor),
	)

	// Register the shell service
	pb.RegisterShellServiceServer(s.grpcServer, s)

	s.logger.Info("Server starting", "address", address)

	// Handle graceful shutdown
	go s.handleShutdown()

	// Start serving
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	if s.grpcServer != nil {
		s.logger.Info("Stopping server gracefully")
		s.grpcServer.GracefulStop()
	}
}

// handleShutdown handles OS signals for graceful shutdown
func (s *Server) handleShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	sig := <-sigCh
	s.logger.Info("Received shutdown signal", "signal", sig.String())
	s.Stop()
}

// unaryInterceptor is a gRPC unary interceptor for logging and recovery
func (s *Server) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Get client address
	clientAddr := "unknown"
	if p, ok := peer.FromContext(ctx); ok {
		clientAddr = p.Addr.String()
	}

	s.logger.Debug("Request received",
		"method", info.FullMethod,
		"client", clientAddr,
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic recovered", "method", info.FullMethod, "panic", r)
		}
	}()

	// Call the handler
	resp, err := handler(ctx, req)

	// Log completion
	duration := time.Since(start)
	if err != nil {
		s.logger.Warn("Request failed",
			"method", info.FullMethod,
			"duration", duration,
			"error", err.Error(),
		)
	} else {
		s.logger.Debug("Request completed",
			"method", info.FullMethod,
			"duration", duration,
		)
	}

	return resp, err
}

// streamInterceptor is a gRPC stream interceptor for logging and recovery
func (s *Server) streamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()

	// Get client address
	clientAddr := "unknown"
	if p, ok := peer.FromContext(ss.Context()); ok {
		clientAddr = p.Addr.String()
	}

	s.logger.Debug("Stream started",
		"method", info.FullMethod,
		"client", clientAddr,
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("Panic recovered in stream", "method", info.FullMethod, "panic", r)
		}
	}()

	err := handler(srv, ss)

	duration := time.Since(start)
	if err != nil {
		s.logger.Warn("Stream failed",
			"method", info.FullMethod,
			"duration", duration,
			"error", err.Error(),
		)
	} else {
		s.logger.Debug("Stream completed",
			"method", info.FullMethod,
			"duration", duration,
		)
	}

	return err
}

// CreateSession creates a new shell session for a client
func (s *Server) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	if req.ClientId == "" {
		return nil, status.Error(codes.InvalidArgument, "client_id is required")
	}

	sess, err := s.sessionManager.Create(req.ClientId)
	if err != nil {
		if err == session.ErrMaxSessions {
			return nil, status.Error(codes.ResourceExhausted, "maximum sessions reached")
		}
		return nil, status.Errorf(codes.Internal, "failed to create session: %v", err)
	}

	s.logger.Info("Session created",
		"session_id", sess.ID,
		"client_id", req.ClientId,
	)

	return &pb.CreateSessionResponse{
		SessionId:        sess.ID,
		WorkingDirectory: sess.WorkingDir,
	}, nil
}

// CloseSession terminates an existing shell session
func (s *Server) CloseSession(ctx context.Context, req *pb.CloseSessionRequest) (*pb.CloseSessionResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	err := s.sessionManager.Delete(req.SessionId)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to close session: %v", err)
	}

	s.logger.Info("Session closed", "session_id", req.SessionId)

	return &pb.CloseSessionResponse{
		Success: true,
		Message: "Session closed successfully",
	}, nil
}

// ExecuteCommand runs a command and returns the complete result
func (s *Server) ExecuteCommand(ctx context.Context, req *pb.CommandRequest) (*pb.CommandResponse, error) {
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	if req.Command == "" {
		return nil, status.Error(codes.InvalidArgument, "command is required")
	}

	// Get session
	sess, err := s.sessionManager.Get(req.SessionId)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	// Check for dangerous commands
	if executor.IsDangerousCommand(req.Command) {
		return nil, status.Error(codes.PermissionDenied, "dangerous command blocked")
	}

	// Handle special commands
	if handled, response := s.handleSpecialCommand(sess, req.Command); handled {
		return response, nil
	}

	// Set timeout
	timeout := s.config.CommandTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sess.UpdateActivity()

	s.logger.Debug("Executing command",
		"session_id", req.SessionId,
		"command", req.Command,
	)

	// Execute command
	result, err := sess.Executor.Execute(ctx, req.Command)
	if err != nil {
		if err == executor.ErrCommandTimeout {
			return nil, status.Error(codes.DeadlineExceeded, "command execution timeout")
		}
		if err == executor.ErrEmptyCommand {
			return nil, status.Error(codes.InvalidArgument, "empty command")
		}
		s.logger.Warn("Command execution failed",
			"session_id", req.SessionId,
			"command", req.Command,
			"error", err.Error(),
		)
	}

	return &pb.CommandResponse{
		Output:          result.Output,
		Error:           result.Error,
		ExitCode:        int32(result.ExitCode),
		ExecutionTimeMs: result.ExecutionTime.Milliseconds(),
	}, nil
}

// ExecuteCommandStream runs a command and streams the output
func (s *Server) ExecuteCommandStream(req *pb.CommandRequest, stream pb.ShellService_ExecuteCommandStreamServer) error {
	if req.SessionId == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}
	if req.Command == "" {
		return status.Error(codes.InvalidArgument, "command is required")
	}

	// Get session
	sess, err := s.sessionManager.Get(req.SessionId)
	if err != nil {
		if err == session.ErrSessionNotFound {
			return status.Error(codes.NotFound, "session not found")
		}
		return status.Errorf(codes.Internal, "failed to get session: %v", err)
	}

	// Check for dangerous commands
	if executor.IsDangerousCommand(req.Command) {
		return status.Error(codes.PermissionDenied, "dangerous command blocked")
	}

	// Handle special commands
	if handled, response := s.handleSpecialCommand(sess, req.Command); handled {
		// Send as stream output
		output := &pb.CommandOutput{
			Type:       pb.CommandOutput_STDOUT,
			Data:       []byte(response.Output),
			IsComplete: true,
			ExitCode:   response.ExitCode,
		}
		return stream.Send(output)
	}

	// Set timeout
	timeout := s.config.CommandTimeout
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(stream.Context(), timeout)
	defer cancel()

	sess.UpdateActivity()

	s.logger.Debug("Executing command (stream)",
		"session_id", req.SessionId,
		"command", req.Command,
	)

	// Execute command with streaming
	outputCh, err := sess.Executor.ExecuteStream(ctx, req.Command)
	if err != nil {
		if err == executor.ErrEmptyCommand {
			return status.Error(codes.InvalidArgument, "empty command")
		}
		return status.Errorf(codes.Internal, "failed to execute command: %v", err)
	}

	// Stream output to client
	for output := range outputCh {
		var outputType pb.CommandOutput_OutputType
		if output.Type == executor.Stderr {
			outputType = pb.CommandOutput_STDERR
		} else {
			outputType = pb.CommandOutput_STDOUT
		}

		msg := &pb.CommandOutput{
			Type:       outputType,
			Data:       output.Data,
			IsComplete: output.IsComplete,
			ExitCode:   int32(output.ExitCode),
		}

		if err := stream.Send(msg); err != nil {
			s.logger.Warn("Failed to send stream output",
				"session_id", req.SessionId,
				"error", err.Error(),
			)
			return err
		}
	}

	return nil
}

// handleSpecialCommand handles special built-in commands like cd
func (s *Server) handleSpecialCommand(sess *session.Session, command string) (bool, *pb.CommandResponse) {
	command = strings.TrimSpace(command)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false, nil
	}

	switch parts[0] {
	case "cd":
		return s.handleCdCommand(sess, parts)
	}

	return false, nil
}

// handleCdCommand handles the cd command
func (s *Server) handleCdCommand(sess *session.Session, parts []string) (bool, *pb.CommandResponse) {
	var targetDir string

	if len(parts) == 1 {
		// cd without argument goes to home
		home, err := os.UserHomeDir()
		if err != nil {
			return true, &pb.CommandResponse{
				Error:    "cannot determine home directory",
				ExitCode: 1,
			}
		}
		targetDir = home
	} else {
		targetDir = parts[1]
	}

	// Handle relative paths
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(sess.GetWorkingDir(), targetDir)
	}

	// Clean the path
	targetDir = filepath.Clean(targetDir)

	// Check if directory exists
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, &pb.CommandResponse{
				Error:    fmt.Sprintf("cd: %s: No such file or directory", parts[1]),
				ExitCode: 1,
			}
		}
		return true, &pb.CommandResponse{
			Error:    fmt.Sprintf("cd: %s: %v", parts[1], err),
			ExitCode: 1,
		}
	}

	if !info.IsDir() {
		return true, &pb.CommandResponse{
			Error:    fmt.Sprintf("cd: %s: Not a directory", parts[1]),
			ExitCode: 1,
		}
	}

	sess.SetWorkingDir(targetDir)

	return true, &pb.CommandResponse{
		Output:   "",
		ExitCode: 0,
	}
}

// GetSessionCount returns the number of active sessions
func (s *Server) GetSessionCount() int {
	return s.sessionManager.Count()
}
