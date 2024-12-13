package prospectiveparachains

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/stretchr/testify/assert"
)

type MockFragmentChain struct {
	fragmentChain
}

func (m *MockFragmentChain) FindBackableChain(ancestors Ancestors, qty uint32) []parachaintypes.CandidateHashAndRelayParent {
	return []parachaintypes.CandidateHashAndRelayParent{
		{
			CandidateHash:        parachaintypes.CandidateHash{Value: common.Hash{0x10}},
			CandidateRelayParent: common.Hash{0x20},
		},
		{
			CandidateHash:        parachaintypes.CandidateHash{Value: common.Hash{0x11}},
			CandidateRelayParent: common.Hash{0x21},
		},
	}
}

func TestGetBackableCandidates(t *testing.T) {
	relayParentHash := common.Hash{0x01}
	paraId := parachaintypes.ParaID(1)
	requestedQty := uint32(2)
	ancestors := Ancestors{
		parachaintypes.CandidateHash{Value: common.Hash{0x02}}: {},
		parachaintypes.CandidateHash{Value: common.Hash{0x03}}: {},
	}
	responseChan := make(chan []parachaintypes.CandidateHashAndRelayParent, 1)

	mockFragmentChain := &MockFragmentChain{}

	pp := &ProspectiveParachains{
		View: &View{
			ActiveLeaves: map[common.Hash]bool{
				relayParentHash: true,
			},
			PerRelayParent: map[common.Hash]*RelayParentData{
				relayParentHash: {
					FragmentChains: map[parachaintypes.ParaID]*fragmentChain{
						paraId: &mockFragmentChain.fragmentChain,
					},
				},
			},
		},
	}

	// Create the test message
	msg := GetBackableCandidates{
		RelayParentHash: relayParentHash,
		ParaId:          paraId,
		RequestedQty:    requestedQty,
		Ancestors:       ancestors,
		Response:        responseChan,
	}

	// Run the method in a goroutine
	go pp.getBackableCandidates(msg)

	// Collect the response
	result := <-responseChan

	// Assertions
	assert.NotNil(t, result, "Result should not be nil")
	assert.Len(t, result, 2, "Should return 2 backable candidates")
	assert.Equal(t, result[0].CandidateHash.Value, common.Hash{0x10}, "First candidate hash mismatch")
	assert.Equal(t, result[0].CandidateRelayParent, common.Hash{0x20}, "First candidate relay parent mismatch")
	assert.Equal(t, result[1].CandidateHash.Value, common.Hash{0x11}, "Second candidate hash mismatch")
	assert.Equal(t, result[1].CandidateRelayParent, common.Hash{0x21}, "Second candidate relay parent mismatch")
}
