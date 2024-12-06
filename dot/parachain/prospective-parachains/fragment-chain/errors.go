package fragmentchain

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	ErrCandidateAlreadyKnown           = errors.New("candidate already known")
	ErrZeroLengthCycle                 = errors.New("candidate's parent head is equal to its output head. Would introduce a cycle")
	ErrCycle                           = errors.New("candidate would introduce a cycle")
	ErrMultiplePaths                   = errors.New("candidate would introduce two paths to the same output state")
	ErrIntroduceBackedCandidate        = errors.New("attempting to directly introduce a Backed candidate. It should first be introduced as Seconded")
	ErrParentCandidateNotFound         = errors.New("could not find parent of the candidate")
	ErrRelayParentMovedBackwards       = errors.New("relay parent would move backwards from the latest candidate in the chain")
	ErrPersistedValidationDataMismatch = errors.New("candidate does not match the persisted validation data provided alongside it")
)

type ErrRelayParentPrecedesCandidatePendingAvailability struct {
	relayParentA, relayParentB common.Hash
}

func (e ErrRelayParentPrecedesCandidatePendingAvailability) Error() string {
	return fmt.Sprintf("relay parent %x of the candidate precedes the relay parent %x of a pending availability candidate",
		e.relayParentA, e.relayParentB)
}

type ErrForkWithCandidatePendingAvailability struct {
	candidateHash parachaintypes.CandidateHash
}

func (e ErrForkWithCandidatePendingAvailability) Error() string {
	return fmt.Sprintf("candidate would introduce a fork with a pending availability candidate: %x", e.candidateHash.Value)
}

type ErrForkChoiceRule struct {
	candidateHash parachaintypes.CandidateHash
}

func (e ErrForkChoiceRule) Error() string {
	return fmt.Sprintf("fork selection rule favours another candidate: %x", e.candidateHash.Value)
}

type ErrComputeConstraints struct {
	modificationErr error
}

func (e ErrComputeConstraints) Error() string {
	return fmt.Sprintf("could not compute candidate constraints: %s", e.modificationErr)
}

type ErrCheckAgainstConstraints struct {
	fragmentValidityErr error
}

func (e ErrCheckAgainstConstraints) Error() string {
	return fmt.Sprintf("candidate violates constraints: %s", e.fragmentValidityErr)
}

type ErrRelayParentNotInScope struct {
	relayParentA, relayParentB common.Hash
}

func (e ErrRelayParentNotInScope) Error() string {
	return fmt.Sprintf("relay parent %s not in scope, earliest relay parent allowed %s",
		e.relayParentA.String(), e.relayParentB.String())
}

type ErrUnexpectedAncestor struct {
	// The block number that this error occurred at
	Number uint
	// The previous seen block number, which did not match `number`.
	Prev uint
}

func (e ErrUnexpectedAncestor) Error() string {
	return fmt.Sprintf("unexpected ancestor %d, expected %d", e.Number, e.Prev)
}
