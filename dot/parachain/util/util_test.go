package util

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
)

func TestReputationAggregator_SendImmediately(t *testing.T) {
	// Mock channel
	overseerCh := make(chan NetworkBridgeTxMessage, 1)

	// Create a new aggregator with immediate send logic for changes < 0
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return rep.Change < 0
	})

	// Mock peer and reputation change
	peerID := peer.ID("peer1")
	repChange := UnifiedReputationChange{
		Change: -10,
		Reason: "Malicious behavior",
	}

	// Modify the aggregator
	aggregator.Modify(overseerCh, peerID, repChange)

	// Verify the message is sent immediately
	select {
	case msg := <-overseerCh:
		assert.Len(t, msg.ReportPeerMessageBatch, 1)
		assert.Equal(t, int32(-10), msg.ReportPeerMessageBatch[peerID])
	default:
		t.Error("Expected immediate message, but none was sent")
	}
}

func TestReputationAggregator_BatchSend(t *testing.T) {
	// Mock channel
	overseerCh := make(chan NetworkBridgeTxMessage, 1)

	// Create a new aggregator with no immediate send logic
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add multiple reputation changes
	peerID1 := peer.ID("peer1")
	peerID2 := peer.ID("peer2")
	aggregator.Modify(overseerCh, peerID1, UnifiedReputationChange{Change: 5, Reason: "Good behavior"})
	aggregator.Modify(overseerCh, peerID2, UnifiedReputationChange{Change: 10, Reason: "Great behavior"})

	// Verify no messages were sent yet
	select {
	case <-overseerCh:
		t.Error("Expected no message to be sent, but one was sent")
	default:
	}

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	// Verify the batch message
	select {
	case msg := <-overseerCh:
		assert.Len(t, msg.ReportPeerMessageBatch, 2)
		assert.Equal(t, int32(5), msg.ReportPeerMessageBatch[peerID1])
		assert.Equal(t, int32(10), msg.ReportPeerMessageBatch[peerID2])
	default:
		t.Error("Expected batch message, but none was sent")
	}
}

func TestReputationAggregator_ClearAfterSend(t *testing.T) {
	// Mock channel
	overseerCh := make(chan NetworkBridgeTxMessage, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add a reputation change
	peerID := peer.ID("peer1")
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Change: 10, Reason: "Good behavior"})

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	// Verify the batch message
	select {
	case <-overseerCh:
		// Expected message sent
	default:
		t.Error("Expected batch message, but none was sent")
	}

	// Verify the internal state is cleared
	assert.Empty(t, aggregator.byPeer)
}

func TestReputationAggregator_ConflictResolution(t *testing.T) {
	// Mock channel
	overseerCh := make(chan NetworkBridgeTxMessage, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false // Always accumulate
	})

	// Add multiple reputation changes for the same peer
	peerID := peer.ID("peer1")
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Change: 10, Reason: "Good behavior"})
	aggregator.Modify(overseerCh, peerID, UnifiedReputationChange{Change: -5, Reason: "Minor issue"})

	// Call Send to flush changes
	aggregator.Send(overseerCh)

	// Verify the accumulated result
	select {
	case msg := <-overseerCh:
		assert.Len(t, msg.ReportPeerMessageBatch, 1)
		assert.Equal(t, int32(5), msg.ReportPeerMessageBatch[peerID]) // 10 + (-5) = 5
	default:
		t.Error("Expected batch message, but none was sent")
	}
}

func TestReputationAggregator_NoActionWithoutChanges(t *testing.T) {
	// Mock channel
	overseerCh := make(chan NetworkBridgeTxMessage, 1)

	// Create a new aggregator
	aggregator := NewReputationAggregator(func(rep UnifiedReputationChange) bool {
		return false
	})

	// Call Send without any changes
	aggregator.Send(overseerCh)

	// Verify no messages were sent
	select {
	case <-overseerCh:
		t.Error("Expected no message, but one was sent")
	default:
		// Expected behavior
	}
}
