package prospectiveparachains

import (
	"errors"
	"fmt"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var (
	errCandidateAlreadyKnown           = errors.New("candidate already known")
	errZeroLengthCycle                 = errors.New("candidate's parent head is equal to its output head. Would introduce a cycle") //nolint:lll
	errCycle                           = errors.New("candidate would introduce a cycle")
	errMultiplePaths                   = errors.New("candidate would introduce two paths to the same output state")
	errIntroduceBackedCandidate        = errors.New("attempting to directly introduce a Backed candidate. It should first be introduced as Seconded") //nolint:lll
	errParentCandidateNotFound         = errors.New("could not find parent of the candidate")
	errRelayParentMovedBackwards       = errors.New("relay parent would move backwards from the latest candidate in the chain")     //nolint:lll
	errPersistedValidationDataMismatch = errors.New("candidate does not match the persisted validation data provided alongside it") //nolint:lll
)

type errRelayParentPrecedesCandidatePendingAvailability struct {
	relayParentA, relayParentB common.Hash
}

func (e errRelayParentPrecedesCandidatePendingAvailability) Error() string {
	return fmt.Sprintf("relay parent %x of the candidate precedes the relay parent %x of a pending availability candidate",
		e.relayParentA, e.relayParentB)
}

type errForkWithCandidatePendingAvailability struct {
	candidateHash parachaintypes.CandidateHash
}

func (e errForkWithCandidatePendingAvailability) Error() string {
	return fmt.Sprintf("candidate would introduce a fork with a pending availability candidate: %x", e.candidateHash.Value)
}

type errForkChoiceRule struct {
	candidateHash parachaintypes.CandidateHash
}

func (e errForkChoiceRule) Error() string {
	return fmt.Sprintf("fork selection rule favours another candidate: %x", e.candidateHash.Value)
}

type errComputeConstraints struct {
	modificationErr error
}

func (e errComputeConstraints) Error() string {
	return fmt.Sprintf("could not compute candidate constraints: %s", e.modificationErr)
}

type errCheckAgainstConstraints struct {
	fragmentValidityErr error
}

func (e errCheckAgainstConstraints) Error() string {
	return fmt.Sprintf("candidate violates constraints: %s", e.fragmentValidityErr)
}

type errRelayParentNotInScope struct {
	relayParentA, relayParentB common.Hash
}

func (e errRelayParentNotInScope) Error() string {
	return fmt.Sprintf("relay parent %s not in scope, earliest relay parent allowed %s",
		e.relayParentA.String(), e.relayParentB.String())
}

type errUnexpectedAncestor struct {
	// The block number that this error occurred at
	Number uint
	// The previous seen block number, which did not match `number`.
	Prev uint
}

func (e errUnexpectedAncestor) Error() string {
	return fmt.Sprintf("unexpected ancestor %d, expected %d", e.Number, e.Prev)
}
