// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package parachain

import (
	"testing"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/require"
)

var hash = common.Hash{0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
	0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA}

func TestEncodeApprovalDistributionMessageAssignmentModulo(t *testing.T) {
	approvalDistributionMessage := NewApprovalDistributionMessageVDT()
	// expected encoding is generated by running rust test code:
	// fn try_msg_assignments_encode() {
	//	let hash = Hash::repeat_byte(0xAA);
	//
	//	let validator_index = ValidatorIndex(1);
	//	let cert = fake_assignment_cert(hash, validator_index);
	//	let assignments = vec![(cert.clone(), 4u32)];
	//	let msg = protocol_v1::ApprovalDistributionMessage::Assignments(assignments.clone());
	//
	//	let emsg = msg.encode();
	//	println!("encode: {:?}", emsg);
	//}
	expectedEncoding := []byte{0, 4, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 1, 0, 0, 0, 0, 2, 0, 0, 0, 1,
		2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
		32, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29,
		30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57,
		58, 59, 60, 61, 62, 63, 64, 4, 0, 0, 0}

	approvalDistributionMessage.Set(Assignments{
		Assignments: []Assignment{{
			IndirectAssignmentCert: fakeAssignmentCert(hash, ValidatorIndex(1), false),
			CandidateIndex:         4,
		}},
	})

	encodedMessage, err := scale.Marshal(approvalDistributionMessage)
	require.NoError(t, err)

	require.Equal(t, expectedEncoding, encodedMessage)

	approvalDistributionMessageDecodedTest := NewApprovalDistributionMessageVDT()
	scale.Unmarshal(encodedMessage, &approvalDistributionMessageDecodedTest)
	require.Equal(t, approvalDistributionMessage, approvalDistributionMessageDecodedTest)
}

func TestEncodeApprovalDistributionMessageAssignmentDelay(t *testing.T) {
	approvalDistributionMessage := NewApprovalDistributionMessageVDT()

	expectedEncoding := []byte{0, 4, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 2, 0, 0, 0, 1, 1, 0, 0, 0, 1, 2,
		3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
		31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58,
		59, 60, 61, 62, 63, 64, 2, 0, 0, 0}

	approvalDistributionMessage.Set(Assignments{
		Assignments: []Assignment{{
			IndirectAssignmentCert: fakeAssignmentCert(hash, ValidatorIndex(2), true),
			CandidateIndex:         2,
		}},
	})

	encodedMessage, err := scale.Marshal(approvalDistributionMessage)
	require.NoError(t, err)

	require.Equal(t, expectedEncoding, encodedMessage)
}

func TestEncodeAssignmentCertKindModulo(t *testing.T) {
	assignmentCertKind := NewAssignmentCertKindVDT()
	assignmentCertKind.Set(RelayVRFModulo{Sample: 4})
	expectedEncoding := []byte{0, 4, 0, 0, 0}
	encodedAssignmentCertKind, err := scale.Marshal(assignmentCertKind)
	require.NoError(t, err)
	require.Equal(t, expectedEncoding, encodedAssignmentCertKind)

	assignmentCertTest := NewAssignmentCertKindVDT()
	err = scale.Unmarshal(encodedAssignmentCertKind, &assignmentCertTest)
	require.NoError(t, err)
	require.Equal(t, assignmentCertKind, assignmentCertTest)
}

func TestEncodeAssignmentCertKindDelay(t *testing.T) {
	assignmentCertKind := NewAssignmentCertKindVDT()
	assignmentCertKind.Set(RelayVRFDelay{CoreIndex: 5})
	expectedEncoding := []byte{1, 5, 0, 0, 0}
	encodedAssignmentCertKind, err := scale.Marshal(assignmentCertKind)
	require.NoError(t, err)
	require.Equal(t, expectedEncoding, encodedAssignmentCertKind)

	assignmentCertTest := NewAssignmentCertKindVDT()
	err = scale.Unmarshal(encodedAssignmentCertKind, &assignmentCertTest)
	require.NoError(t, err)
	require.Equal(t, assignmentCertKind, assignmentCertTest)
}

func TestEncodeApprovalDistributionMessageApprovals(t *testing.T) {
	approvalDistributionMessage := NewApprovalDistributionMessageVDT()

	expectedEncoding := []byte{1, 4, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 2, 0, 0, 0, 3, 0, 0, 0, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	approvalDistributionMessage.Set(Approvals{
		Approvals: []IndirectSignedApprovalVote{{
			BlockHash:      hash,
			CandidateIndex: CandidateIndex(2),
			ValidatorIndex: ValidatorIndex(3),
			Signature: ValidatorSignature{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1},
		}},
	})

	encodedMessage, err := scale.Marshal(approvalDistributionMessage)
	require.NoError(t, err)
	require.Equal(t, expectedEncoding, encodedMessage)
}

func TestDecodeApprovalDistributionMessageAssignmentModulo(t *testing.T) {
	encoding := []byte{0, 4, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 2, 0, 0, 0, 0, 2, 0, 0, 0, 1, 2, 3, 4,
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 1, 2,
		3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32,
		33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
		61, 62, 63, 64, 4, 0, 0, 0}
	approvalDistributionMessage := NewApprovalDistributionMessageVDT()
	err := scale.Unmarshal(encoding, &approvalDistributionMessage)
	require.NoError(t, err)

	expectedApprovalDistributionMessage := NewApprovalDistributionMessageVDT()
	expectedApprovalDistributionMessage.Set(Assignments{
		Assignments: []Assignment{{
			IndirectAssignmentCert: fakeAssignmentCert(hash, ValidatorIndex(2), false),
			CandidateIndex:         4,
		}},
	})

	approvalValue, err := approvalDistributionMessage.Value()
	require.NoError(t, err)
	expectedValue, err := expectedApprovalDistributionMessage.Value()
	require.NoError(t, err)
	require.Equal(t, expectedValue, approvalValue)
}

func TestDecodeApprovalDistributionMessageApprovals(t *testing.T) {
	encoding := []byte{1, 4, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170,
		170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 170, 2, 0, 0, 0, 3, 0, 0, 0, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1}
	expectedApprovalDistributionMessage := NewApprovalDistributionMessageVDT()
	expectedApprovalDistributionMessage.Set(Approvals{
		Approvals: []IndirectSignedApprovalVote{{
			BlockHash:      hash,
			CandidateIndex: CandidateIndex(2),
			ValidatorIndex: ValidatorIndex(3),
			Signature: ValidatorSignature{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
				1, 1, 1, 1, 1, 1},
		}},
	})

	approvalDistributionMessage := NewApprovalDistributionMessageVDT()
	err := scale.Unmarshal(encoding, &approvalDistributionMessage)
	require.NoError(t, err)
	require.Equal(t, expectedApprovalDistributionMessage, approvalDistributionMessage)
}

func fakeAssignmentCert(blockHash common.Hash, validator ValidatorIndex, useDelay bool) IndirectAssignmentCert {
	output := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
		27, 28, 29, 30, 31, 32}
	proof := [64]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26,
		27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54,
		55, 56, 57, 58, 59, 60, 61, 62, 63, 64}
	assignmentCertKind := NewAssignmentCertKindVDT()
	if useDelay {
		assignmentCertKind.Set(RelayVRFDelay{CoreIndex: 1})
	} else {
		assignmentCertKind.Set(RelayVRFModulo{Sample: 2})
	}

	return IndirectAssignmentCert{
		BlockHash: blockHash,
		Validator: validator,
		Cert: AssignmentCert{
			Kind: assignmentCertKind,
			Vrf: VrfSignature{
				Output: output,
				Proof:  proof,
			},
		},
	}
}
