// Package nats provides NATS JetStream integration for the Predictive Liquidity Mesh.
// Implements async work queues for exactly-once event processing.
package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamName constants
const (
	LiquidityUpdatesStream  = "LIQUIDITY_UPDATES"
	LiquidityUpdatesSubject = "liquidity.updates"
	SettlementEventsStream  = "SETTLEMENT_EVENTS"
	SettlementEventsSubject = "settlement.events"
)

// Config holds NATS connection configuration
type Config struct {
	// Connection URLs (comma-separated for cluster)
	URLs string

	// Authentication
	Token    string
	User     string
	Password string

	// TLS
	CertFile string
	KeyFile  string
	CAFile   string

	// Reconnection
	MaxReconnects   int
	ReconnectWait   time.Duration
	ReconnectJitter time.Duration
}

// DefaultConfig returns development defaults
func DefaultConfig() *Config {
	return &Config{
		URLs:            "nats://localhost:4222",
		MaxReconnects:   -1, // Unlimited
		ReconnectWait:   2 * time.Second,
		ReconnectJitter: 500 * time.Millisecond,
	}
}

// Client wraps NATS connection with JetStream support
type Client struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	mu  sync.RWMutex
	cfg *Config
}

// NewClient creates a new NATS client with JetStream
func NewClient(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	opts := []nats.Option{
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.ReconnectJitter(cfg.ReconnectJitter, cfg.ReconnectJitter*2),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
	}

	// Authentication
	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	} else if cfg.User != "" && cfg.Password != "" {
		opts = append(opts, nats.UserInfo(cfg.User, cfg.Password))
	}

	// Connect
	nc, err := nats.Connect(cfg.URLs, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &Client{
		nc:  nc,
		js:  js,
		cfg: cfg,
	}, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.nc != nil {
		c.nc.Drain()
	}
}

// JetStream returns the JetStream context
func (c *Client) JetStream() jetstream.JetStream {
	return c.js
}

// Connection returns the underlying NATS connection
func (c *Client) Connection() *nats.Conn {
	return c.nc
}

// SetupStreams initializes all required JetStream streams
func (c *Client) SetupStreams(ctx context.Context) error {
	// Liquidity Updates Stream - Work Queue pattern
	_, err := c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        LiquidityUpdatesStream,
		Description: "Liquidity update events for mesh synchronization",
		Subjects:    []string{"liquidity.>"}, // Use only wildcard
		Retention:   jetstream.WorkQueuePolicy, // Exactly-once processing
		MaxAge:      24 * time.Hour,
		MaxBytes:    1024 * 1024 * 1024, // 1GB
		MaxMsgs:     1000000,
		Discard:     jetstream.DiscardOld,
		Replicas:    1, // Increase for HA
		Storage:     jetstream.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create liquidity stream: %w", err)
	}

	// Settlement Events Stream
	_, err = c.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        SettlementEventsStream,
		Description: "Settlement transaction events",
		Subjects:    []string{"settlement.>"}, // Use only wildcard
		Retention:   jetstream.LimitsPolicy, // Keep for replay
		MaxAge:      7 * 24 * time.Hour,
		MaxBytes:    512 * 1024 * 1024, // 512MB (reduced for dev)
		MaxMsgs:     1000000,
		Discard:     jetstream.DiscardOld,
		Replicas:    1,
		Storage:     jetstream.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("failed to create settlement stream: %w", err)
	}

	return nil
}

// LiquidityUpdateEvent represents a liquidity change event
type LiquidityUpdateEvent struct {
	EventID   string    `json:"event_id"`
	NodeID    string    `json:"node_id"`
	EdgeID    string    `json:"edge_id,omitempty"`
	SourceID  string    `json:"source_id,omitempty"`
	TargetID  string    `json:"target_id,omitempty"`
	EventType string    `json:"event_type"` // "volume_change", "fee_change", "status_change"
	OldValue  float64   `json:"old_value,omitempty"`
	NewValue  float64   `json:"new_value"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PublishLiquidityUpdate publishes a liquidity update event
func (c *Client) PublishLiquidityUpdate(ctx context.Context, event *LiquidityUpdateEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := fmt.Sprintf("liquidity.updates.%s", event.NodeID)
	_, err = c.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// SettlementEvent represents a settlement transaction event
type SettlementEvent struct {
	EventID      string    `json:"event_id"`
	RequestID    string    `json:"request_id"`
	EventType    string    `json:"event_type"` // "initiated", "hop_complete", "completed", "failed", "rerouted"
	SourceID     string    `json:"source_id"`
	TargetID     string    `json:"target_id"`
	Amount       int64     `json:"amount"`
	Path         []string  `json:"path"`
	CurrentHop   int       `json:"current_hop"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// PublishSettlementEvent publishes a settlement event
func (c *Client) PublishSettlementEvent(ctx context.Context, event *SettlementEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	subject := fmt.Sprintf("settlement.events.%s", event.EventType)
	_, err = c.js.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// ConsumerConfig configures a work queue consumer
type ConsumerConfig struct {
	StreamName    string
	ConsumerName  string
	FilterSubject string
	MaxDeliver    int
	AckWait       time.Duration
	MaxAckPending int
}

// DefaultConsumerConfig returns sensible consumer defaults
func DefaultConsumerConfig(stream, name string) *ConsumerConfig {
	return &ConsumerConfig{
		StreamName:    stream,
		ConsumerName:  name,
		MaxDeliver:    3,
		AckWait:       30 * time.Second,
		MaxAckPending: 1000,
	}
}

// CreateWorkQueueConsumer creates a durable work queue consumer
func (c *Client) CreateWorkQueueConsumer(ctx context.Context, cfg *ConsumerConfig) (jetstream.Consumer, error) {
	consumerCfg := jetstream.ConsumerConfig{
		Durable:       cfg.ConsumerName,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    cfg.MaxDeliver,
		AckWait:       cfg.AckWait,
		MaxAckPending: cfg.MaxAckPending,
	}

	if cfg.FilterSubject != "" {
		consumerCfg.FilterSubject = cfg.FilterSubject
	}

	consumer, err := c.js.CreateOrUpdateConsumer(ctx, cfg.StreamName, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	return consumer, nil
}
