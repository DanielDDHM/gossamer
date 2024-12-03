package prospectiveparachains

import (
	"context"
	"errors"

	fragmentchain "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/fragment-chain"
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
	FragmentChains map[parachaintypes.ParaID]*fragmentchain.FragmentChain
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
		panic("not implemented yet: see issue #4310")
	case GetHypotheticalMembership:
		panic("not implemented yet: see issue #4311")
	case GetMinimumRelayParents:
		// Directly use the msg since it's already of type GetMinimumRelayParents
		pp.AnswerMinimumRelayParentsRequest(msg.RelayChainBlockHash, msg.Sender)
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

func (pp *ProspectiveParachains) AnswerMinimumRelayParentsRequest(
	relayChainBlockHash common.Hash,
	sender chan []ParaIDBlockNumber,
) {
	// Slice to store the results
	var result []ParaIDBlockNumber

	// Check if the relayChainBlockHash exists in active_leaves
	if exists := pp.View.ActiveLeaves[relayChainBlockHash]; exists {
		// Retrieve data associated with the relayChainBlockHash
		if leafData, found := pp.View.PerRelayParent[relayChainBlockHash]; found {
			// Iterate over fragment_chains and collect the data
			for paraID, fragmentChain := range leafData.FragmentChains {
				result = append(result, ParaIDBlockNumber{
					ParaId:      paraID,
					BlockNumber: parachaintypes.BlockNumber(fragmentChain.Scope().EarliestRelayParent().Number),
				})
			}
		}
	}

	// Send the result through the sender channel
	sender <- result
}
