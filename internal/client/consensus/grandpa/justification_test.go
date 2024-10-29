// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

import (
	"github.com/ChainSafe/gossamer/lib/common"
	"reflect"
	"testing"

	primitives "github.com/ChainSafe/gossamer/internal/primitives/consensus/grandpa"
	ced25519 "github.com/ChainSafe/gossamer/internal/primitives/core/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/keyring/ed25519"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime"
	"github.com/ChainSafe/gossamer/internal/primitives/runtime/generic"
	grandpa "github.com/ChainSafe/gossamer/pkg/finality-grandpa"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makePrecommit(t *testing.T,
	targetHash string,
	targetNumber uint64,
	round uint64, //nolint:unparam
	setID uint64,
	voter ed25519.Keyring,
) grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID] {
	t.Helper()

	precommit := grandpa.Precommit[hash.H256, uint64]{
		TargetHash:   hash.H256(targetHash),
		TargetNumber: targetNumber,
	}
	msg := grandpa.NewMessage(precommit)
	encoded := primitives.NewLocalizedPayload(primitives.RoundNumber(round), primitives.SetID(setID), msg)
	signature := voter.Sign(encoded)

	return grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]{
		Precommit: grandpa.Precommit[hash.H256, uint64]{
			TargetHash:   hash.H256(targetHash),
			TargetNumber: targetNumber,
		},
		Signature: signature,
		ID:        voter.Pair().Public().(ced25519.Public),
	}
}

func TestJustificationEncoding(t *testing.T) {
	var hashA = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll
	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommit := makePrecommit(t, hashA, 1, 1, 1, ed25519.Alice)
	precommits = append(precommits, precommit)

	expAncestries := make([]runtime.Header[uint64, hash.H256], 0)
	expAncestries = append(expAncestries, generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		100,
		hash.H256(""),
		hash.H256(""),
		hash.H256(hashA),
		runtime.Digest{}),
	)

	expected := primitives.GrandpaJustification[hash.H256, uint64]{
		Round: 2,
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash: hash.H256(
				"b\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", //nolint:lll
			),
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VoteAncestries: expAncestries,
	}

	encodedJustification, err := scale.Marshal(expected)
	require.NoError(t, err)

	justification, err := DecodeJustification[hash.H256, uint64, runtime.BlakeTwo256](encodedJustification)
	require.NoError(t, err)
	require.Equal(t, expected, justification.Justification)
}

func TestDecodeJustification_WndBlock2318339(t *testing.T) {
	encodedJustification := "0x0600000000000000a6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be4036" +
		"023002ca6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be403602300951efca205be6bf72de00ce74a" +
		"d6778453f470ef6d3404a61ca2bfbfe5bcc93a7620522d08803e5081f9e93bf0dc2e8143fe764201aedc13b3455a14cf77c50307d" +
		"952daf2d0e2616e5344a6cff989a3fcc5a79a5799198c15ff1c06c51a1280a6836b1530280181eabce3ded01e62283c7d6a64c6ed" +
		"bc9c789a28b10dc14be403602300c6291ffc574f4802d2a8c1da0581d64a3da41ae080495b0b5f746fc7fb8918a6444cad5103379" +
		"9a87a8a8270041392d3e4c0aaacceab1c18308b0481dd03b10e169da96fe889fe19f2e9463c4cb730b33473a561f0a15d5581ca7c" +
		"52362a252da6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be4036023002cb07761a88973058336b9a" +
		"6cb72535334d7ab6df78ec7f9238aa9573a25afc85fc3ec4483dd885f90b0f2cc9885c7001ce794a92df5804f458caaa11c6e8308" +
		"26a50082ec634a6c1bfac4d49d100555edf613df703d8a0ace3e4c95745ea699a6836b1530280181eabce3ded01e62283c7d6a64c" +
		"6edbc9c789a28b10dc14be4036023004eab479a55b3bd51982351f7eb37134d8374bb163055c708cce288da2d4971ff8dda5226a2" +
		"c77e88bd9fb1b83c814242538f656fb082176c16e1efcb6484510431dae797bbac0e27b901355cc90446369dcd15d8f965933b577" +
		"23ee389670d55a6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be403602300c0b0dbe81a2e565ad555" +
		"90c417db624031313d0fcf2dc3d9f5b09b755d0e899afa4739a2ce2aa7b6ca924cb3c54e886dba41bb0fe86e8f4222dd2b8106e2c" +
		"20a404b31a6663344c68c7e7c2a1cf4a767f53a525fe022f80da283d2c8bb13686da6836b1530280181eabce3ded01e62283c7d6a" +
		"64c6edbc9c789a28b10dc14be403602300a3c39a878944703d205b21957a2d6b06af1e18160beaf29b1d38f2f698eff5b727a2808" +
		"df428e58548278b312937b0679609cb561090e2827ad268ce6a4f51055af0167bdf2c135191f8ffb155c75f097b8573456879d1d8" +
		"9c51945090e645e7a6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be4036023000f5928de38552f67f" +
		"f5e62daf6c8af411450e2964ad67e32e061813ec47ec0e66f95b181644046aac8de316859a2e4adf3b900c05b171e20edd177169a" +
		"1e01095d05c538467ed259f2520b56f49ff58832ccc4408a69b28a12bb273f3b419f2ca6836b1530280181eabce3ded01e62283c7" +
		"d6a64c6edbc9c789a28b10dc14be403602300ce3b5aee44c76557c01d131f3f3b23e5cc0cc3314ff371e358adedfc9cd7f49a0e47" +
		"dac3605f7b4390005f5f35cfa5003059e41c9339ea378b3e3497fc6a3f0fc6dc4264862e119c84ab2ea3bf4eaa9d7be104d88d7f3" +
		"ee08e098be001b5abf8a6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be4036023005722ee77a43f27" +
		"6ac49f356d3b8f3a82a68bc7e05049a538c9e14229da133a7458493825954a9f271dbb9e8c784eaa2bf318f9c2f935add547cb963" +
		"e10f4820dca6fda68419e374351c2fc8314b2b1b636932390a23224d23c30197a20ec4cb2a6836b1530280181eabce3ded01e6228" +
		"3c7d6a64c6edbc9c789a28b10dc14be4036023002369be2840ac2f4a60813d021a48a81a644aaa114a9251acc65cc39f08345c4a9" +
		"3c20cf5f1f5e1e1461ef762a7ea0ffcf05ef5a54b6333a9b029b7ab938c6a0dcbb96907efbc3ff8ed91d171ca26658a2c39058902" +
		"75375eda937eb45681627ea6836b1530280181eabce3ded01e62283c7d6a64c6edbc9c789a28b10dc14be403602300245960fffbb" +
		"6413a53b1716f450602d7a80850cf70bcec1257a0d4a090412f317fd875488fd65a0c0b38fee40f05b466f43ddfd65ec2a353f481" +
		"d9e4eb052e0ccf2cdbf7aa9c86c4ac3ac45d110231d5ce22fe54ad15f6f78b8666c391de3e7800"

	justification, err := DecodeJustification[hash.H256, uint64, runtime.BlakeTwo256](
		common.MustHexToBytes(encodedJustification),
	)

	require.NoError(t, err)
	require.NotNil(t, justification.Justification)
}

func TestDecodeGrandpaJustificationVerifyFinalizes(t *testing.T) {
	var a hash.H256 = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll

	// Invalid Encoding
	invalidEncoding := []byte{21}
	_, err := DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		invalidEncoding,
		HashNumber[hash.H256, uint64]{},
		2,
		grandpa.VoterSet[string]{})
	require.Error(t, err)

	// Invalid target
	justification := primitives.GrandpaJustification[hash.H256, uint64]{
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash:   a,
			TargetNumber: 1,
		},
	}

	encWrongTarget, err := scale.Marshal(justification)
	require.NoError(t, err)
	_, err = DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		encWrongTarget,
		HashNumber[hash.H256, uint64]{},
		2,
		grandpa.VoterSet[string]{})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid commit target in grandpa justification")

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		a,
		runtime.Digest{})

	hederList := []runtime.Header[uint64, hash.H256]{headerB}

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 1, ed25519.Alice))
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 1, ed25519.Bob))
	precommits = append(precommits, makePrecommit(t, string(headerB.Hash()), 2, 1, 1, ed25519.Charlie))

	expectedJustification := primitives.GrandpaJustification[hash.H256, uint64]{
		Round: 1,
		Commit: primitives.Commit[hash.H256, uint64]{
			TargetHash:   a,
			TargetNumber: 1,
			Precommits:   precommits,
		},
		VoteAncestries: hederList,
	}

	encodedJustification, err := scale.Marshal(expectedJustification)
	require.NoError(t, err)

	target := HashNumber[hash.H256, uint64]{
		Hash:   a,
		Number: 1,
	}

	idWeights := make([]grandpa.IDWeight[string], 0)
	for i := 1; i <= 4; i++ {
		var id ced25519.Public
		switch i {
		case 1:
			id = ed25519.Alice.Pair().Public().(ced25519.Public)
		case 2:
			id = ed25519.Bob.Pair().Public().(ced25519.Public)
		case 3:
			id = ed25519.Charlie.Pair().Public().(ced25519.Public)
		case 4:
			id = ed25519.Ferdie.Pair().Public().(ced25519.Public)
		}
		idWeights = append(idWeights, grandpa.IDWeight[string]{
			ID: string(id[:]), Weight: 1,
		})
	}
	voters := grandpa.NewVoterSet(idWeights)

	newJustification, err := DecodeGrandpaJustificationVerifyFinalizes[hash.H256, uint64, runtime.BlakeTwo256](
		encodedJustification,
		target,
		1,
		*voters)
	require.NoError(t, err)
	require.Equal(t, expectedJustification, newJustification.Justification)
}

func TestJustification_verify(t *testing.T) {
	// Nil voter case
	auths := make(primitives.AuthorityList, 0)
	justification := GrandpaJustification[hash.H256, uint64]{}
	err := justification.Verify(2, auths)
	require.ErrorIs(t, err, errInvalidAuthoritiesSet)

	// happy path
	for i := 1; i <= 4; i++ {
		var id ced25519.Public
		switch i {
		case 1:
			id = ed25519.Alice.Pair().Public().(ced25519.Public)
		case 2:
			id = ed25519.Bob.Pair().Public().(ced25519.Public)
		case 3:
			id = ed25519.Charlie.Pair().Public().(ced25519.Public)
		case 4:
			id = ed25519.Ferdie.Pair().Public().(ced25519.Public)
		}
		auths = append(auths, primitives.AuthorityIDWeight{
			AuthorityID:     id,
			AuthorityWeight: 1,
		})
	}

	var a hash.H256 = "a\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" //nolint:lll
	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		a,
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{headerB}

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 2, ed25519.Alice))
	precommits = append(precommits, makePrecommit(t, string(a), 1, 1, 2, ed25519.Bob))
	precommits = append(precommits, makePrecommit(t, string(headerB.Hash()), 2, 1, 2, ed25519.Charlie))

	validJustification := GrandpaJustification[hash.H256, uint64]{
		Justification: primitives.GrandpaJustification[hash.H256, uint64]{
			Round: 1,
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   a,
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: headerList,
		},
	}

	err = validJustification.Verify(2, auths)
	require.NoError(t, err)
}

func TestJustification_verifyWithVoterSet(t *testing.T) {
	// 1) invalid commit
	idWeights := make([]grandpa.IDWeight[string], 0)
	for i := 1; i <= 4; i++ {
		var id ced25519.Public
		switch i {
		case 1:
			id = ed25519.Alice.Pair().Public().(ced25519.Public)
		case 2:
			id = ed25519.Bob.Pair().Public().(ced25519.Public)
		case 3:
			id = ed25519.Charlie.Pair().Public().(ced25519.Public)
		case 4:
			id = ed25519.Ferdie.Pair().Public().(ced25519.Public)
		}
		idWeights = append(idWeights, grandpa.IDWeight[string]{
			ID: string(id[:]), Weight: 1,
		})
	}
	voters := grandpa.NewVoterSet(idWeights)

	invalidJustification := GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   "B",
				TargetNumber: 2,
			},
		},
	}

	err := invalidJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: invalid commit in grandpa justification")

	// 2) visitedHashes != ancestryHashes
	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		headerA.Hash(),
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{
		headerA,
		headerB,
	}

	var precommits []grandpa.SignedPrecommit[hash.H256, uint64, primitives.AuthoritySignature, primitives.AuthorityID]
	precommits = append(precommits, makePrecommit(t, string(headerA.Hash()), 1, 1, 2, ed25519.Alice))
	precommits = append(precommits, makePrecommit(t, string(headerA.Hash()), 1, 1, 2, ed25519.Bob))
	precommits = append(precommits, makePrecommit(t, string(headerB.Hash()), 2, 1, 2, ed25519.Charlie))

	validJustification := GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   headerA.Hash(),
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: headerList,
			Round:          1,
		},
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.ErrorIs(t, err, errBadJustification)
	require.Equal(t, err.Error(), "bad justification for header: "+
		"invalid precommit ancestries in grandpa justification with unused headers")

	// Valid case
	headerList = []runtime.Header[uint64, hash.H256]{
		headerB,
	}

	validJustification = GrandpaJustification[hash.H256, uint64]{
		primitives.GrandpaJustification[hash.H256, uint64]{
			Commit: primitives.Commit[hash.H256, uint64]{
				TargetHash:   headerA.Hash(),
				TargetNumber: 1,
				Precommits:   precommits,
			},
			VoteAncestries: headerList,
			Round:          1,
		},
	}

	err = validJustification.verifyWithVoterSet(2, *voters)
	require.NoError(t, err)
}

func Test_newAncestryChain(t *testing.T) {
	dummyHeader := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	expAncestryMap := make(map[hash.H256]runtime.Header[uint64, hash.H256])
	expAncestryMap[dummyHeader.Hash()] = dummyHeader
	type testCase struct {
		name    string
		headers []runtime.Header[uint64, hash.H256]
		want    ancestryChain[hash.H256, uint64]
	}
	tests := []testCase{
		{
			name:    "noInputHeaders",
			headers: []runtime.Header[uint64, hash.H256]{},
			want: ancestryChain[hash.H256, uint64]{
				ancestry: make(map[hash.H256]runtime.Header[uint64, hash.H256]),
			},
		},
		{
			name: "validInput",
			headers: []runtime.Header[uint64, hash.H256]{
				dummyHeader,
			},
			want: ancestryChain[hash.H256, uint64]{
				ancestry: expAncestryMap,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newAncestryChain[hash.H256, uint64](tt.headers); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newAncestryChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAncestryChain_Ancestry(t *testing.T) {
	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		headerA.Hash(),
		runtime.Digest{})

	headerC := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		3,
		hash.H256(""),
		hash.H256(""),
		headerB.Hash(),
		runtime.Digest{})

	invalidParentHeader := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		hash.H256("invalid"),
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{
		headerA,
		headerB,
		headerC,
	}
	invalidHeaderList := []runtime.Header[uint64, hash.H256]{
		invalidParentHeader,
	}
	validAncestryMap := newAncestryChain[hash.H256, uint64](headerList)
	invalidAncestryMap := newAncestryChain[hash.H256, uint64](invalidHeaderList)

	type testCase struct {
		name   string
		chain  ancestryChain[hash.H256, uint64]
		base   hash.H256
		block  hash.H256
		want   []hash.H256
		expErr error
	}
	tests := []testCase{
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerA.Hash(),
			want:  []hash.H256{},
		},
		{
			name:   "baseEqualsBlock",
			chain:  validAncestryMap,
			base:   headerA.Hash(),
			block:  "notDescendant",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:   "invalidParentHashField",
			chain:  invalidAncestryMap,
			base:   headerA.Hash(),
			block:  "notDescendant",
			expErr: errBlockNotDescendentOfBase,
		},
		{
			name:  "validRoute",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerC.Hash(),
			want:  []hash.H256{headerB.Hash()},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.chain.Ancestry(tt.base, tt.block)
			assert.ErrorIs(t, err, tt.expErr)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAncestryChain_IsEqualOrDescendantOf(t *testing.T) {
	headerA := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		1,
		hash.H256(""),
		hash.H256(""),
		hash.H256(""),
		runtime.Digest{})

	headerB := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		2,
		hash.H256(""),
		hash.H256(""),
		headerA.Hash(),
		runtime.Digest{})

	headerC := generic.NewHeader[uint64, hash.H256, runtime.BlakeTwo256](
		3,
		hash.H256(""),
		hash.H256(""),
		headerB.Hash(),
		runtime.Digest{})

	headerList := []runtime.Header[uint64, hash.H256]{
		headerA,
		headerB,
		headerC,
	}

	validAncestryMap := newAncestryChain[hash.H256, uint64](headerList)

	type testCase struct {
		name  string
		chain ancestryChain[hash.H256, uint64]
		base  hash.H256
		block hash.H256
		want  bool
	}
	tests := []testCase{
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerA.Hash(),
			want:  true,
		},
		{
			name:  "baseEqualsBlock",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: "someInvalidBLock",
			want:  false,
		},
		{
			name:  "validRoute",
			chain: validAncestryMap,
			base:  headerA.Hash(),
			block: headerC.Hash(),
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.chain.IsEqualOrDescendantOf(tt.base, tt.block)
			assert.Equal(t, tt.want, got)
		})
	}
}
