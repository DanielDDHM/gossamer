package prospectiveparachains

import (
	"bytes"
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

const MaxPoVSize = 1_000_000

func dummyPVD(parentHead parachaintypes.HeadData, relayParentNumber uint32) parachaintypes.PersistedValidationData {
	return parachaintypes.PersistedValidationData{
		ParentHead:             parentHead,
		RelayParentNumber:      relayParentNumber,
		RelayParentStorageRoot: common.EmptyHash,
		MaxPovSize:             MaxPoVSize,
	}
}

func dummyCandidateReceiptBadSig(
	relayParentHash common.Hash,
	commitments *common.Hash,
) parachaintypes.CandidateReceipt {
	var commitmentsHash common.Hash

	if commitments != nil {
		commitmentsHash = *commitments
	} else {
		commitmentsHash = common.EmptyHash
	}

	descriptor := parachaintypes.CandidateDescriptor{
		ParaID:                      parachaintypes.ParaID(0),
		RelayParent:                 relayParentHash,
		Collator:                    parachaintypes.CollatorID{},
		PovHash:                     common.EmptyHash,
		ErasureRoot:                 common.EmptyHash,
		Signature:                   parachaintypes.CollatorSignature{},
		ParaHead:                    common.EmptyHash,
		ValidationCodeHash:          parachaintypes.ValidationCodeHash{},
		PersistedValidationDataHash: common.EmptyHash,
	}

	return parachaintypes.CandidateReceipt{
		CommitmentsHash: commitmentsHash,
		Descriptor:      descriptor,
	}
}

func makeCandidate(
	relayParent common.Hash,
	relayParentNumber uint32,
	paraID parachaintypes.ParaID,
	parentHead parachaintypes.HeadData,
	headData parachaintypes.HeadData,
	validationCodeHash parachaintypes.ValidationCodeHash,
) parachaintypes.CommittedCandidateReceipt {
	pvd := dummyPVD(parentHead, relayParentNumber)

	commitments := parachaintypes.CandidateCommitments{
		HeadData:                  headData,
		HorizontalMessages:        []parachaintypes.OutboundHrmpMessage{},
		UpwardMessages:            []parachaintypes.UpwardMessage{},
		NewValidationCode:         nil,
		ProcessedDownwardMessages: 0,
		HrmpWatermark:             relayParentNumber,
	}

	commitmentsHash := commitments.Hash()

	candidate := dummyCandidateReceiptBadSig(relayParent, &commitmentsHash)
	candidate.CommitmentsHash = commitments.Hash()
	candidate.Descriptor.ParaID = paraID

	pvdh, err := pvd.Hash()

	if err != nil {
		panic(err)
	}

	candidate.Descriptor.PersistedValidationDataHash = pvdh
	candidate.Descriptor.ValidationCodeHash = validationCodeHash

	result := parachaintypes.CommittedCandidateReceipt{
		Descriptor:  candidate.Descriptor,
		Commitments: commitments,
	}

	return result
}

func TestGetBackableCandidates(t *testing.T) {
	candidateRelayParent1 := common.Hash{0x01}
	candidateRelayParent2 := common.Hash{0x02}
	candidateRelayParent3 := common.Hash{0x03}

	paraId := parachaintypes.ParaID(1)

	parentHead1 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x01}, 32)}
	parentHead2 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x02}, 32)}
	parentHead3 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x03}, 32)}

	headData1 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x01}, 32)}
	headData2 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x02}, 32)}
	headData3 := parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x03}, 32)}

	validationCodeHash := parachaintypes.ValidationCodeHash{0x01}
	candidateRelayParentNumber := uint32(1)

	candidate1 := makeCandidate(
		candidateRelayParent1,
		candidateRelayParentNumber,
		paraId,
		parentHead1,
		headData1,
		validationCodeHash,
	)

	candidate2 := makeCandidate(
		candidateRelayParent2,
		candidateRelayParentNumber,
		paraId,
		parentHead2,
		headData2,
		validationCodeHash,
	)

	candidate3 := makeCandidate(
		candidateRelayParent3,
		candidateRelayParentNumber,
		paraId,
		parentHead3,
		headData3,
		validationCodeHash,
	)

	mockRelayParent := relayChainBlockInfo{
		Hash:   candidateRelayParent1,
		Number: 10,
	}

	ancestors := []relayChainBlockInfo{
		{
			Hash:   candidateRelayParent2,
			Number: 9,
		},
		{
			Hash:   candidateRelayParent3,
			Number: 8,
		},
	}

	baseConstraints := &parachaintypes.Constraints{
		MinRelayParentNumber: 8,
	}

	mockScope, err := newScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, ancestors)
	assert.NoError(t, err)

	parentHash1, err := parentHead1.Hash()
	assert.NoError(t, err)

	outputHash1, err := headData1.Hash()
	assert.NoError(t, err)

	candidateStorage := newCandidateStorage()
	err = candidateStorage.addCandidateEntry(&candidateEntry{
		candidateHash:      parachaintypes.CandidateHash{Value: candidateRelayParent1},
		parentHeadDataHash: parentHash1,
		outputHeadDataHash: outputHash1,
		relayParent:        candidateRelayParent1,
		candidate: &prospectiveCandidate{
			Commitments:             candidate1.Commitments,
			PersistedValidationData: dummyPVD(parentHead1, 10),
			PoVHash:                 candidate1.Descriptor.PovHash,
			ValidationCodeHash:      validationCodeHash,
		},
		state: backed,
	})
	assert.NoError(t, err)

	parentHash2, err := parentHead2.Hash()
	assert.NoError(t, err)

	outputHash2, err := headData2.Hash()
	assert.NoError(t, err)

	err = candidateStorage.addCandidateEntry(&candidateEntry{
		candidateHash:      parachaintypes.CandidateHash{Value: candidateRelayParent2},
		parentHeadDataHash: parentHash2,
		outputHeadDataHash: outputHash2,
		relayParent:        candidateRelayParent2,
		candidate: &prospectiveCandidate{
			Commitments:             candidate2.Commitments,
			PersistedValidationData: dummyPVD(parentHead2, 9),
			PoVHash:                 candidate2.Descriptor.PovHash,
			ValidationCodeHash:      validationCodeHash,
		},
		state: backed,
	})
	assert.NoError(t, err)

	parentHash3, err := parentHead3.Hash()
	assert.NoError(t, err)

	outputHash3, err := headData3.Hash()
	assert.NoError(t, err)

	err = candidateStorage.addCandidateEntry(&candidateEntry{
		candidateHash:      parachaintypes.CandidateHash{Value: candidateRelayParent3},
		parentHeadDataHash: parentHash3,
		outputHeadDataHash: outputHash3,
		relayParent:        candidateRelayParent3,
		candidate: &prospectiveCandidate{
			Commitments:             candidate3.Commitments,
			PersistedValidationData: dummyPVD(parentHead3, 8),
			PoVHash:                 candidate3.Descriptor.PovHash,
			ValidationCodeHash:      validationCodeHash,
		},
		state: backed,
	})
	assert.NoError(t, err)

	type testCase struct {
		name           string
		view           *view
		msg            GetBackableCandidates
		expectedLength int
	}

	cases := []testCase{
		{
			name: "relay_parent_inactive",
			view: &view{
				activeLeaves:   map[common.Hash]bool{},
				perRelayParent: map[common.Hash]*relayParentData{},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "active_leaves_empty",
			view: &view{
				activeLeaves: map[common.Hash]bool{},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "no_candidates_found",
			view: &view{
				activeLeaves: map[common.Hash]bool{
					candidateRelayParent1: true,
				},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "not_found_parachain_based_on_paraid",
			view: &view{
				activeLeaves: map[common.Hash]bool{
					candidateRelayParent1: true,
				},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(3),
				RequestedQty:    1,
				Ancestors:       Ancestors{},
				Response:        make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 0,
		},
		{
			name: "candidates_found",
			view: &view{
				activeLeaves: map[common.Hash]bool{
					candidateRelayParent1: true,
				},
				perRelayParent: map[common.Hash]*relayParentData{
					candidateRelayParent1: {
						fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
							parachaintypes.ParaID(1): newFragmentChain(mockScope, candidateStorage),
						},
					},
				},
			},
			msg: GetBackableCandidates{
				RelayParentHash: candidateRelayParent1,
				ParaId:          parachaintypes.ParaID(1),
				RequestedQty:    2,
				Ancestors: Ancestors{
					parachaintypes.CandidateHash{Value: candidateRelayParent2}: struct{}{},
					parachaintypes.CandidateHash{Value: candidateRelayParent3}: struct{}{},
				},
				Response: make(chan []parachaintypes.CandidateHashAndRelayParent, 1),
			},
			expectedLength: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pp := &ProspectiveParachains{
				View: tc.view,
			}

			pp.getBackableCandidates(tc.msg)

			select {
			case result := <-tc.msg.Response:
				assert.Equal(t, tc.expectedLength, len(result), "Unexpected number of candidates")
			default:
				t.Fatal("No response received from getBackableCandidates")
			}
		})
	}
}
