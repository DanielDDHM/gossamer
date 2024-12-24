package statementdistribution

import (
	"context"
	"fmt"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"

	statementedistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statement-distribution"))

type StatementDistribution struct {
}

// MuxedMessage represents a combined message with its source
type MuxedMessage struct {
	Source  string
	Message any
}

func (s StatementDistribution) Run(
	ctx context.Context,
	overseerToSubSystem <-chan any,
	v2CommChannel <-chan any,
	receiverRespCh <-chan any,
	retryReqCh <-chan any,
) {
	// Timer for reputation aggregator trigger
	ticker := time.NewTicker(1 * time.Minute) // Adjust the duration as needed
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-overseerToSubSystem:
			if ok {
				err := s.processMessage(msg)
				if err != nil {
					logger.Errorf("error processing overseer message: %v", err)
				}
			}
		case msg, ok := <-v2CommChannel:
			if ok {
				err := s.processMuxedMessage(MuxedMessage{Source: "V2Responder", Message: msg})
				if err != nil {
					logger.Errorf("error processing V2 responder message: %v", err)
				}
			}
		case msg, ok := <-receiverRespCh:
			if ok {
				err := s.processMuxedMessage(MuxedMessage{Source: "Receive_Response", Message: msg})
				if err != nil {
					logger.Errorf("error processing receiver response message: %v", err)
				}
			}
		case _, ok := <-retryReqCh:
			if ok {
				logger.Infof("received retry request, no action taken")
			}
		case <-ticker.C:
			// Trigger reputation aggregator logic
			s.triggerReputationAggregator()
		case <-ctx.Done():
			logger.Infof("shutting down: %v", ctx.Err())
			return
		}
	}
}

func (s StatementDistribution) processMessage(msg any) error {
	switch msg := msg.(type) {
	case statementedistributionmessages.Backed:
		// TODO #4171
	case statementedistributionmessages.Share:
		// TODO #4170
	case parachaintypes.ActiveLeavesUpdateSignal:
		return s.ProcessActiveLeavesUpdateSignal(msg)
	case parachaintypes.BlockFinalizedSignal:
		return s.ProcessBlockFinalizedSignal(msg)
	default:
		return parachaintypes.ErrUnknownOverseerMessage
	}
	return nil
}

func (s StatementDistribution) processMuxedMessage(muxedMsg MuxedMessage) error {
	switch muxedMsg.Source {
	case "SubsystemMsg":
		return s.processMessage(muxedMsg.Message)
	case "V2Responder":
		// Handle V2Responder messages
		return nil
	case "Receive_Response":
		// Handle response messages
		return nil
	case "Retry_Request":
		logger.Infof("received retry request, no action taken")
		return nil
	default:
		logger.Warnf("unknown message source: %s", muxedMsg.Source)
		return fmt.Errorf("unknown message source: %s", muxedMsg.Source)
	}
}

func (s StatementDistribution) Name() parachaintypes.SubSystemName {
	return parachaintypes.StatementDistribution
}

func (s StatementDistribution) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// TODO #4173
	return nil
}

func (s StatementDistribution) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (s StatementDistribution) triggerReputationAggregator() {
	// Implement the logic to send reputation changes
	logger.Infof("triggering reputation aggregator logic")
}

func (s StatementDistribution) Stop() {}
