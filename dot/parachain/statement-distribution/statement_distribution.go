package statementdistribution

import (
	"context"
	"fmt"

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
	v1RequesterChannel <-chan any,
	v1CommChannel <-chan any,
	v2CommChannel <-chan any,
	receiverRespCh <-chan any,
	retryReqCh <-chan any,
) {
	muxedChannel := FanIn(
		ctx,
		overseerToSubSystem,
		v1RequesterChannel,
		v1CommChannel,
		v2CommChannel,
		receiverRespCh,
		retryReqCh,
	)

	for {
		select {
		case muxedMsg := <-muxedChannel:
			err := s.processMuxedMessage(muxedMsg)
			if err != nil {
				logger.Errorf("error processing muxed message: %w", err)
			}
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
	// case statementedistributionmessages.NetworkBridgeUpdate
	// TODO #4172 this above case would need to wait until network bridge receiver side is merged
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
		// Use processMessage for messages from overseerToSubSystem
		return s.processMessage(muxedMsg.Message)
	case "V1Requester":
		// Handle legacy V1Requester messages
		return nil
	case "V1Responder":
		// Handle legacy V1Responder messages
		return nil
	case "V2Responder":
		// Handle V2Responder messages
		return nil
	case "Receive_Response":
		// Handle response messages
		return nil
	case "Retry_Request":
		// Do nothing for retry requests
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

func (s StatementDistribution) Stop() {}

func FanIn(
	ctx context.Context,
	overseerChannel <-chan any,
	v1RequesterChannel <-chan any,
	v1CommChannel <-chan any,
	v2CommChannel <-chan any,
	receiverRespCh <-chan any,
	retryReqCh <-chan any,
) <-chan MuxedMessage {
	output := make(chan MuxedMessage)

	go func() {
		defer close(output)
		for {
			select {
			// On each case verify if the channel is open before send the message
			case <-ctx.Done():
				return
			case msg, ok := <-overseerChannel:
				if ok {
					output <- MuxedMessage{Source: "SubsystemMsg", Message: msg}
				}
			case msg, ok := <-v1RequesterChannel:
				if ok {
					output <- MuxedMessage{Source: "V1Requester", Message: msg}
				}
			case msg, ok := <-v1CommChannel:
				if ok {
					output <- MuxedMessage{Source: "V1Responder", Message: msg}
				}
			case msg, ok := <-v2CommChannel:
				if ok {
					output <- MuxedMessage{Source: "V2Responder", Message: msg}
				}
			case msg, ok := <-receiverRespCh:
				if ok {
					output <- MuxedMessage{Source: "Receive_Response", Message: msg}
				}
			case msg, ok := <-retryReqCh:
				if ok {
					output <- MuxedMessage{Source: "Retry_Request", Message: msg}
				}
			}
		}
	}()

	return output
}
