package statementdistribution

import (
	"context"
	"fmt"
	"time"

	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statement-distribution"))

type StatementDistribution struct {
	SubSystemToOverseer chan<- parachainutil.NetworkBridgeTxMessage
}

type MuxedMessage interface {
	isMuxedMessage()
}

type overseerMessage struct {
	inner any
}

func (*overseerMessage) isMuxedMessage() {}

type responderMessage struct {
	inner any // should be replaced with AttestedCandidateRequest type
}

func (*responderMessage) isMuxedMessage() {}

// Run just receives the ctx and a channel from the overseer to subsystem
func (s *StatementDistribution) Run(ctx context.Context, overseerToSubSystem <-chan any) {
	// Inside the method Run, we spawn a goroutine to handle network incoming requests
	// TODO: https://github.com/ChainSafe/gossamer/issues/4285
	responderCh := make(chan any, 1)
	go s.taskResponder(responderCh)

	// Timer for reputation aggregator trigger
	reputationDelay := time.NewTicker(parachainutil.ReputationChangeInterval) // Adjust the duration as needed
	defer reputationDelay.Stop()

	for {
		message := s.awaitMessageFrom(overseerToSubSystem, responderCh)

		switch innerMessage := message.(type) {
		// Handle each muxed message type
		default:
			logger.Warn("Unhandled message type: " + fmt.Sprintf("%v", innerMessage))
		}
	}
}

func (s *StatementDistribution) taskResponder(responderCh chan any) {
	// TODO: Implement taskResponder logic
}

// awaitMessageFrom waits for messages from either the overseerToSubSystem or responderCh
func (s *StatementDistribution) awaitMessageFrom(overseerToSubSystem <-chan any, responderCh chan any) MuxedMessage {
	select {
	case msg := <-overseerToSubSystem:
		return &overseerMessage{inner: msg}
	case msg := <-responderCh:
		return &responderMessage{inner: msg}
	}
}
