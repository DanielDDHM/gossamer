package prospectiveparachains

import (
	"bytes"
	"context"
	"sync"
	"testing"

	fragmentchain "github.com/ChainSafe/gossamer/dot/parachain/prospective-parachains/fragment-chain"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	inclusionemulator "github.com/ChainSafe/gossamer/dot/parachain/util/inclusion-emulator"
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

// TestGetMinimumRelayParents ensures that getMinimumRelayParents
// processes the relay parent hash and correctly sends the output via the channel
func TestGetMinimumRelayParents(t *testing.T) {
	// Setup a mock View with active leaves and relay parent data

	mockRelayParent := inclusionemulator.RelayChainBlockInfo{
		Hash:   common.Hash([]byte("active_hash")),
		Number: 10,
	}

	ancestors := []inclusionemulator.RelayChainBlockInfo{
		{
			Hash:   common.Hash([]byte("active_hash_7")),
			Number: 9,
		},
		{
			Hash:   common.Hash([]byte("active_hash_8")),
			Number: 8,
		},
		{
			Hash:   common.Hash([]byte("active_hash_9")),
			Number: 7,
		},
	}

	baseConstraints := &inclusionemulator.Constraints{
		MinRelayParentNumber: 5,
	}

	mockScope, err := fragmentchain.NewScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, ancestors)
	assert.NoError(t, err)

	mockScope2, err := fragmentchain.NewScopeWithAncestors(mockRelayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	mockView := &View{
		activeLeaves: map[common.Hash]bool{
			common.BytesToHash([]byte("active_hash")): true,
		},
		PerRelayParent: map[common.Hash]*RelayParentData{
			common.BytesToHash([]byte("active_hash")): {
				fragmentChains: map[parachaintypes.ParaID]*fragmentchain.FragmentChain{
					parachaintypes.ParaID(1): fragmentchain.NewFragmentChain(mockScope, fragmentchain.NewCandidateStorage()),
					parachaintypes.ParaID(2): fragmentchain.NewFragmentChain(mockScope2, fragmentchain.NewCandidateStorage()),
				},
			},
		},
	}

	// Initialize ProspectiveParachains with the mock view
	pp := &ProspectiveParachains{
		View: mockView,
	}

	// Create a channel to capture the output
	sender := make(chan []ParaIDBlockNumber, 1)

	// Execute the method under test
	pp.getMinimumRelayParents(common.BytesToHash([]byte("active_hash")), sender)

	expected := []ParaIDBlockNumber{
		{
			ParaId:      1,
			BlockNumber: 7,
		},
		{
			ParaId:      2,
			BlockNumber: 10,
		},
	}
	// Validate the results
	result := <-sender
	assert.Len(t, result, 2)
	assert.Equal(t, expected, result)
}

// TestGetMinimumRelayParents_NoActiveLeaves ensures that getMinimumRelayParents
// correctly handles the case where there are no active leaves.
func TestGetMinimumRelayParents_NoActiveLeaves(t *testing.T) {
	mockView := &View{
		activeLeaves:   map[common.Hash]bool{},
		PerRelayParent: map[common.Hash]*RelayParentData{},
	}

	// Initialize ProspectiveParachains with the mock view
	pp := &ProspectiveParachains{
		View: mockView,
	}

	// Create a channel to capture the output
	sender := make(chan []ParaIDBlockNumber, 1)

	// Execute the method under test
	pp.getMinimumRelayParents(common.BytesToHash([]byte("active_hash")), sender)
	// Validate the results
	result := <-sender
	assert.Empty(t, result, "Expected result to be empty when no active leaves are present")
}

func TestProspectiveParachains_HandleMinimumRelayParents(t *testing.T) {
	candidateRelayParent := common.Hash{0x01}
	paraId := parachaintypes.ParaID(1)
	parentHead := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x01}, 32),
	}
	headData := parachaintypes.HeadData{
		Data: bytes.Repeat([]byte{0x02}, 32),
	}
	validationCodeHash := parachaintypes.ValidationCodeHash{0x01}
	candidateRelayParentNumber := uint32(0)

	candidate := makeCandidate(
		candidateRelayParent,
		candidateRelayParentNumber,
		paraId,
		parentHead,
		headData,
		validationCodeHash,
	)

	subsystemToOverseer := make(chan any)
	overseerToSubsystem := make(chan any)

	prospectiveParachains := NewProspectiveParachains(subsystemToOverseer)

	relayParent := inclusionemulator.RelayChainBlockInfo{
		Hash:        candidateRelayParent,
		Number:      0,
		StorageRoot: common.Hash{0x00},
	}

	baseConstraints := &inclusionemulator.Constraints{
		RequiredParent:       parachaintypes.HeadData{Data: bytes.Repeat([]byte{0x01}, 32)},
		MinRelayParentNumber: 0,
		ValidationCodeHash:   validationCodeHash,
		MaxPoVSize:           1000000,
	}

	scope, err := fragmentchain.NewScopeWithAncestors(relayParent, baseConstraints, nil, 10, nil)
	assert.NoError(t, err)

	prospectiveParachains.View.PerRelayParent[candidateRelayParent] = &RelayParentData{
		fragmentChains: map[parachaintypes.ParaID]*fragmentchain.FragmentChain{
			paraId: fragmentchain.NewFragmentChain(scope, fragmentchain.NewCandidateStorage()),
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	var wg sync.WaitGroup

	// Run the subsystem
	wg.Add(1)
	go func() {
		defer wg.Done()
		prospectiveParachains.Run(ctx, overseerToSubsystem)
	}()

	sender := make(chan []ParaIDBlockNumber, 1)
	go func() {
		overseerToSubsystem <- GetMinimumRelayParents{
			RelayChainBlockHash: candidateRelayParent,
			Sender:              sender,
		}
	}()

	result := <-sender
	assert.Len(t, result, 1, "Expected one ParaIDBlockNumber in the result")
	assert.Equal(t, paraId, result[0].ParaId, "ParaId mismatch in the result")
	assert.Equal(t, uint32(0), result[0].BlockNumber, "BlockNumber mismatch in the result")

	_, err = candidate.Hash()
	assert.NoError(t, err)

	// Ensure subsystem stops gracefully
	cancel()
	wg.Wait()
}
