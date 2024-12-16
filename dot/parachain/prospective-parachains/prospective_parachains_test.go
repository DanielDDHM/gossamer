package prospectiveparachains

import (
	"testing"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
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

func padTo32Bytes(input []byte) []byte {
	if len(input) > 32 {
		return input[:32]
	}
	return append(input, make([]byte, 32-len(input))...)
}

func TestGetBackableCandidates(t *testing.T) {
	// TODO: Elaborate testcases
}
