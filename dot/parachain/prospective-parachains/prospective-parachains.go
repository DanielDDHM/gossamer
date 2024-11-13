package prospective_parachains

import (
	"context"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "prospective_parachains"), log.SetLevel(log.Debug))

type ProspectiveParachains struct {
	SubsystemToOverseer chan<- any
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

// Run starts the CandidateValidation subsystem
func (pp *ProspectiveParachains) Run(ctx context.Context, overseerToSubsystem <-chan any) {
	for {
		select {
		case msg := <-overseerToSubsystem:
			pp.processMessage(msg)
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
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
		panic("not implemented yet")
	case CandidateBacked:
		panic("not implemented yet")
	case GetBackableCandidates:
		panic("not implemented yet")
	case GetHypotheticalMembership:
		panic("not implemented yet")
	case GetMinimumRelayParents:
		panic("not implemented yet")
	case GetProspectiveValidationData:
		panic("not implemented yet")
	default:
		logger.Errorf("%w: %T", parachaintypes.ErrUnknownOverseerMessage, msg)
	}

}

// ProcessActiveLeavesUpdateSignal processes active leaves update signal
func (pp *ProspectiveParachains) ProcessActiveLeavesUpdateSignal(parachaintypes.ActiveLeavesUpdateSignal) error {
	panic("not implemented yet")
}

// ProcessBlockFinalizedSignal processes block finalized signal
func (*ProspectiveParachains) ProcessBlockFinalizedSignal(parachaintypes.BlockFinalizedSignal) error {
	// NOTE: this subsystem does not process block finalized signal
	return nil
}
