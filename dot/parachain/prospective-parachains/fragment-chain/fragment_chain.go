package fragmentchain

import (
	"bytes"
	"container/list"
	"fmt"
	"iter"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	inclusionemulator "github.com/ChainSafe/gossamer/dot/parachain/util/inclusion-emulator"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/tidwall/btree"
)

type CandidateState int

const (
	Seconded CandidateState = iota
	Backed
)

func forkSelectionRule(hash1, hash2 parachaintypes.CandidateHash) int {
	return bytes.Compare(hash1.Value[:], hash2.Value[:])
}

// CandidateEntry represents a candidate into the CandidateStorage
// TODO: Should CandidateEntry implements `HypotheticalOrConcreteCandidate`
type CandidateEntry struct {
	candidateHash      parachaintypes.CandidateHash
	parentHeadDataHash common.Hash
	outputHeadDataHash common.Hash
	relayParent        common.Hash
	// TODO: this is under a Arc<ProspectiveCandidate> smart pointer, should we
	// have that here? maybe some specialized struct that protects the underlying data?
	candidate inclusionemulator.ProspectiveCandidate
	state     CandidateState
}

func (c *CandidateEntry) Hash() parachaintypes.CandidateHash {
	return c.candidateHash
}

func NewCandidateEntry(
	candidateHash parachaintypes.CandidateHash,
	candidate parachaintypes.CommittedCandidateReceipt,
	persistedValidationData parachaintypes.PersistedValidationData,
	state CandidateState,
) (*CandidateEntry, error) {
	pvdHash, err := persistedValidationData.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing persisted validation data: %w", err)
	}

	if pvdHash != candidate.Descriptor.PersistedValidationDataHash {
		return nil, ErrPersistedValidationDataMismatch
	}

	parendHeadDataHash, err := persistedValidationData.ParentHead.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing parent head data: %w", err)
	}

	outputHeadDataHash, err := candidate.Commitments.HeadData.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing output head data: %w", err)
	}

	if parendHeadDataHash == outputHeadDataHash {
		return nil, ErrCandidateEntryZeroLengthCycle
	}

	return &CandidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parendHeadDataHash,
		outputHeadDataHash: outputHeadDataHash,
		relayParent:        candidate.Descriptor.RelayParent,
		state:              state,
		candidate: inclusionemulator.ProspectiveCandidate{
			Commitments:             candidate.Commitments,
			PersistedValidationData: persistedValidationData,
			PoVHash:                 candidate.Descriptor.PovHash,
			ValidationCodeHash:      candidate.Descriptor.ValidationCodeHash,
		},
	}, nil
}

// CandidateStorage is an utility for storing candidates and information about them such as
// their relay-parents and their backing states. This does not assume any restriction on whether
// or not candidates form a chain. Useful for storing all kinds of candidates.
type CandidateStorage struct {
	byParentHead    map[common.Hash]map[parachaintypes.CandidateHash]any
	byOutputHead    map[common.Hash]map[parachaintypes.CandidateHash]any
	byCandidateHash map[parachaintypes.CandidateHash]*CandidateEntry
}

func (c *CandidateStorage) Clone() *CandidateStorage {
	clone := NewCandidateStorage()

	for parentHead, candidates := range c.byParentHead {
		clone.byParentHead[parentHead] = make(map[parachaintypes.CandidateHash]any)
		for candidateHash := range candidates {
			clone.byParentHead[parentHead][candidateHash] = struct{}{}
		}
	}

	for outputHead, candidates := range c.byOutputHead {
		clone.byOutputHead[outputHead] = make(map[parachaintypes.CandidateHash]any)
		for candidateHash := range candidates {
			clone.byOutputHead[outputHead][candidateHash] = struct{}{}
		}
	}

	for candidateHash, entry := range c.byCandidateHash {
		clone.byCandidateHash[candidateHash] = &CandidateEntry{
			candidateHash:      entry.candidateHash,
			parentHeadDataHash: entry.parentHeadDataHash,
			outputHeadDataHash: entry.outputHeadDataHash,
			relayParent:        entry.relayParent,
			candidate: inclusionemulator.ProspectiveCandidate{
				Commitments:             entry.candidate.Commitments,
				PersistedValidationData: entry.candidate.PersistedValidationData,
				PoVHash:                 entry.candidate.PoVHash,
				ValidationCodeHash:      entry.candidate.ValidationCodeHash,
			},
			state: entry.state,
		}
	}

	return clone
}

func NewCandidateStorage() *CandidateStorage {
	return &CandidateStorage{
		byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
		byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
		byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
	}
}

func (c *CandidateStorage) AddPendingAvailabilityCandidate(
	candidateHash parachaintypes.CandidateHash,
	candidate parachaintypes.CommittedCandidateReceipt,
	persistedValidationData parachaintypes.PersistedValidationData,
) error {
	entry, err := NewCandidateEntry(candidateHash, candidate, persistedValidationData, Backed)
	if err != nil {
		return err
	}

	return c.addCandidateEntry(entry)
}

// Len return the number of stored candidate
func (c *CandidateStorage) Len() int {
	return len(c.byCandidateHash)
}

func (c *CandidateStorage) addCandidateEntry(candidate *CandidateEntry) error {
	_, ok := c.byCandidateHash[candidate.candidateHash]
	if ok {
		return ErrCandidateAlradyKnown
	}

	// updates the reference parent hash -> candidate
	// we don't check the `ok` value since the key can
	// exists in the map but pointing to a nil hashset
	setOfCandidates := c.byParentHead[candidate.parentHeadDataHash]
	if setOfCandidates == nil {
		setOfCandidates = make(map[parachaintypes.CandidateHash]any)
	}
	setOfCandidates[candidate.candidateHash] = struct{}{}
	c.byParentHead[candidate.parentHeadDataHash] = setOfCandidates

	// udpates the reference output hash -> candidate
	setOfCandidates = c.byOutputHead[candidate.outputHeadDataHash]
	if setOfCandidates == nil {
		setOfCandidates = make(map[parachaintypes.CandidateHash]any)
	}
	setOfCandidates[candidate.candidateHash] = struct{}{}
	c.byOutputHead[candidate.outputHeadDataHash] = setOfCandidates

	c.byCandidateHash[candidate.candidateHash] = candidate
	return nil
}

func (c *CandidateStorage) removeCandidate(candidateHash parachaintypes.CandidateHash) {
	entry, ok := c.byCandidateHash[candidateHash]
	if !ok {
		return
	}

	delete(c.byCandidateHash, candidateHash)

	if setOfCandidates, ok := c.byParentHead[entry.parentHeadDataHash]; ok {
		delete(setOfCandidates, candidateHash)
		if len(setOfCandidates) == 0 {
			delete(c.byParentHead, entry.parentHeadDataHash)
		}
	}

	if setOfCandidates, ok := c.byOutputHead[entry.outputHeadDataHash]; ok {
		delete(setOfCandidates, candidateHash)
		if len(setOfCandidates) == 0 {
			delete(c.byOutputHead, entry.outputHeadDataHash)
		}
	}
}

func (c *CandidateStorage) markBacked(candidateHash parachaintypes.CandidateHash) {
	entry, ok := c.byCandidateHash[candidateHash]
	if !ok {
		fmt.Println("candidate not found while marking as backed")
	}

	entry.state = Backed
}

func (c *CandidateStorage) contains(candidateHash parachaintypes.CandidateHash) bool {
	_, ok := c.byCandidateHash[candidateHash]
	return ok
}

// candidates returns an iterator over references to the stored candidates, in arbitrary order.
func (c *CandidateStorage) candidates() iter.Seq[*CandidateEntry] {
	return func(yield func(*CandidateEntry) bool) {
		for _, entry := range c.byCandidateHash {
			if !yield(entry) {
				return
			}
		}
	}
}

func (c *CandidateStorage) headDataByHash(hash common.Hash) *parachaintypes.HeadData {
	// first, search for candidates outputting this head data and extract the head data
	// from their commitments if they exist.
	// otherwise, search for candidates building upon this head data and extract the
	// head data from their persisted validation data if they exist.

	if setOfCandidateHashes, ok := c.byOutputHead[hash]; ok {
		for candidateHash := range setOfCandidateHashes {
			if candidate, ok := c.byCandidateHash[candidateHash]; ok {
				return &candidate.candidate.Commitments.HeadData
			}
		}
	}

	if setOfCandidateHashes, ok := c.byParentHead[hash]; ok {
		for candidateHash := range setOfCandidateHashes {
			if candidate, ok := c.byCandidateHash[candidateHash]; ok {
				return &candidate.candidate.PersistedValidationData.ParentHead
			}
		}
	}

	return nil
}

func (c *CandidateStorage) possibleBackedParaChildren(parentHeadHash common.Hash) iter.Seq[*CandidateEntry] {
	return func(yield func(*CandidateEntry) bool) {
		seqOfCandidateHashes, ok := c.byParentHead[parentHeadHash]
		if !ok {
			return
		}

		for candidateHash := range seqOfCandidateHashes {
			if entry, ok := c.byCandidateHash[candidateHash]; ok && entry.state == Backed {
				if !yield(entry) {
					return
				}
			}
		}
	}
}

// PendindAvailability is a candidate on-chain but pending availability, for special
// treatment in the `Scope`
type PendindAvailability struct {
	CandidateHash parachaintypes.CandidateHash
	RelayParent   inclusionemulator.RelayChainBlockInfo
}

// The scope of a fragment chain
type Scope struct {
	// the relay parent we're currently building on top of
	relayParent inclusionemulator.RelayChainBlockInfo
	// the other relay parents candidates are allowed to build upon,
	// mapped by the block number
	ancestors *btree.Map[uint, inclusionemulator.RelayChainBlockInfo]
	// the other relay parents candidates are allowed to build upon,
	// mapped by hash
	ancestorsByHash map[common.Hash]inclusionemulator.RelayChainBlockInfo
	// candidates pending availability at this block
	pendindAvailability []*PendindAvailability
	// the base constraints derived from the latest included candidate
	baseConstraints *inclusionemulator.Constraints
	// equal to `max_candidate_depth`
	maxDepth uint
}

// NewScopeWithAncestors defines a new scope, all arguments are straightforward
// expect ancestors. Ancestor should be in reverse order, starting with the parent
// of the relayParent, and proceeding backwards in block number decrements of 1.
// Ancestors not following these conditions will be rejected.
//
// This function will only consume ancestors up to the `MinRelayParentNumber` of the
// `baseConstraints`.
//
// Only ancestor whose children have the same session id as the relay parent's children
// should be provided. It is allowed to provide 0 ancestors.
func NewScopeWithAncestors(
	relayParent inclusionemulator.RelayChainBlockInfo,
	baseConstraints *inclusionemulator.Constraints,
	pendingAvailability []*PendindAvailability,
	maxDepth uint,
	ancestors []inclusionemulator.RelayChainBlockInfo,
) (*Scope, error) {
	ancestorsMap := btree.NewMap[uint, inclusionemulator.RelayChainBlockInfo](100)
	ancestorsByHash := make(map[common.Hash]inclusionemulator.RelayChainBlockInfo)

	prev := relayParent.Number
	for _, ancestor := range ancestors {
		if prev == 0 {
			return nil, ErrUnexpectedAncestor{Number: ancestor.Number, Prev: prev}
		}

		if ancestor.Number != prev-1 {
			return nil, ErrUnexpectedAncestor{Number: ancestor.Number, Prev: prev}
		}

		if prev == baseConstraints.MinRelayParentNumber {
			break
		}

		prev = ancestor.Number
		ancestorsByHash[ancestor.Hash] = ancestor
		ancestorsMap.Set(ancestor.Number, ancestor)
	}

	return &Scope{
		relayParent:         relayParent,
		baseConstraints:     baseConstraints,
		pendindAvailability: pendingAvailability,
		maxDepth:            maxDepth,
		ancestors:           ancestorsMap,
		ancestorsByHash:     ancestorsByHash,
	}, nil
}

// EarliestRelayParent gets the earliest relay-parent allowed in the scope of the fragment chain.
func (s *Scope) EarliestRelayParent() inclusionemulator.RelayChainBlockInfo {
	if iter := s.ancestors.Iter(); iter.Next() {
		return iter.Value()
	}
	return s.relayParent
}

// Ancestor gets the relay ancestor of the fragment chain by hash.
func (s *Scope) Ancestor(hash common.Hash) *inclusionemulator.RelayChainBlockInfo {
	if hash == s.relayParent.Hash {
		return &s.relayParent
	}

	if blockInfo, ok := s.ancestorsByHash[hash]; ok {
		return &blockInfo
	}

	return nil
}

// Whether the candidate in question is one pending availability in this scope.
func (s *Scope) GetPendingAvailability(candidateHash parachaintypes.CandidateHash) *PendindAvailability {
	for _, c := range s.pendindAvailability {
		if c.CandidateHash == candidateHash {
			return c
		}
	}
	return nil
}

// Fragment node is a node that belongs to a `BackedChain`. It holds constraints based on
// the ancestors in the chain
type FragmentNode struct {
	fragment                *inclusionemulator.Fragment
	candidateHash           parachaintypes.CandidateHash
	cumulativeModifications *inclusionemulator.ConstraintModifications
	parentHeadDataHash      common.Hash
	outputHeadDataHash      common.Hash
}

func (f *FragmentNode) relayParent() common.Hash {
	return f.fragment.RelayParent().Hash
}

// NewCandidateEntryFromFragment creates a candidate entry from a fragment, we dont need
// to perform the checks done in `NewCandidateEntry` since a `FragmentNode` always comes
// from a `CandidateEntry`
func NewCandidateEntryFromFragment(node *FragmentNode) *CandidateEntry {
	return &CandidateEntry{
		candidateHash:      node.candidateHash,
		parentHeadDataHash: node.parentHeadDataHash,
		outputHeadDataHash: node.outputHeadDataHash,
		candidate:          node.fragment.Candidate(),
		relayParent:        node.relayParent(),
		// a fragment node is always backed
		state: Backed,
	}
}

// BackedChain is a chain of backed/backable candidates
// Includes candidates pending availability and candidates which may be backed on-chain
type BackedChain struct {
	// holds the candidate chain
	chain []*FragmentNode

	// index from parent head data to the candidate that has that head data as parent
	// only contains the candidates present in the `chain`
	byParentHead map[common.Hash]parachaintypes.CandidateHash

	// index from head data hash to the candidate hash outputting that head data
	// only contains the candidates present in the `chain`
	byOutputHead map[common.Hash]parachaintypes.CandidateHash

	// a set of candidate hashes in the `chain`
	candidates map[parachaintypes.CandidateHash]struct{}
}

func NewBackedChain() *BackedChain {
	return &BackedChain{
		chain:        make([]*FragmentNode, 0),
		byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
		byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
		candidates:   make(map[parachaintypes.CandidateHash]struct{}),
	}
}

func (bc *BackedChain) Push(candidate *FragmentNode) {
	bc.candidates[candidate.candidateHash] = struct{}{}
	bc.byParentHead[candidate.parentHeadDataHash] = candidate.candidateHash
	bc.byOutputHead[candidate.outputHeadDataHash] = candidate.candidateHash
	bc.chain = append(bc.chain, candidate)
}

func (bc *BackedChain) Clear() []*FragmentNode {
	bc.byParentHead = make(map[common.Hash]parachaintypes.CandidateHash)
	bc.byOutputHead = make(map[common.Hash]parachaintypes.CandidateHash)
	bc.candidates = make(map[parachaintypes.CandidateHash]struct{})

	oldChain := bc.chain
	bc.chain = nil
	return oldChain
}

func (bc *BackedChain) RevertToParentHash(parentHeadDataHash common.Hash) []*FragmentNode {
	foundIndex := -1

	for i := 0; i < len(bc.chain); i++ {
		node := bc.chain[i]

		if foundIndex != -1 {
			delete(bc.byParentHead, node.parentHeadDataHash)
			delete(bc.byOutputHead, node.outputHeadDataHash)
			delete(bc.candidates, node.candidateHash)
		} else if node.outputHeadDataHash == parentHeadDataHash {
			foundIndex = i
		}
	}

	if foundIndex != -1 {
		// drain the elements from the found index until
		// the end of the slice and return them
		removed := make([]*FragmentNode, len(bc.chain)-(foundIndex+1))
		copy(removed, bc.chain[foundIndex+1:])
		bc.chain = slices.Delete(bc.chain, foundIndex+1, len(bc.chain))

		return removed
	}

	return nil
}

func (bc *BackedChain) Contains(hash parachaintypes.CandidateHash) bool {
	_, ok := bc.candidates[hash]
	return ok
}

// this is a fragment chain specific to an active leaf. It holds the current
// best backable candidate chain, as well as potential candidates which could
// become connected to the chain in the future or which could even overwrite
// the existing chain
type FragmentChain struct {
	// the current scope, which dictates the on-chain operating constraints that
	// all future candidates must ad-here to.
	scope *Scope

	// the current best chain of backable candidates. It only contains candidates
	// which build on top of each other and which have reached the backing quorum.
	// In the presence of potential forks, this chain will pick a fork according to
	// the `forkSelectionRule`
	bestChain *BackedChain

	// the potential candidate storage. Contains candidates which are not yet part of
	// the `chain` but may become in the future. These can form any tree shape as well
	// as contain unconnected candidates for which we don't know the parent.
	unconnected *CandidateStorage
}

// NewFragmentChain createa a new fragment chain with the given scope and populates it with
// the candidates pending availability
func NewFragmentChain(scope *Scope, candidatesPendingAvailability *CandidateStorage) *FragmentChain {
	fragmentChain := &FragmentChain{
		scope:       scope,
		bestChain:   NewBackedChain(),
		unconnected: NewCandidateStorage(),
	}

	// we only need to populate the best backable chain. Candidates pending availability
	// must form a chain with the latest included head.
	fragmentChain.populateChain(candidatesPendingAvailability)
	return fragmentChain
}

// PopulateFromPrevious populates the `FragmentChain` given the new candidates pending
// availability and the optional previous fragment chain (of the previous relay parent)
func (f *FragmentChain) PopulateFromPrevious(prevFragmentChain *FragmentChain) {
	prevStorage := prevFragmentChain.unconnected.Clone()
	for _, candidate := range prevFragmentChain.bestChain.chain {
		// if they used to be pending availability, dont add them. This is fine because:
		// - if they still are pending availability, they have already been added to
		// the new storage
		// - if they were included, no point in keeping them
		//
		// This cannot happen for the candidates in the unconnected storage. The pending
		// availability candidates will always be part of the best chain
		pending := prevFragmentChain.scope.GetPendingAvailability(candidate.candidateHash)
		if pending == nil {
			prevStorage.addCandidateEntry(NewCandidateEntryFromFragment(candidate))
		}
	}

	// first populate the best backable chain
	f.populateChain(prevStorage)

	// now that we picked the best backable chain, trim the forks generated by candidates
	// which are not present in the best chain
	f.trimUneligibleForks(prevStorage, nil)

	// finally, keep any candidates which haven't been trimmed but still have potential
	f.populateUnconnectedPotentialCandidates(prevStorage)
}

func (f *FragmentChain) Scope() *Scope {
	return f.scope
}

func (f *FragmentChain) BestChainLen() int {
	return len(f.bestChain.chain)
}

func (f *FragmentChain) UnconnectedLen() int {
	return f.unconnected.Len()
}

func (f *FragmentChain) ContainsUnconnectedCandidate(candidate parachaintypes.CandidateHash) bool {
	return f.unconnected.contains(candidate)
}

// BestChainVec returns a vector of the chain's candidate hashes, in-order.
func (f *FragmentChain) BestChainVec() (hashes []parachaintypes.CandidateHash) {
	hashes = make([]parachaintypes.CandidateHash, len(f.bestChain.chain))
	for idx, node := range f.bestChain.chain {
		hashes[idx] = node.candidateHash
	}
	return hashes
}

// Unconnected returns a vector of the unconnected potential candidate hashes, in arbitrary order.
func (f *FragmentChain) Unconnected() iter.Seq[*CandidateEntry] {
	return f.unconnected.candidates()
}

func (f *FragmentChain) IsCandidateBacked(hash parachaintypes.CandidateHash) bool {
	if f.bestChain.Contains(hash) {
		return true
	}

	candidate := f.unconnected.byCandidateHash[hash]
	return candidate != nil && candidate.state == Backed
}

// CandidateBacked marks a candidate as backed. This can trigger a recreation of the best backable chain.
func (f *FragmentChain) CandidateBacked(newlyBackedCandidate parachaintypes.CandidateHash) {
	// already backed
	if f.bestChain.Contains(newlyBackedCandidate) {
		return
	}

	candidateEntry, ok := f.unconnected.byCandidateHash[newlyBackedCandidate]
	if !ok {
		// candidate is not in unconnected storage
		return
	}

	parentHeadDataHash := candidateEntry.parentHeadDataHash

	f.unconnected.markBacked(newlyBackedCandidate)

	if !f.revertTo(parentHeadDataHash) {
		// if nothing was reverted, there is nothing we can do for now
		return
	}

	prevStorage := f.unconnected.Clone()
	f.unconnected = NewCandidateStorage()

	f.populateChain(prevStorage)
	f.trimUneligibleForks(prevStorage, &parentHeadDataHash)
	f.populateUnconnectedPotentialCandidates(prevStorage)
}

// CanAddCandidateAsPotential checks if this candidate could be added in the future
func (f *FragmentChain) CanAddCandidateAsPotential(entry *CandidateEntry) error {
	candidateHash := entry.candidateHash
	if f.bestChain.Contains(candidateHash) || f.unconnected.contains(candidateHash) {
		return ErrCandidateAlradyKnown
	}

	return f.checkPotential(entry)
}

// TryAddingSecondedCandidate tries to add a candidate as a seconded candidate, if the
// candidate has potential. It will never be added to the chain directly in the seconded
// state, it will only be part of the unconnected storage
func (f *FragmentChain) TryAddingSecondedCandidate(entry *CandidateEntry) error {
	if entry.state == Backed {
		return ErrIntroduceBackedCandidate
	}

	err := f.CanAddCandidateAsPotential(entry)
	if err != nil {
		return err
	}

	return f.unconnected.addCandidateEntry(entry)
}

// GetHeadDataByHash tries to get the full head data associated with this hash
func (f *FragmentChain) GetHeadDataByHash(headDataHash common.Hash) (*parachaintypes.HeadData, error) {
	reqParent := f.scope.baseConstraints.RequiredParent
	reqParentHash, err := reqParent.Hash()
	if err != nil {
		return nil, fmt.Errorf("while hashing required parent: %w", err)
	}
	if reqParentHash == headDataHash {
		return &reqParent, nil
	}

	hasHeadDataInChain := false
	if _, ok := f.bestChain.byParentHead[headDataHash]; ok {
		hasHeadDataInChain = true
	} else if _, ok := f.bestChain.byOutputHead[headDataHash]; ok {
		hasHeadDataInChain = true
	}

	if hasHeadDataInChain {
		for _, candidate := range f.bestChain.chain {
			if candidate.parentHeadDataHash == headDataHash {
				headData := candidate.
					fragment.
					Candidate().
					PersistedValidationData.
					ParentHead
				return &headData, nil
			} else if candidate.outputHeadDataHash == headDataHash {
				headData := candidate.fragment.Candidate().Commitments.HeadData
				return &headData, nil
			} else {
				continue
			}
		}
	}

	return f.unconnected.headDataByHash(headDataHash), nil
}

type CandidateAndRelayParent struct {
	CandidateHash   parachaintypes.CandidateHash
	RealyParentHash common.Hash
}

// FindBackableChain selects `count` candidates after the given `ancestors` which
// can be backed on chain next. The intention of the `ancestors` is to allow queries
// on the basis of one or more candidates which were previously pending availability
// becoming available or candidates timing out
func (f *FragmentChain) FindBackableChain(
	ancestors map[parachaintypes.CandidateHash]struct{}, count uint32) []*CandidateAndRelayParent {
	if count == 0 {
		return nil
	}

	basePos := f.findAncestorPath(ancestors)

	actualEndIdx := min(basePos+int(count), len(f.bestChain.chain))
	res := make([]*CandidateAndRelayParent, 0, actualEndIdx-basePos)

	for _, elem := range f.bestChain.chain[basePos:actualEndIdx] {
		// only supply candidates which are not yet pending availability.
		// `ancestors` should have already contained them, but check just in case
		if pending := f.scope.GetPendingAvailability(elem.candidateHash); pending == nil {
			res = append(res, &CandidateAndRelayParent{
				CandidateHash:   elem.candidateHash,
				RealyParentHash: elem.relayParent(),
			})
		} else {
			break
		}
	}

	return res
}

// findAncestorPath tries to orders the ancestors into a viable path from root to the last one.
// stops when the ancestors are all used or when a node in the chain is not present in the
// ancestors set. Returns the index in the chain were the search stopped
func (f *FragmentChain) findAncestorPath(ancestors map[parachaintypes.CandidateHash]struct{}) int {
	if len(f.bestChain.chain) == 0 {
		return 0
	}

	for idx, candidate := range f.bestChain.chain {
		_, ok := ancestors[candidate.candidateHash]
		if !ok {
			return idx
		}
		delete(ancestors, candidate.candidateHash)
	}

	// this means that we found the entire chain in the ancestor set. There wont be
	// anything left to back.
	return len(f.bestChain.chain)
}

// earliestRelayParent returns the earliest relay parent a new candidate can have in order
// to be added to the chain right now. This is the relay parent of the latest candidate in
// the chain. The value returned may not be valid if we want to add a candidate pending
// availability, which may have a relay parent which is out of scope, special handling
// is needed in that case.
func (f *FragmentChain) earliestRelayParent() *inclusionemulator.RelayChainBlockInfo {
	if len(f.bestChain.chain) > 0 {
		lastCandidate := f.bestChain.chain[len(f.bestChain.chain)-1]
		info := f.scope.Ancestor(lastCandidate.relayParent())
		if info != nil {
			return info
		}

		// if the relay parent is out of scope AND it is in the chain
		// it must be a candidate pending availability
		pending := f.scope.GetPendingAvailability(lastCandidate.candidateHash)
		if pending == nil {
			return nil
		}

		return &pending.RelayParent
	}

	earliest := f.scope.EarliestRelayParent()
	return &earliest
}

// earliestRelayParentPendingAvailability returns the earliest relay parent a potential
// candidate may have for it to ever be added to the chain. This is the relay parent of
// the last candidate pending availability or the earliest relay parent in scope.
func (f *FragmentChain) earliestRelayParentPendingAvailability() *inclusionemulator.RelayChainBlockInfo {
	for i := len(f.bestChain.chain) - 1; i >= 0; i-- {
		candidate := f.bestChain.chain[i]
		if pending := f.scope.GetPendingAvailability(candidate.candidateHash); pending != nil {
			return &pending.RelayParent
		}
	}
	earliest := f.scope.EarliestRelayParent()
	return &earliest
}

// populateUnconnectedPotentialCandidates populates the unconnected potential candidate storage
// starting from a previous storage
func (f *FragmentChain) populateUnconnectedPotentialCandidates(oldStorage *CandidateStorage) {
	for _, candidate := range oldStorage.byCandidateHash {
		// sanity check, all pending availability candidates should be already present
		// in the chain
		if pending := f.scope.GetPendingAvailability(candidate.candidateHash); pending != nil {
			continue
		}

		// we can just use the error to check if we can add
		// or not an entry since an error can legitimately
		// happen when pruning stale candidates.
		err := f.CanAddCandidateAsPotential(candidate)
		if err == nil {
			_ = f.unconnected.addCandidateEntry(candidate)
		}
	}
}

func (f *FragmentChain) checkPotential(candidate *CandidateEntry) error {
	relayParent := candidate.relayParent
	parentHeadHash := candidate.parentHeadDataHash

	// trivial 0-length cycle
	if candidate.outputHeadDataHash == parentHeadHash {
		return ErrZeroLengthCycle
	}

	// Check if the relay parent is in scope
	relayParentInfo := f.scope.Ancestor(relayParent)
	if relayParentInfo == nil {
		return ErrRelayParentNotInScope{
			relayParentA: relayParent,
			relayParentB: f.scope.EarliestRelayParent().Hash,
		}
	}

	// Check if the relay parent moved backwards from the latest candidate pending availability
	earliestRPOfPendingAvailability := f.earliestRelayParentPendingAvailability()
	if relayParentInfo.Number < earliestRPOfPendingAvailability.Number {
		return ErrRelayParentPrecedesCandidatePendingAvailability{
			relayParentA: relayParentInfo.Hash,
			relayParentB: earliestRPOfPendingAvailability.Hash,
		}
	}

	// If it's a fork with a backed candidate in the current chain
	if otherCandidateHash, ok := f.bestChain.byParentHead[parentHeadHash]; ok {
		if f.scope.GetPendingAvailability(otherCandidateHash) != nil {
			// Cannot accept a fork with a candidate pending availability
			return ErrForkWithCandidatePendingAvailability{candidateHash: otherCandidateHash}
		}

		// If the candidate is backed and in the current chain, accept only a candidate
		// according to the fork selection rule
		if forkSelectionRule(otherCandidateHash, candidate.candidateHash) == -1 {
			return ErrForkChoiceRule{candidateHash: otherCandidateHash}
		}
	}

	// Try seeing if the parent candidate is in the current chain or if it is the latest
	// included candidate. If so, get the constraints the candidate must satisfy
	var constraints *inclusionemulator.Constraints
	var maybeMinRelayParentNumber *uint

	requiredParentHash, err := f.scope.baseConstraints.RequiredParent.Hash()
	if err != nil {
		return fmt.Errorf("while hashing required parent: %w", err)
	}

	if parentCandidateHash, ok := f.bestChain.byOutputHead[parentHeadHash]; ok {
		var parentCandidate *FragmentNode

		for _, c := range f.bestChain.chain {
			if c.candidateHash == parentCandidateHash {
				parentCandidate = c
				break
			}
		}

		if parentCandidate == nil {
			return ErrParentCandidateNotFound
		}

		var err error
		constraints, err = f.scope.baseConstraints.ApplyModifications(parentCandidate.cumulativeModifications)
		if err != nil {
			return ErrComputeConstraints{modificationErr: err}
		}

		if ancestor := f.scope.Ancestor(parentCandidate.relayParent()); ancestor != nil {
			maybeMinRelayParentNumber = &ancestor.Number
		}
	} else if requiredParentHash == parentHeadHash {
		// It builds on the latest included candidate
		constraints = f.scope.baseConstraints.Clone()
	} else {
		// If the parent is not yet part of the chain, there's nothing else we can check for now
		return nil
	}

	// Check for cycles or invalid tree transitions
	if err := f.checkCyclesOrInvalidTree(candidate.outputHeadDataHash); err != nil {
		return err
	}

	// Check against constraints if we have a full concrete candidate
	_, err = inclusionemulator.CheckAgainstConstraints(
		relayParentInfo,
		constraints,
		candidate.candidate.Commitments,
		candidate.candidate.ValidationCodeHash,
		candidate.candidate.PersistedValidationData,
	)
	if err != nil {
		return ErrCheckAgainstConstraints{fragmentValidityErr: err}
	}

	if relayParentInfo.Number < constraints.MinRelayParentNumber {
		return ErrRelayParentMovedBackwards
	}

	if maybeMinRelayParentNumber != nil && relayParentInfo.Number < *maybeMinRelayParentNumber {
		return ErrRelayParentMovedBackwards
	}

	return nil
}

// trimUneligibleForks once the backable chain was populated, trim the forks generated by candidate
// hashes which are not present in the best chain. Fan this out into a full breadth-first search. If
// starting point is not nil then start the search from the candidates haing this parent head hash.
func (f *FragmentChain) trimUneligibleForks(storage *CandidateStorage, startingPoint *common.Hash) {
	type queueItem struct {
		hash         common.Hash
		hasPotential bool
	}

	queue := list.New()

	// start out with the candidates in the chain. They are all valid candidates.
	if startingPoint != nil {
		queue.PushBack(queueItem{hash: *startingPoint, hasPotential: true})
	} else {
		if len(f.bestChain.chain) == 0 {
			reqParentHeadHash, err := f.scope.baseConstraints.RequiredParent.Hash()
			if err != nil {
				panic(fmt.Sprintf("while hashing required parent: %s", err.Error()))
			}

			queue.PushBack(queueItem{hash: reqParentHeadHash, hasPotential: true})
		} else {
			for _, candidate := range f.bestChain.chain {
				queue.PushBack(queueItem{hash: candidate.parentHeadDataHash, hasPotential: true})
			}
		}
	}

	// to make sure that cycles dont make us loop forever, keep track
	// of the visited parent head hashes
	visited := map[common.Hash]struct{}{}

	for queue.Len() > 0 {
		// queue.PopFront()
		parent := queue.Remove(queue.Front()).(queueItem)
		visited[parent.hash] = struct{}{}

		children, ok := storage.byParentHead[parent.hash]
		if !ok {
			continue
		}

		// cannot remove while iterating so store them here temporarily
		var toRemove []parachaintypes.CandidateHash

		for childHash := range children {
			child, ok := storage.byCandidateHash[childHash]
			if !ok {
				continue
			}

			// already visited this child. either is a cycle or multipath that lead
			// to the same candidate. either way, stop this branch to avoid looping
			// forever
			if _, ok = visited[child.outputHeadDataHash]; ok {
				continue
			}

			// only keep a candidate if its full ancestry was already kept as potential
			// and this candidate itself has potential
			if parent.hasPotential && f.checkPotential(child) == nil {
				queue.PushBack(queueItem{hash: child.outputHeadDataHash, hasPotential: true})
			} else {
				// otherwise, remove this candidate and continue looping for its children
				// but mark the parent's potential as false. we only want to remove its children.
				toRemove = append(toRemove, childHash)
				queue.PushBack(queueItem{hash: child.outputHeadDataHash, hasPotential: false})
			}
		}

		for _, hash := range toRemove {
			storage.removeCandidate(hash)
		}
	}
}

type possibleChild struct {
	fragment           *inclusionemulator.Fragment
	candidateHash      parachaintypes.CandidateHash
	outputHeadDataHash common.Hash
	parentHeadDataHash common.Hash
}

// populateChain populates the fragment chain with candidates from the supplied `CandidateStorage`.
// Can be called by the `NewFragmentChain` or when backing a new candidate. When this is called
// it may cause the previous chain to be completely erased or it may add more than one candidate
func (f *FragmentChain) populateChain(storage *CandidateStorage) {
	var cumulativeModifications *inclusionemulator.ConstraintModifications
	if len(f.bestChain.chain) > 0 {
		lastCandidate := f.bestChain.chain[len(f.bestChain.chain)-1]
		cumulativeModifications = lastCandidate.cumulativeModifications.Clone()
	} else {
		cumulativeModifications = inclusionemulator.NewConstraintModificationsIdentity()
	}

	earliestRelayParent := f.earliestRelayParent()
	if earliestRelayParent == nil {
		return
	}

	for len(f.bestChain.chain) < int(f.scope.maxDepth) {
		childConstraints, err := f.scope.baseConstraints.ApplyModifications(cumulativeModifications)
		if err != nil {
			// TODO: include logger
			fmt.Println("failed to apply modifications:", err)
			break
		}

		requiredHeadHash, err := childConstraints.RequiredParent.Hash()
		if err != nil {
			panic(fmt.Sprintf("failed while hashing required parent: %s", err.Error()))
		}

		possibleChildren := make([]*possibleChild, 0)
		// select the few possible backed/backable children which can be added to the chain right now
		for candidateEntry := range storage.possibleBackedParaChildren(requiredHeadHash) {
			// only select a candidate if:
			// 1. it does not introduce a fork or a cycle
			// 2. parent hash is correct
			// 3. relay parent does not move backwards
			// 4. all non-pending-availability candidates have relay-parent in the scope
			// 5. candidate outputs fulfill constraints

			var relayParent *inclusionemulator.RelayChainBlockInfo
			var minRelayParent uint

			pending := f.scope.GetPendingAvailability(candidateEntry.candidateHash)
			if pending != nil {
				relayParent = &pending.RelayParent
				if len(f.bestChain.chain) == 0 {
					minRelayParent = pending.RelayParent.Number
				} else {
					minRelayParent = earliestRelayParent.Number
				}
			} else {
				info := f.scope.Ancestor(candidateEntry.relayParent)
				if info == nil {
					continue
				}

				relayParent = info
				minRelayParent = earliestRelayParent.Number
			}

			if err := f.checkCyclesOrInvalidTree(candidateEntry.outputHeadDataHash); err != nil {
				fmt.Println("checking cycle or invalid tree:", err)
				continue
			}

			// require: candidates dont move backwards and only pending availability
			// candidates can be out-of-scope.
			//
			// earliest relay parent can be before the
			if relayParent.Number < minRelayParent {
				// relay parent moved backwards
				continue
			}

			// don't add candidates if they're already present in the chain
			// this can never happen, as candidates can only be duplicated
			// if there's a cycle and we shouldnt have allowed for a cycle
			// to be chained
			if f.bestChain.Contains(candidateEntry.candidateHash) {
				continue
			}

			constraints := childConstraints.Clone()
			if pending != nil {
				// overwrite for candidates pending availability as a special-case
				constraints.MinRelayParentNumber = pending.RelayParent.Number
			}

			fragment, err := inclusionemulator.NewFragment(relayParent, constraints, candidateEntry.candidate)
			if err != nil {
				fmt.Println("failed to create fragment:", err)
				continue
			}

			possibleChildren = append(possibleChildren, &possibleChild{
				fragment:           fragment,
				candidateHash:      candidateEntry.candidateHash,
				outputHeadDataHash: candidateEntry.outputHeadDataHash,
				parentHeadDataHash: candidateEntry.parentHeadDataHash,
			})
		}

		if len(possibleChildren) == 0 {
			break
		}

		// choose the best candidate
		bestCandidate := slices.MinFunc(possibleChildren, func(fst, snd *possibleChild) int {
			// always pick a candidate pending availability as best.
			if f.scope.GetPendingAvailability(fst.candidateHash) != nil {
				return -1
			} else if f.scope.GetPendingAvailability(snd.candidateHash) != nil {
				return 1
			} else {
				return forkSelectionRule(fst.candidateHash, snd.candidateHash)
			}
		})

		// remove the candidate from storage
		storage.removeCandidate(bestCandidate.candidateHash)

		// update the cumulative constraint modifications
		cumulativeModifications.Stack(bestCandidate.fragment.ConstraintModifications())

		// update the earliest relay parent
		earliestRelayParent = &inclusionemulator.RelayChainBlockInfo{
			Hash:        bestCandidate.fragment.RelayParent().Hash,
			Number:      bestCandidate.fragment.RelayParent().Number,
			StorageRoot: bestCandidate.fragment.RelayParent().StorageRoot,
		}

		node := &FragmentNode{
			fragment:                bestCandidate.fragment,
			candidateHash:           bestCandidate.candidateHash,
			parentHeadDataHash:      bestCandidate.parentHeadDataHash,
			outputHeadDataHash:      bestCandidate.outputHeadDataHash,
			cumulativeModifications: cumulativeModifications.Clone(),
		}

		// add the candidate to the chain now
		f.bestChain.Push(node)
	}
}

// checkCyclesOrInvalidTree checks whether a candidate outputting this head data would
// introduce a cycle or multiple paths to the same state. Trivial 0-length cycles are
// checked  in `NewCandidateEntry`.
func (f *FragmentChain) checkCyclesOrInvalidTree(outputHeadDataHash common.Hash) error {
	// this should catch a cycle where this candidate would point back to the parent
	// of some candidate in the chain
	_, ok := f.bestChain.byParentHead[outputHeadDataHash]
	if ok {
		return ErrCycle
	}

	// multiple paths to the same state, which cannot happen for a chain
	_, ok = f.bestChain.byOutputHead[outputHeadDataHash]
	if ok {
		return ErrMultiplePaths
	}

	return nil
}

// revertTo reverts the best backable chain so that the last candidate will be one outputting the given
// `parent_head_hash`. If the `parent_head_hash` is exactly the required parent of the base
// constraints (builds on the latest included candidate), revert the entire chain.
// Return false if we couldn't find the parent head hash
func (f *FragmentChain) revertTo(parentHeadDataHash common.Hash) bool {
	var removedItems []*FragmentNode = nil

	requiredParentHash, err := f.scope.baseConstraints.RequiredParent.Hash()
	if err != nil {
		panic(fmt.Sprintf("failed while hashing required parent: %s", err.Error()))
	}

	if requiredParentHash == parentHeadDataHash {
		removedItems = f.bestChain.Clear()
	}

	if _, ok := f.bestChain.byOutputHead[parentHeadDataHash]; removedItems == nil && ok {
		removedItems = f.bestChain.RevertToParentHash(parentHeadDataHash)
	}

	if removedItems == nil {
		return false
	}

	// Even if it's empty, we need to return true, because we'll be able to add a new candidate
	// to the chain.
	for _, node := range removedItems {
		_ = f.unconnected.addCandidateEntry(NewCandidateEntryFromFragment(node))
	}

	return true
}
