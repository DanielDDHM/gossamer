package statementdistribution

import (
	"context"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"

	statementedistributionmessages "github.com/ChainSafe/gossamer/dot/parachain/statement-distribution/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	parachainutil "github.com/ChainSafe/gossamer/dot/parachain/util"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "statement-distribution"))

type StatementDistribution struct {
	reputationAggregator *parachainutil.ReputationAggregator
	SubSystemToOverseer  chan<- parachainutil.NetworkBridgeTxMessage
}

func (s *StatementDistribution) Run(
	ctx context.Context,
	overseerToSubSystem <-chan any,
	v2CommChannel <-chan any,
	receiverRespCh <-chan any,
	retryReqCh <-chan any,
) {
	// Timer for reputation aggregator trigger
	reputationDelay := time.NewTicker(parachainutil.ReputationChangeInterval) // Adjust the duration as needed
	defer reputationDelay.Stop()

	for {
		select {
		case msg := <-overseerToSubSystem:
			err := s.processMessage(msg)
			if err != nil {
				logger.Errorf("error processing overseer message: %v", err)
			}
			// case _ = <-v2CommChannel:
			// 	panic("Not Implemented")
			// case _ = <-receiverRespCh:
			// 	panic("Not Implemented")
			// case _ = <-retryReqCh:
			logger.Infof("received retry request, no action taken")
		case <-reputationDelay.C:
			// Trigger reputation aggregator logic
			s.reputationAggregator.Send(s.SubSystemToOverseer)
		case <-ctx.Done():
			logger.Infof("shutting down: %v", ctx.Err())
			return
		}
	}
}

func (s *StatementDistribution) processMessage(msg any) error {
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

func (s *StatementDistribution) Name() parachaintypes.SubSystemName {
	return parachaintypes.StatementDistribution
}

func (s *StatementDistribution) ProcessActiveLeavesUpdateSignal(signal parachaintypes.ActiveLeavesUpdateSignal) error {
	// TODO #4173
	return nil
}

func (s *StatementDistribution) ProcessBlockFinalizedSignal(signal parachaintypes.BlockFinalizedSignal) error {
	// nothing to do here
	return nil
}

func (s *StatementDistribution) Stop() {}
