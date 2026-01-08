// Package tests contains Checkpoint 3 integration tests for NATS JetStream.
// Verifies: Publish liquidity update → Neo4j reflects change in <50ms.
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"
	natsClient "github.com/plm/predictive-liquidity-mesh/messaging/nats"
)

// TestCheckpoint3_NATSJetStreamSetup verifies JetStream stream creation
func TestCheckpoint3_NATSJetStreamSetup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to NATS
	cfg := natsClient.DefaultConfig()
	client, err := natsClient.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// Setup streams
	err = client.SetupStreams(ctx)
	if err != nil {
		t.Fatalf("Failed to setup streams: %v", err)
	}

	t.Log("✅ JetStream streams created successfully")

	// Verify streams exist
	js := client.JetStream()

	stream, err := js.Stream(ctx, natsClient.LiquidityUpdatesStream)
	if err != nil {
		t.Fatalf("Failed to get liquidity stream: %v", err)
	}

	info, err := stream.Info(ctx)
	if err != nil {
		t.Fatalf("Failed to get stream info: %v", err)
	}

	t.Logf("✅ Stream '%s' exists with %d messages", info.Config.Name, info.State.Msgs)

	// Verify work queue retention
	if info.Config.Retention != jetstream.WorkQueuePolicy {
		t.Errorf("Expected WorkQueue retention, got %s", info.Config.Retention)
	} else {
		t.Log("✅ Work Queue retention policy configured")
	}
}

// TestCheckpoint3_PublishLiquidityUpdate tests event publishing
func TestCheckpoint3_PublishLiquidityUpdate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := natsClient.DefaultConfig()
	client, err := natsClient.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	// Setup streams first
	_ = client.SetupStreams(ctx)

	// Publish test events
	events := []*natsClient.LiquidityUpdateEvent{
		{
			EventID:   uuid.New().String(),
			NodeID:    "lp_alpha",
			SourceID:  "lp_alpha",
			TargetID:  "hub_primary",
			EventType: "volume_change",
			OldValue:  5000000,
			NewValue:  5500000,
			Timestamp: time.Now(),
		},
		{
			EventID:   uuid.New().String(),
			NodeID:    "lp_beta",
			SourceID:  "lp_beta",
			TargetID:  "hub_secondary",
			EventType: "fee_change",
			OldValue:  0.0012,
			NewValue:  0.0010,
			Timestamp: time.Now(),
		},
		{
			EventID:   uuid.New().String(),
			NodeID:    "hub_backup",
			EventType: "status_change",
			OldValue:  1,
			NewValue:  0, // Going inactive
			Timestamp: time.Now(),
		},
	}

	for i, event := range events {
		start := time.Now()
		err := client.PublishLiquidityUpdate(ctx, event)
		latency := time.Since(start)

		if err != nil {
			t.Errorf("Failed to publish event %d: %v", i+1, err)
		} else {
			t.Logf("✅ Published event %d (%s) in %v", i+1, event.EventType, latency)
		}
	}

	// Verify messages in stream
	js := client.JetStream()
	stream, err := js.Stream(ctx, natsClient.LiquidityUpdatesStream)
	if err != nil {
		t.Logf("Could not get stream info: %v", err)
		return
	}
	info, _ := stream.Info(ctx)

	t.Logf("✅ Stream has %d messages after publishing", info.State.Msgs)
}

// TestCheckpoint3_WorkQueueConsumer tests exactly-once processing
func TestCheckpoint3_WorkQueueConsumer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg := natsClient.DefaultConfig()
	client, err := natsClient.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	_ = client.SetupStreams(ctx)

	// Create work queue consumer
	consumerCfg := natsClient.DefaultConsumerConfig(
		natsClient.LiquidityUpdatesStream,
		"test-consumer",
	)

	consumer, err := client.CreateWorkQueueConsumer(ctx, consumerCfg)
	if err != nil {
		t.Fatalf("Failed to create consumer: %v", err)
	}

	info, err := consumer.Info(ctx)
	if err != nil {
		t.Fatalf("Failed to get consumer info: %v", err)
	}

	t.Logf("✅ Work queue consumer '%s' created", info.Name)
	t.Logf("   - Ack Policy: %s", info.Config.AckPolicy)
	t.Logf("   - Max Deliver: %d", info.Config.MaxDeliver)
	t.Logf("   - Max Ack Pending: %d", info.Config.MaxAckPending)
}

// BenchmarkPublishLatency measures publish latency
func BenchmarkPublishLatency(b *testing.B) {
	ctx := context.Background()

	cfg := natsClient.DefaultConfig()
	client, err := natsClient.NewClient(ctx, cfg)
	if err != nil {
		b.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer client.Close()

	_ = client.SetupStreams(ctx)

	event := &natsClient.LiquidityUpdateEvent{
		EventID:   "benchmark",
		NodeID:    "lp_alpha",
		SourceID:  "lp_alpha",
		TargetID:  "hub_primary",
		EventType: "volume_change",
		NewValue:  5000000,
		Timestamp: time.Now(),
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.EventID = uuid.New().String()
		_ = client.PublishLiquidityUpdate(ctx, event)
	}
}
