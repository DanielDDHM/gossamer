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
		MinRelayParentNumber: 5,
	}

	mockScope, err := newScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, ancestors)
	assert.NoError(t, err)

	parentHash1, err := parentHead1.Hash()
	if err != nil {
		t.Fatalf("failed to hash parentHead2: %v", err)
	}

	outputHash1, err := headData1.Hash()
	if err != nil {
		t.Fatalf("failed to hash headData2: %v", err)
	}

	candidateStorage := newCandidateStorage()
	_ = candidateStorage.addCandidateEntry(&candidateEntry{
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

	parentHash2, err := parentHead2.Hash()
	if err != nil {
		t.Fatalf("failed to hash parentHead2: %v", err)
	}

	outputHash2, err := headData2.Hash()
	if err != nil {
		t.Fatalf("failed to hash headData2: %v", err)
	}

	_ = candidateStorage.addCandidateEntry(&candidateEntry{
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

	parentHash3, err := parentHead3.Hash()
	if err != nil {
		t.Fatalf("failed to hash parentHead2: %v", err)
	}

	outputHash3, err := headData3.Hash()
	if err != nil {
		t.Fatalf("failed to hash headData2: %v", err)
	}

	_ = candidateStorage.addCandidateEntry(&candidateEntry{
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

	mockView := &View{
		activeLeaves: map[common.Hash]bool{
			candidateRelayParent1: true,
		},
		perRelayParent: map[common.Hash]*relayParentData{
			candidateRelayParent1: {
				fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
					paraId: newFragmentChain(mockScope, candidateStorage),
				},
			},
		},
	}

	pp := &ProspectiveParachains{
		View: mockView,
	}

	responseChan := make(chan []parachaintypes.CandidateHashAndRelayParent, 1)

	mockAncestors := Ancestors{
		parachaintypes.CandidateHash{Value: candidateRelayParent2}: {},
		parachaintypes.CandidateHash{Value: candidateRelayParent3}: {},
	}

	msg := GetBackableCandidates{
		RelayParentHash: candidateRelayParent1,
		ParaId:          paraId,
		RequestedQty:    3,
		Ancestors:       mockAncestors,
		Response:        responseChan,
	}

	pp.getBackableCandidates(msg)

	select {
	case result := <-responseChan:
		assert.NotNil(t, result, "Result should not be nil")
		assert.Equal(t, 3, len(result), "Expected 3 candidates to be returned")

		expectedHashes := []parachaintypes.CandidateHash{
			{Value: candidateRelayParent1},
			{Value: candidateRelayParent2},
			{Value: candidateRelayParent3},
		}

		for i, candidate := range result {
			assert.Equal(t, expectedHashes[i], candidate.CandidateHash, "Candidate hash does not match")
			assert.Equal(t, expectedHashes[i].Value, candidate.CandidateRelayParent, "Relay parent does not match")
		}
	default:
		t.Fatal("No response received from getBackableCandidates")
	}
}

func TestGetBackableCandidates_NoCandidatesFound(t *testing.T) {
	candidateRelayParent := common.Hash{0x01}
	paraId := parachaintypes.ParaID(1)

	mockRelayParent := relayChainBlockInfo{
		Hash:   candidateRelayParent,
		Number: 10,
	}

	ancestors := []relayChainBlockInfo{}

	baseConstraints := &parachaintypes.Constraints{
		MinRelayParentNumber: 5,
	}

	mockScope, err := newScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, ancestors)
	assert.NoError(t, err)

	candidateStorage := newCandidateStorage()

	mockView := &View{
		activeLeaves: map[common.Hash]bool{
			candidateRelayParent: true,
		},
		perRelayParent: map[common.Hash]*relayParentData{
			candidateRelayParent: {
				fragmentChains: map[parachaintypes.ParaID]*fragmentChain{
					paraId: newFragmentChain(mockScope, candidateStorage),
				},
			},
		},
	}

	pp := &ProspectiveParachains{
		View: mockView,
	}

	responseChan := make(chan []parachaintypes.CandidateHashAndRelayParent, 1)

	mockAncestors := Ancestors{}

	msg := GetBackableCandidates{
		RelayParentHash: candidateRelayParent,
		ParaId:          paraId,
		RequestedQty:    3,
		Ancestors:       mockAncestors,
		Response:        responseChan,
	}

	pp.getBackableCandidates(msg)

	select {
	case result := <-responseChan:
		assert.NotNil(t, result, "Result should not be nil")
		assert.Equal(t, 0, len(result), "Expected 0 candidates to be returned")
	default:
		t.Fatal("No response received from getBackableCandidates")
	}
}
