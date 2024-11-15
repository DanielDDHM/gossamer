package fragmentchain

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	inclusionemulator "github.com/ChainSafe/gossamer/dot/parachain/util/inclusion-emulator"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/btree"
)

func TestCandidateStorage_RemoveCandidate(t *testing.T) {
	storage := &CandidateStorage{
		byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
		byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
		byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
	}

	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
	parentHeadHash := common.Hash{4, 5, 6}
	outputHeadHash := common.Hash{7, 8, 9}

	entry := &CandidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parentHeadHash,
		outputHeadDataHash: outputHeadHash,
		state:              Backed,
	}

	storage.byCandidateHash[candidateHash] = entry
	storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}
	storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}

	storage.removeCandidate(candidateHash)

	_, exists := storage.byCandidateHash[candidateHash]
	assert.False(t, exists, "candidate should be removed from byCandidateHash")

	_, exists = storage.byParentHead[parentHeadHash]
	assert.False(t, exists, "candidate should be removed from byParentHead")

	_, exists = storage.byOutputHead[outputHeadHash]
	assert.False(t, exists, "candidate should be removed from byOutputHead")
}

func TestCandidateStorage_MarkBacked(t *testing.T) {
	storage := &CandidateStorage{
		byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
		byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
		byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
	}

	candidateHash := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
	parentHeadHash := common.Hash{4, 5, 6}
	outputHeadHash := common.Hash{7, 8, 9}

	entry := &CandidateEntry{
		candidateHash:      candidateHash,
		parentHeadDataHash: parentHeadHash,
		outputHeadDataHash: outputHeadHash,
		state:              Seconded,
	}

	storage.byCandidateHash[candidateHash] = entry
	storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}
	storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}

	storage.markBacked(candidateHash)

	assert.Equal(t, Backed, entry.state, "candidate state should be marked as backed")
}

func TestCandidateStorage_HeadDataByHash(t *testing.T) {
	tests := map[string]struct {
		setup    func() *CandidateStorage
		hash     common.Hash
		expected *parachaintypes.HeadData
	}{
		"find_head_data_of_first_candidate_using_output_head_data_hash": {
			setup: func() *CandidateStorage {
				storage := &CandidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
				}

				candidateHash := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
				parentHeadHash := common.Hash{4, 5, 6}
				outputHeadHash := common.Hash{7, 8, 9}
				headData := parachaintypes.HeadData{Data: []byte{10, 11, 12}}

				entry := &CandidateEntry{
					candidateHash:      candidateHash,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash,
					candidate: inclusionemulator.ProspectiveCandidate{
						Commitments: parachaintypes.CandidateCommitments{
							HeadData: headData,
						},
					},
				}

				storage.byCandidateHash[candidateHash] = entry
				storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}
				storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}

				return storage
			},
			hash:     common.Hash{7, 8, 9},
			expected: &parachaintypes.HeadData{Data: []byte{10, 11, 12}},
		},
		"find_head_data_using_parent_head_data_hash_from_second_candidate": {
			setup: func() *CandidateStorage {
				storage := &CandidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
				}

				candidateHash := parachaintypes.CandidateHash{Value: common.Hash{13, 14, 15}}
				parentHeadHash := common.Hash{16, 17, 18}
				outputHeadHash := common.Hash{19, 20, 21}
				headData := parachaintypes.HeadData{Data: []byte{22, 23, 24}}

				entry := &CandidateEntry{
					candidateHash:      candidateHash,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash,
					candidate: inclusionemulator.ProspectiveCandidate{
						PersistedValidationData: parachaintypes.PersistedValidationData{
							ParentHead: headData,
						},
					},
				}

				storage.byCandidateHash[candidateHash] = entry
				storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}
				storage.byOutputHead[outputHeadHash] = map[parachaintypes.CandidateHash]any{candidateHash: struct{}{}}

				return storage
			},
			hash:     common.Hash{16, 17, 18},
			expected: &parachaintypes.HeadData{Data: []byte{22, 23, 24}},
		},
		"use_nonexistent_hash_and_should_get_nil": {
			setup: func() *CandidateStorage {
				storage := &CandidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
				}
				return storage
			},
			hash:     common.Hash{99, 99, 99},
			expected: nil,
		},
		"insert_0_candidates_and_try_to_find_but_should_get_nil": {
			setup: func() *CandidateStorage {
				return &CandidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
				}
			},
			hash:     common.Hash{7, 8, 9},
			expected: nil,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			storage := tt.setup()
			result := storage.headDataByHash(tt.hash)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCandidateStorage_PossibleBackedParaChildren(t *testing.T) {
	tests := map[string]struct {
		setup    func() *CandidateStorage
		hash     common.Hash
		expected []*CandidateEntry
	}{
		"insert_2_candidates_for_same_parent_one_seconded_one_backed": {
			setup: func() *CandidateStorage {
				storage := &CandidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
				}

				candidateHash1 := parachaintypes.CandidateHash{Value: common.Hash{1, 2, 3}}
				parentHeadHash := common.Hash{4, 5, 6}
				outputHeadHash1 := common.Hash{7, 8, 9}

				candidateHash2 := parachaintypes.CandidateHash{Value: common.Hash{10, 11, 12}}
				outputHeadHash2 := common.Hash{13, 14, 15}

				entry1 := &CandidateEntry{
					candidateHash:      candidateHash1,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash1,
					state:              Seconded,
				}

				entry2 := &CandidateEntry{
					candidateHash:      candidateHash2,
					parentHeadDataHash: parentHeadHash,
					outputHeadDataHash: outputHeadHash2,
					state:              Backed,
				}

				storage.byCandidateHash[candidateHash1] = entry1
				storage.byCandidateHash[candidateHash2] = entry2
				storage.byParentHead[parentHeadHash] = map[parachaintypes.CandidateHash]any{
					candidateHash1: struct{}{},
					candidateHash2: struct{}{},
				}

				return storage
			},
			hash:     common.Hash{4, 5, 6},
			expected: []*CandidateEntry{{candidateHash: parachaintypes.CandidateHash{Value: common.Hash{10, 11, 12}}, parentHeadDataHash: common.Hash{4, 5, 6}, outputHeadDataHash: common.Hash{13, 14, 15}, state: Backed}},
		},
		"insert_nothing_and_call_function_should_return_nothing": {
			setup: func() *CandidateStorage {
				return &CandidateStorage{
					byParentHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byOutputHead:    make(map[common.Hash]map[parachaintypes.CandidateHash]any),
					byCandidateHash: make(map[parachaintypes.CandidateHash]*CandidateEntry),
				}
			},
			hash:     common.Hash{4, 5, 6},
			expected: nil,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			storage := tt.setup()
			var result []*CandidateEntry
			for entry := range storage.possibleBackedParaChildren(tt.hash) {
				result = append(result, entry)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEarliestRelayParent(t *testing.T) {
	tests := map[string]struct {
		setup  func() *Scope
		expect inclusionemulator.RelayChainBlockInfo
	}{
		"returns from ancestors": {
			setup: func() *Scope {
				relayParent := inclusionemulator.RelayChainBlockInfo{
					Hash:   common.Hash{0x01},
					Number: 10,
				}
				baseConstraints := &inclusionemulator.Constraints{
					MinRelayParentNumber: 5,
				}
				ancestor := inclusionemulator.RelayChainBlockInfo{
					Hash:   common.Hash{0x02},
					Number: 9,
				}
				ancestorsMap := btree.NewMap[uint, inclusionemulator.RelayChainBlockInfo](100)
				ancestorsMap.Set(ancestor.Number, ancestor)
				return &Scope{
					relayParent:     relayParent,
					baseConstraints: baseConstraints,
					ancestors:       ancestorsMap,
				}
			},
			expect: inclusionemulator.RelayChainBlockInfo{
				Hash:   common.Hash{0x02},
				Number: 9,
			},
		},
		"returns relayParent": {
			setup: func() *Scope {
				relayParent := inclusionemulator.RelayChainBlockInfo{
					Hash:   common.Hash{0x01},
					Number: 10,
				}
				baseConstraints := &inclusionemulator.Constraints{
					MinRelayParentNumber: 5,
				}
				return &Scope{
					relayParent:     relayParent,
					baseConstraints: baseConstraints,
					ancestors:       btree.NewMap[uint, inclusionemulator.RelayChainBlockInfo](100),
				}
			},
			expect: inclusionemulator.RelayChainBlockInfo{
				Hash:   common.Hash{0x01},
				Number: 10,
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			scope := tt.setup()
			result := scope.EarliestRelayParent()
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestBackedChain_RevertToParentHash(t *testing.T) {
	tests := map[string]struct {
		setup                    func() *BackedChain
		hash                     common.Hash
		expectedChainSize        int
		expectedRemovedFragments int
	}{
		"revert_to_parent_at_pos_2": {
			setup: func() *BackedChain {
				chain := &BackedChain{
					chain:        make([]*FragmentNode, 0),
					byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
					byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
					candidates:   make(map[parachaintypes.CandidateHash]struct{}),
				}

				for i := 0; i < 5; i++ {
					node := &FragmentNode{
						candidateHash:           parachaintypes.CandidateHash{Value: common.Hash{byte(i)}},
						parentHeadDataHash:      common.Hash{byte(i)},
						outputHeadDataHash:      common.Hash{byte(i + 1)},
						cumulativeModifications: &inclusionemulator.ConstraintModifications{},
					}
					chain.Push(node)
				}
				return chain
			},
			hash:                     common.Hash{3},
			expectedChainSize:        3,
			expectedRemovedFragments: 2,
		},
		"revert_to_parent_at_pos_0": {
			setup: func() *BackedChain {
				chain := &BackedChain{
					chain:        make([]*FragmentNode, 0),
					byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
					byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
					candidates:   make(map[parachaintypes.CandidateHash]struct{}),
				}

				for i := 0; i < 2; i++ {
					node := &FragmentNode{
						candidateHash:           parachaintypes.CandidateHash{Value: common.Hash{byte(i)}},
						parentHeadDataHash:      common.Hash{byte(i)},
						outputHeadDataHash:      common.Hash{byte(i + 1)},
						cumulativeModifications: &inclusionemulator.ConstraintModifications{},
					}
					chain.Push(node)
				}
				return chain
			},
			hash:                     common.Hash{1},
			expectedChainSize:        1,
			expectedRemovedFragments: 1,
		},
		"no_node_removed": {
			setup: func() *BackedChain {
				chain := &BackedChain{
					chain:        make([]*FragmentNode, 0),
					byParentHead: make(map[common.Hash]parachaintypes.CandidateHash),
					byOutputHead: make(map[common.Hash]parachaintypes.CandidateHash),
					candidates:   make(map[parachaintypes.CandidateHash]struct{}),
				}

				for i := 0; i < 3; i++ {
					node := &FragmentNode{
						candidateHash:           parachaintypes.CandidateHash{Value: common.Hash{byte(i)}},
						parentHeadDataHash:      common.Hash{byte(i)},
						outputHeadDataHash:      common.Hash{byte(i + 1)},
						cumulativeModifications: &inclusionemulator.ConstraintModifications{},
					}
					chain.Push(node)
				}
				return chain
			},
			hash:                     common.Hash{99}, // Non-existent hash
			expectedChainSize:        3,
			expectedRemovedFragments: 0,
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			backedChain := tt.setup()
			removedNodes := backedChain.RevertToParentHash(tt.hash)

			// Check the number of removed nodes
			assert.Equal(t, tt.expectedRemovedFragments, len(removedNodes))

			// Check the properties of the chain
			assert.Equal(t, tt.expectedChainSize, len(backedChain.chain))
			assert.Equal(t, tt.expectedChainSize, len(backedChain.byParentHead))
			assert.Equal(t, tt.expectedChainSize, len(backedChain.byOutputHead))
			assert.Equal(t, tt.expectedChainSize, len(backedChain.candidates))

			// Check that the remaining nodes are correct
			for i := 0; i < len(backedChain.chain); i++ {
				assert.Contains(t, backedChain.byParentHead, common.Hash{byte(i)})
				assert.Contains(t, backedChain.byOutputHead, common.Hash{byte(i + 1)})
				assert.Contains(t, backedChain.candidates, parachaintypes.CandidateHash{Value: common.Hash{byte(i)}})
			}
		})
	}
}
