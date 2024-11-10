package prospective_parachains

import (
	"fmt"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type ProspectiveParachainMessageValues interface {
	IntroduceSecondedCandidate | CandidateBacked | GetBackableCandidates | GetHypotheticalMembership |
		GetMinimumRelayParents
}

// ProspectiveParachainsMessage Messages sent to the Prospective Parachains subsystem.
type ProspectiveParachainsMessage struct {
	inner any
}

func setProspectiveParachainMessage[Value ProspectiveParachainMessageValues](ppm *ProspectiveParachainsMessage,
	value Value) {
	ppm.inner = value
}

func (ppm *ProspectiveParachainsMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case IntroduceSecondedCandidate:
		setProspectiveParachainMessage(ppm, value)
		return

	case CandidateBacked:
		setProspectiveParachainMessage(ppm, value)
		return

	case GetBackableCandidates:
		setProspectiveParachainMessage(ppm, value)
		return

	case GetHypotheticalMembership:
		setProspectiveParachainMessage(ppm, value)
		return

	case GetMinimumRelayParents:
		setProspectiveParachainMessage(ppm, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (ppm ProspectiveParachainsMessage) IndexValue() (index uint, value any, err error) {
	switch ppm.inner.(type) {
	case IntroduceSecondedCandidate:
		return 0, ppm.inner, nil

	case CandidateBacked:
		return 1, ppm.inner, nil

	case GetBackableCandidates:
		return 2, ppm.inner, nil

	case GetHypotheticalMembership:
		return 3, ppm.inner, nil

	case GetMinimumRelayParents:
		return 4, ppm.inner, nil
	}

	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (ppm ProspectiveParachainsMessage) Value() (value any, err error) {
	_, value, err = ppm.IndexValue()
	return
}

func (ppm ProspectiveParachainsMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(IntroduceSecondedCandidate), nil

	case 1:
		return *new(CandidateBacked), nil

	case 2:
		return *new(GetBackableCandidates), nil

	case 3:
		return *new(GetHypotheticalMembership), nil

	case 4:
		return *new(GetMinimumRelayParents), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollatorProtocolMessage returns a new collator protocol message varying data type
func NewProspectiveParachainMessage() ProspectiveParachainsMessage {
	return ProspectiveParachainsMessage{}
}

type IntroduceSecondedCandidate struct {
	IntroduceSecondedCandidateRequest
	Sender chan bool
}

// IntroduceSecondedCandidateRequest Request introduction of a seconded candidate into the prospective parachains
// subsystem.
type IntroduceSecondedCandidateRequest struct {
	// The para-id of the candidate.
	CandidatePara parachaintypes.ParaID
	// The candidate receipt itself.
	CandidateReceipt parachaintypes.CommittedCandidateReceipt
	// The persisted validation data of the candidate.
	PersistedValidationData parachaintypes.PersistedValidationData
}

// CandidateBacked Inform the Prospective Parachains Subsystem that a previously introduced candidate
// has been backed. This requires that the candidate was successfully introduced in
// the past.
type CandidateBacked struct {
	ParaId        parachaintypes.ParaID
	CandidateHash parachaintypes.CandidateHash
}

// GetBackableCandidates Try getting N backable candidate hashes along with their relay parents for the given
// parachain, under the given relay-parent hash, which is a descendant of the given ancestors.
// Timed out ancestors should not be included in the collection.
// N should represent the number of scheduled cores of this ParaId.
// A timed out ancestor frees the cores of all of its descendants, so if there's a hole in the
// supplied ancestor path, we'll get candidates that backfill those timed out slots first. It
// may also return less/no candidates, if there aren't enough backable candidates recorded.
type GetBackableCandidates struct {
	Hash      common.Hash
	ParaId    parachaintypes.ParaID
	N         uint32
	Ancestors Ancestors
	Sender    chan []struct {
		CandidateHash parachaintypes.CandidateHash
		RelayParent   common.Hash
	}
}

// Ancestors A collection of ancestor candidates of a parachain.
// TODO: is this correct?  Rust code has 'Ancestors' as pub type Ancestors = HashSet<CandidateHash>;
type Ancestors []parachaintypes.CandidateHash

// GetHypotheticalMembership Get the hypothetical or actual membership of candidates with the given properties
// under the specified active leave's fragment chain.
//
// For each candidate, we return a vector of leaves where the candidate is present or could be
// added. "Could be added" either means that the candidate can be added to the chain right now
// or could be added in the future (we may not have its ancestors yet).
// Note that even if we think it could be added in the future, we may find out that it was
// invalid, as time passes.
// If an active leaf is not in the vector, it means that there's no
// chance this candidate will become valid under that leaf in the future.
//
// If `fragment_chain_relay_parent` in the request is `Some()`, the return vector can only
// contain this relay parent (or none).
type GetHypotheticalMembership struct {
	HypotheticalMembershipRequest HypotheticalMembershipRequest
	Sender                        chan []struct {
		HypotheticalCandidate  parachaintypes.HypotheticalCandidate
		HypotheticalMembership HypotheticalMembership
	}
}

// HypotheticalMembershipRequest Request specifying which candidates are either already included
// or might become included in fragment chain under a given active leaf (or any active leaf if
// `fragment_chain_relay_parent` is `None`).
type HypotheticalMembershipRequest struct {
	// Candidates, in arbitrary order, which should be checked for
	// hypothetical/actual membership in fragment chains.
	Candidates []parachaintypes.HypotheticalCandidate
	// Either a specific fragment chain to check, otherwise all.
	FragmentChainRelayParent common.Hash
}

// HypotheticalMembership Indicates the relay-parents whose fragment chain a candidate
// is present in or can be added in (right now or in the future).
type HypotheticalMembership []common.Hash

// GetMinimumRelayParents Get the minimum accepted relay-parent number for each para in the fragment chain
// for the given relay-chain block hash.
//
// That is, if the block hash is known and is an active leaf, this returns the
// minimum relay-parent block number in the same branch of the relay chain which
// is accepted in the fragment chain for each para-id.
//
// If the block hash is not an active leaf, this will return an empty vector.
//
// Para-IDs which are omitted from this list can be assumed to have no
// valid candidate relay-parents under the given relay-chain block hash.
//
// Para-IDs are returned in no particular order.
type GetMinimumRelayParents struct {
	Hash   common.Hash
	Sender chan []struct {
		ParaId      parachaintypes.ParaID
		BlockNumber parachaintypes.BlockNumber
	}
}

// GetProspectiveValidationData Get the validation data of some prospective candidate. The candidate doesn't need
// to be part of any fragment chain, but this only succeeds if the parent head-data and
// relay-parent are part of the `CandidateStorage` (meaning that it's a candidate which is
// part of some fragment chain or which prospective-parachains predicted will become part of
// some fragment chain).
type GetProspectiveValidationData struct {
	ProspectiveValidationDataRequest
	Sender chan parachaintypes.PersistedValidationData
}

// ProspectiveValidationDataRequest A request for the persisted validation data stored in the prospective
// parachains subsystem.
type ProspectiveValidationDataRequest struct {
	// The para-id of the candidate.
	ParaId parachaintypes.ParaID
	// The relay-parent of the candidate.
	CandidateRelayParent common.Hash
	// The parent head-data.
	ParentHeadData ParentHeadData
}

type ParentHeadDataValues interface {
	OnlyHash | WithData
}

// ParentHeadData The parent head-data hash with optional data itself.
type ParentHeadData struct {
	inner any
}

func setParentHeadData[Value ParentHeadDataValues](phd *ParentHeadData, value Value) {
	phd.inner = value
}

func (phd *ParentHeadData) SetValue(value any) (err error) {
	switch value := value.(type) {
	case OnlyHash:
		setParentHeadData(phd, value)
		return

	case WithData:
		setParentHeadData(phd, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (phd ParentHeadData) IndexValue() (index uint, value any, err error) {
	switch phd.inner.(type) {
	case OnlyHash:
		return 0, phd.inner, nil

	case WithData:
		return 1, phd.inner, nil
	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (phd ParentHeadData) Value() (value any, err error) {
	_, value, err = phd.IndexValue()
	return
}

func (phd ParentHeadData) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(OnlyHash), nil

	case 1:
		return *new(WithData), nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// OnlyHash Parent head-data hash.
type OnlyHash common.Hash

type WithData struct {
	/// This will be provided for collations with elastic scaling enabled.
	HeadData parachaintypes.HeadData
	// Parent head-data hash.
	Hash common.Hash
}
