// Package consumers provides NATS JetStream consumers for the Predictive Liquidity Mesh.
// Implements graph sync for eventual consistency between Postgres and Neo4j.
package consumers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	natsClient "github.com/plm/predictive-liquidity-mesh/messaging/nats"
	"github.com/plm/predictive-liquidity-mesh/storage/neo4j"
)

// GraphSyncConsumer synchronizes liquidity updates to Neo4j
type GraphSyncConsumer struct {
	nats      *natsClient.Client
	neo4j     *neo4j.Client
	consumer  jetstream.Consumer
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	workers   int
	batchSize int
}

// GraphSyncConfig configures the graph sync consumer
type GraphSyncConfig struct {
	Workers      int           // Number of parallel workers
	BatchSize    int           // Messages per batch
	PollInterval time.Duration // How often to poll for messages
}

// DefaultGraphSyncConfig returns sensible defaults
func DefaultGraphSyncConfig() *GraphSyncConfig {
	return &GraphSyncConfig{
		Workers:      5,
		BatchSize:    100,
		PollInterval: 100 * time.Millisecond,
	}
}

// NewGraphSyncConsumer creates a new graph sync consumer
func NewGraphSyncConsumer(
	ctx context.Context,
	nats *natsClient.Client,
	neo4j *neo4j.Client,
	cfg *GraphSyncConfig,
) (*GraphSyncConsumer, error) {
	if cfg == nil {
		cfg = DefaultGraphSyncConfig()
	}

	// Create work queue consumer for liquidity updates
	consumerCfg := natsClient.DefaultConsumerConfig(
		natsClient.LiquidityUpdatesStream,
		"graph-sync-consumer",
	)
	consumerCfg.FilterSubject = "liquidity.>"
	consumerCfg.MaxAckPending = cfg.BatchSize * cfg.Workers

	consumer, err := nats.CreateWorkQueueConsumer(ctx, consumerCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	consumerCtx, cancel := context.WithCancel(ctx)

	return &GraphSyncConsumer{
		nats:      nats,
		neo4j:     neo4j,
		consumer:  consumer,
		ctx:       consumerCtx,
		cancel:    cancel,
		workers:   cfg.Workers,
		batchSize: cfg.BatchSize,
	}, nil
}

// Start begins consuming messages and syncing to Neo4j
func (c *GraphSyncConsumer) Start() error {
	log.Printf("Starting GraphSyncConsumer with %d workers", c.workers)

	for i := 0; i < c.workers; i++ {
		c.wg.Add(1)
		go c.worker(i)
	}

	return nil
}

// Stop gracefully stops the consumer
func (c *GraphSyncConsumer) Stop() {
	log.Println("Stopping GraphSyncConsumer...")
	c.cancel()
	c.wg.Wait()
	log.Println("GraphSyncConsumer stopped")
}

// worker processes messages in a loop
func (c *GraphSyncConsumer) worker(id int) {
	defer c.wg.Done()

	log.Printf("GraphSync worker %d started", id)

	for {
		select {
		case <-c.ctx.Done():
			log.Printf("GraphSync worker %d stopping", id)
			return
		default:
			// Fetch messages with timeout
			msgs, err := c.consumer.Fetch(c.batchSize, jetstream.FetchMaxWait(time.Second))
			if err != nil {
				if c.ctx.Err() != nil {
					return
				}
				// Timeout is expected when no messages
				continue
			}

			for msg := range msgs.Messages() {
				if err := c.processMessage(msg); err != nil {
					log.Printf("Worker %d: Failed to process message: %v", id, err)
					// NAK for redelivery
					msg.Nak()
				} else {
					// ACK on success
					msg.Ack()
				}
			}

			if msgs.Error() != nil && c.ctx.Err() == nil {
				log.Printf("Worker %d: Fetch error: %v", id, msgs.Error())
			}
		}
	}
}

// processMessage processes a single liquidity update message
func (c *GraphSyncConsumer) processMessage(msg jetstream.Msg) error {
	var event natsClient.LiquidityUpdateEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	start := time.Now()

	// Apply update to Neo4j based on event type
	switch event.EventType {
	case "volume_change":
		err := c.updateLiquidityVolume(&event)
		if err != nil {
			return err
		}

	case "fee_change":
		err := c.updateBaseFee(&event)
		if err != nil {
			return err
		}

	case "status_change":
		err := c.updateNodeStatus(&event)
		if err != nil {
			return err
		}

	case "latency_change":
		err := c.updateLatency(&event)
		if err != nil {
			return err
		}

	default:
		log.Printf("Unknown event type: %s", event.EventType)
	}

	latency := time.Since(start)
	log.Printf("Processed %s for node %s in %v", event.EventType, event.NodeID, latency)

	return nil
}

// updateLiquidityVolume updates an edge's liquidity volume
func (c *GraphSyncConsumer) updateLiquidityVolume(event *natsClient.LiquidityUpdateEvent) error {
	if event.SourceID == "" || event.TargetID == "" {
		return fmt.Errorf("missing source or target ID for volume update")
	}

	return c.neo4j.UpdateEdge(c.ctx, event.SourceID, event.TargetID, map[string]interface{}{
		"liquidity_volume": int64(event.NewValue),
		"last_updated":     event.Timestamp,
	})
}

// updateBaseFee updates an edge's base fee
func (c *GraphSyncConsumer) updateBaseFee(event *natsClient.LiquidityUpdateEvent) error {
	if event.SourceID == "" || event.TargetID == "" {
		return fmt.Errorf("missing source or target ID for fee update")
	}

	return c.neo4j.UpdateEdge(c.ctx, event.SourceID, event.TargetID, map[string]interface{}{
		"base_fee":     event.NewValue,
		"last_updated": event.Timestamp,
	})
}

// updateNodeStatus updates a node's active status
func (c *GraphSyncConsumer) updateNodeStatus(event *natsClient.LiquidityUpdateEvent) error {
	isActive := event.NewValue > 0
	return c.neo4j.SetNodeActive(c.ctx, event.NodeID, isActive)
}

// updateLatency updates an edge's latency
func (c *GraphSyncConsumer) updateLatency(event *natsClient.LiquidityUpdateEvent) error {
	if event.SourceID == "" || event.TargetID == "" {
		return fmt.Errorf("missing source or target ID for latency update")
	}

	return c.neo4j.UpdateEdge(c.ctx, event.SourceID, event.TargetID, map[string]interface{}{
		"latency":      int64(event.NewValue),
		"last_updated": event.Timestamp,
	})
}

// Stats returns consumer statistics
type Stats struct {
	Processed   int64
	Failed      int64
	AvgLatency  time.Duration
	LastMessage time.Time
}

// GetStats returns current consumer statistics
func (c *GraphSyncConsumer) GetStats() (*Stats, error) {
	info, err := c.consumer.Info(c.ctx)
	if err != nil {
		return nil, err
	}

	return &Stats{
		Processed: int64(info.NumAckPending),
	}, nil
}
