package prospectiveparachains

import (
	"context"
	"errors"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "prospective_parachains"), log.SetLevel(log.Debug))

type ProspectiveParachains struct {
	SubsystemToOverseer chan<- any
	View                *View
}

type View struct {
	ActiveLeaves   map[common.Hash]bool
	PerRelayParent map[common.Hash]*RelayParentData
}

type RelayParentData struct {
	FragmentChains map[parachaintypes.ParaID]*fragmentChain
}

// Name returns the name of the subsystem
func (*ProspectiveParachains) Name() parachaintypes.SubSystemName {
	return parachaintypes.ProspectiveParachains
}

// NewProspectiveParachains creates a new ProspectiveParachain subsystem
func NewProspectiveParachains(overseerChan chan<- any) *ProspectiveParachains {
	prospectiveParachain := ProspectiveParachains{
		SubsystemToOverseer: overseerChan,
	}
	return &prospectiveParachain
}

// Run starts the ProspectiveParachains subsystem
func (pp *ProspectiveParachains) Run(ctx context.Context, overseerToSubsystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubsystem:
			pp.processMessage(msg)
		case <-ctx.Done():
			if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
				logger.Errorf("ctx error: %s\n", err)
			}
			return
		}
	}
}

func (*ProspectiveParachains) Stop() {}

func (pp *ProspectiveParachains) processMessage(msg any) {
	switch msg := msg.(type) {
	case parachaintypes.Conclude:
		pp.Stop()
	case parachaintypes.ActiveLeavesUpdateSignal:
		_ = pp.ProcessActiveLeavesUpdateSignal(msg)
	case parachaintypes.BlockFinalizedSignal:
		_ = pp.ProcessBlockFinalizedSignal(msg)
	case IntroduceSecondedCandidate:
		panic("not implemented yet: see issue #4308")
	case CandidateBacked:
		panic("not implemented yet: see issue #4309")
	case GetBackableCandidates:
		go pp.getBackableCandidates(msg)
	case GetHypotheticalMembership:
		panic("not implemented yet: see issue #4311")
	case GetMinimumRelayParents:
		panic("not implemented yet: see issue #4312")
	case GetProspectiveValidationData:
		panic("not implemented yet: see issue #4313")
	default:
		logger.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}

}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (pp *ProspectiveParachains) ProcessActiveLeavesUpdateSignal(parachaintypes.ActiveLeavesUpdateSignal) error {
	panic("not implemented yet: see issue #4305")
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (*ProspectiveParachains) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// NOTE: this subsystem does not process block finalized signal
	return nil
}

func (pp *ProspectiveParachains) getBackableCandidates(
	msg GetBackableCandidates,
) {
	// Extract details from the message
	relayParentHash := msg.RelayParentHash
	paraId := msg.ParaId
	requestedQty := msg.RequestedQty
	ancestors := msg.Ancestors
	responseChan := msg.Response

	// Check if the relay parent is active
	if _, exists := pp.View.ActiveLeaves[relayParentHash]; !exists {
		logger.Debugf(
			"Requested backable candidates for inactive relay-parent. "+
				"RelayParentHash: %v, ParaId: %v",
			relayParentHash, paraId,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	// Retrieve data for the relay parent
	data, ok := pp.View.PerRelayParent[relayParentHash]
	if !ok {
		logger.Debugf(
			"Requested backable candidates for nonexistent relay-parent. "+
				"RelayParentHash: %v, ParaId: %v",
			relayParentHash, paraId,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	// Retrieve the fragment chain for the ParaID
	chain, ok := data.FragmentChains[paraId]
	if !ok {
		logger.Debugf(
			"Requested backable candidates for inactive ParaID. "+
				"RelayParentHash: %v, ParaId: %v",
			relayParentHash, paraId,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	// Retrieve backable candidates from the fragment chain
	backableCandidates := chain.FindBackableChain(ancestors, requestedQty)
	if len(backableCandidates) == 0 {
		logger.Tracef(
			"No backable candidates found. RelayParentHash: %v, ParaId: %v, Ancestors: %v",
			relayParentHash, paraId, ancestors,
		)
		responseChan <- []parachaintypes.CandidateHashAndRelayParent{}
		return
	}

	logger.Tracef(
		"Found backable candidates: %v. RelayParentHash: %v, ParaId: %v, Ancestors: %v",
		backableCandidates, relayParentHash, paraId, ancestors,
	)

	// Convert backable candidates to the expected response format
	candidateHashes := make([]parachaintypes.CandidateHashAndRelayParent, len(backableCandidates))
	for i, candidate := range backableCandidates {
		candidateHashes[i] = parachaintypes.CandidateHashAndRelayParent{
			CandidateHash:        candidate.candidateHash,
			CandidateRelayParent: candidate.realyParentHash,
		}
	}

	// Send the result through the response channel
	responseChan <- candidateHashes
}
