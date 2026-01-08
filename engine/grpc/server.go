// Package grpc provides mTLS-secured gRPC server for node-to-node settlement.
package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
)

// ServerConfig holds gRPC server configuration
type ServerConfig struct {
	// Bind address (e.g., ":50051")
	Address string

	// mTLS configuration
	CertFile   string
	KeyFile    string
	CACertFile string

	// Performance tuning
	MaxConcurrentStreams uint32
	MaxRecvMsgSize       int
	MaxSendMsgSize       int

	// Keepalive
	KeepaliveTime    time.Duration
	KeepaliveTimeout time.Duration
}

// DefaultServerConfig returns production-ready defaults
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Address:              ":50051",
		MaxConcurrentStreams: 1000,
		MaxRecvMsgSize:       4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:       4 * 1024 * 1024, // 4MB
		KeepaliveTime:        30 * time.Second,
		KeepaliveTimeout:     10 * time.Second,
	}
}

// Server wraps gRPC server with mTLS support
type Server struct {
	cfg        *ServerConfig
	grpcServer *grpc.Server
	listener   net.Listener
	mu         sync.Mutex
	running    bool
}

// NewServer creates a new gRPC server with optional mTLS
func NewServer(cfg *ServerConfig) (*Server, error) {
	if cfg == nil {
		cfg = DefaultServerConfig()
	}

	var opts []grpc.ServerOption

	// mTLS configuration if certificates are provided
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		tlsConfig, err := loadTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS config: %w", err)
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsConfig)))
	}

	// Performance options
	opts = append(opts,
		grpc.MaxConcurrentStreams(cfg.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.KeepaliveTime,
			Timeout: cfg.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	return &Server{
		cfg:        cfg,
		grpcServer: grpc.NewServer(opts...),
	}, nil
}

// loadTLSConfig loads mTLS configuration
func loadTLSConfig(cfg *ServerConfig) (*tls.Config, error) {
	// Load server certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	// Load CA certificate for client verification (mTLS)
	if cfg.CACertFile != "" {
		caCert, err := os.ReadFile(cfg.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}

		tlsConfig.ClientCAs = certPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// GRPCServer returns the underlying gRPC server for service registration
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

// Start starts the gRPC server
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	listener, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	s.running = true
	s.mu.Unlock()

	return s.grpcServer.Serve(listener)
}

// Stop gracefully stops the server
func (s *Server) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.grpcServer.GracefulStop()
	s.running = false
}

// StopNow immediately stops the server
func (s *Server) StopNow() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.grpcServer.Stop()
	s.running = false
}

// ClientConfig holds gRPC client configuration
type ClientConfig struct {
	// Target address
	Address string

	// mTLS configuration
	CertFile   string
	KeyFile    string
	CACertFile string

	// Timeouts
	DialTimeout time.Duration
	CallTimeout time.Duration

	// Retry configuration
	MaxRetries int
}

// DefaultClientConfig returns sensible client defaults
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		DialTimeout: 10 * time.Second,
		CallTimeout: 30 * time.Second,
		MaxRetries:  3,
	}
}

// NewClientConn creates a new gRPC client connection with optional mTLS
func NewClientConn(ctx context.Context, cfg *ClientConfig) (*grpc.ClientConn, error) {
	if cfg == nil {
		cfg = DefaultClientConfig()
	}

	var opts []grpc.DialOption

	// mTLS configuration
	if cfg.CertFile != "" && cfg.KeyFile != "" && cfg.CACertFile != "" {
		tlsConfig, err := loadClientTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to load client TLS config: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	// Connection options
	opts = append(opts,
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	dialCtx, cancel := context.WithTimeout(ctx, cfg.DialTimeout)
	defer cancel()

	return grpc.DialContext(dialCtx, cfg.Address, opts...)
}

// loadClientTLSConfig loads mTLS configuration for client
func loadClientTLSConfig(cfg *ClientConfig) (*tls.Config, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(cfg.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// SettlementServiceServer interface for the settlement service
type SettlementServiceServer interface {
	Settle(ctx context.Context, req *SettleRequest) (*SettleResponse, error)
	StreamSettle(stream SettlementStream) error
	GetNodeStatus(ctx context.Context, req *NodeStatusRequest) (*NodeStatusResponse, error)
	Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error)
}

// SettlementStream interface for bidirectional streaming
type SettlementStream interface {
	Send(*SettleResponse) error
	Recv() (*SettleRequest, error)
	Context() context.Context
}

// Request/Response types (matching proto definitions)

type SettleRequest struct {
	RequestID     string
	SourceID      string
	TargetID      string
	DestinationID string
	Amount        int64
	Path          []string
	HopIndex      int32
	Signature     []byte
	Timestamp     int64
	Priority      int32
	Metadata      map[string]string
}

type SettleResponse struct {
	RequestID     string
	Status        SettlementStatus
	LedgerEntryID string
	ErrorCode     ErrorCode
	ErrorMessage  string
	ActualPath    []string
	TotalFeeBps   int64
	LatencyMs     int64
	CompletedAt   int64
}

type SettlementStatus int32

const (
	SettlementStatusUnspecified SettlementStatus = 0
	SettlementStatusPending     SettlementStatus = 1
	SettlementStatusProcessing  SettlementStatus = 2
	SettlementStatusCompleted   SettlementStatus = 3
	SettlementStatusFailed      SettlementStatus = 4
	SettlementStatusRerouted    SettlementStatus = 5
)

type ErrorCode int32

const (
	ErrorCodeUnspecified           ErrorCode = 0
	ErrorCodeInsufficientLiquidity ErrorCode = 1
	ErrorCodeNodeUnavailable       ErrorCode = 2
	ErrorCodeCircuitOpen           ErrorCode = 3
	ErrorCodeRateLimited           ErrorCode = 4
	ErrorCodeSignatureInvalid      ErrorCode = 5
	ErrorCodePathNotFound          ErrorCode = 6
	ErrorCodeTimeout               ErrorCode = 7
	ErrorCodeInternal              ErrorCode = 8
)

type NodeStatusRequest struct {
	NodeID string
}

type NodeStatusResponse struct {
	NodeID             string
	IsActive           bool
	CircuitState       CircuitState
	CurrentLoad        int64
	AvailableLiquidity int64
	PendingSettlements int64
	Timestamp          int64
}

type CircuitState int32

const (
	CircuitStateUnspecified CircuitState = 0
	CircuitStateClosed      CircuitState = 1
	CircuitStateOpen        CircuitState = 2
	CircuitStateHalfOpen    CircuitState = 3
)

type HeartbeatRequest struct {
	NodeID    string
	Timestamp int64
}

type HeartbeatResponse struct {
	NodeID    string
	Healthy   bool
	Timestamp int64
	Version   string
}

// Placeholder for io import usage
var _ = io.EOF
