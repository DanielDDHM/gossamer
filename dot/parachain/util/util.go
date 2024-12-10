// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package util

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/dot/parachain/chainapi"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/ChainSafe/gossamer/lib/runtime"
	wazero_runtime "github.com/ChainSafe/gossamer/lib/runtime/wazero"
	"github.com/libp2p/go-libp2p/core/peer"
)

type HashHeader struct {
	Hash   common.Hash
	Header types.Header
}

type AncestorsResponse struct {
	Ancestors []common.Hash
	Error     error
}

// Ancestors is a message to get the ancestors of a block.
type Ancestors struct {
	Hash              common.Hash
	numberOfAncestors uint32
}

// UnifiedReputationChange represents a reputation change for a peer.
type UnifiedReputationChange struct {
	Change int32
	Reason string
}

// NetworkBridgeTxMessage represents the message sent to the network subsystem.
type NetworkBridgeTxMessage struct {
	ReportPeerMessageBatch map[peer.ID]int32
}

// ReputationAggregator aggregates reputation changes for peers.
type ReputationAggregator struct {
	sendImmediatelyIf func(rep UnifiedReputationChange) bool
	byPeer            map[peer.ID]int32
	mu                sync.Mutex // Mutex for thread safety
}

// NewReputationAggregator creates a new ReputationAggregator.
func NewReputationAggregator(sendImmediatelyIf func(rep UnifiedReputationChange) bool) *ReputationAggregator {
	return &ReputationAggregator{
		byPeer:            make(map[peer.ID]int32),
		sendImmediatelyIf: sendImmediatelyIf,
	}
}

// Send sends the aggregated reputation changes in a batch and clears the state.
func (r *ReputationAggregator) Send(overseerCh chan<- NetworkBridgeTxMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// If there's nothing to send, exit
	if len(r.byPeer) == 0 {
		return
	}

	// Create the batch message
	message := NetworkBridgeTxMessage{
		ReportPeerMessageBatch: r.byPeer,
	}

	// Send the message
	overseerCh <- message

	// Clear the map after sending
	r.byPeer = make(map[peer.ID]int32)
}

// Modify adds a reputation change to the internal state or sends it immediately if needed.
func (r *ReputationAggregator) Modify(overseerCh chan<- NetworkBridgeTxMessage, peerID peer.ID, rep UnifiedReputationChange) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if the change should be sent immediately
	if r.sendImmediatelyIf(rep) {
		// Send immediately without adding to the aggregator
		overseerCh <- NetworkBridgeTxMessage{
			ReportPeerMessageBatch: map[peer.ID]int32{
				peerID: rep.Change,
			},
		}
		return
	}

	// Otherwise, accumulate the reputation change
	if r.byPeer == nil {
		r.byPeer = make(map[peer.ID]int32)
	}
	r.byPeer[peerID] += rep.Change
}

// SigningKeyAndIndex finds the first key we can sign with from the given set of validators,
// if any, and returns it along with the validator index.
func SigningKeyAndIndex(
	validators []parachaintypes.ValidatorID,
	ks keystore.Keystore,
) (*parachaintypes.ValidatorID, parachaintypes.ValidatorIndex) {
	for i, validator := range validators {
		publicKey, _ := sr25519.NewPublicKey(validator[:])
		keypair := ks.GetKeypair(publicKey)

		if keypair != nil {
			return &validator, parachaintypes.ValidatorIndex(i)
		}
	}
	return nil, 0
}

// DetermineNewBlocks determines the hashes of all new blocks we should track metadata for, given this head.
//
// This is guaranteed to be a subset of the (inclusive) ancestry of `head` determined as all
// blocks above the lower bound or above the highest known block, whichever is higher.
// This is formatted in descending order by block height.
//
// An implication of this is that if `head` itself is known or not above the lower bound,
// then the returned list will be empty.
//
// This may be somewhat expensive when first recovering from major sync.
//
// NOTE: TOTO: this issue needs to be finished, see issue #3933
func DetermineNewBlocks(subsystemToOverseer chan<- any, isKnown func(hash common.Hash) bool, head common.Hash,
	header types.Header,
	lowerBoundNumber parachaintypes.BlockNumber) ([]HashHeader, error) {
	const maxNumberOfAncestors = 4
	minBlockNeeded := uint(lowerBoundNumber + 1)

	// Early exit if the block is in the DB or too early.
	alreadyKnown := isKnown(head)
	beforeRelevant := header.Number < minBlockNeeded
	if alreadyKnown || beforeRelevant {
		return nil, nil
	}

	ancestry := make([]HashHeader, 0)
	headerClone, err := header.DeepCopy()
	if err != nil {
		return nil, fmt.Errorf("failed to deep copy header: %w", err)
	}

	ancestry = append(ancestry, HashHeader{Hash: head, Header: *headerClone})

	// Early exit if the parent hash is in the DB or no further blocks are needed.
	if isKnown(header.ParentHash) || header.Number == minBlockNeeded {
		return ancestry, nil
	}

	lastHeader := ancestry[len(ancestry)-1].Header
	// This is always non-zero as determined by the loop invariant above.
	numberOfAncestors := min(maxNumberOfAncestors, (lastHeader.Number - minBlockNeeded))

	ancestors, err := GetBlockAncestors(subsystemToOverseer, head, uint32(numberOfAncestors))
	if err != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", err)
	}
	fmt.Printf("ancestors: %v\n", ancestors)
	// TODO: finish this, see issue #3933

	return ancestry, nil
}

// SendOverseerMessage sends the given message to the given channel and waits for a response with a timeout
func SendOverseerMessage(channel chan<- any, message any, responseChan chan any) (any, error) {
	channel <- message
	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		return nil, parachaintypes.ErrSubsystemRequestTimeout
	}
}

// GetBlockAncestors sends a message to the overseer to get the ancestors of a block.
func GetBlockAncestors(
	overseerChannel chan<- any,
	head common.Hash,
	numAncestors uint32,
) ([]common.Hash, error) {
	respChan := make(chan any, 1)
	message := chainapi.ChainAPIMessage[Ancestors]{
		Message: Ancestors{
			Hash:              head,
			numberOfAncestors: numAncestors,
		},
		ResponseChannel: respChan,
	}
	res, err := SendOverseerMessage(overseerChannel, message, message.ResponseChannel)
	if err != nil {
		return nil, err
	}

	response, ok := res.(AncestorsResponse)
	if !ok {
		return nil, fmt.Errorf("got unexpected response type %T", res)
	}
	if response.Error != nil {
		return nil, response.Error
	}

	return response.Ancestors, nil
}

func ExecutorParamsAtRelayParent(rt runtime.Instance, relayParent common.Hash,
) (*parachaintypes.ExecutorParams, error) {
	sessionIndex, err := rt.ParachainHostSessionIndexForChild()
	if err != nil {
		return nil, fmt.Errorf("getting session index for relay parent %s: %w", relayParent, err)
	}

	executorParams, err := rt.ParachainHostSessionExecutorParams(sessionIndex)
	if err != nil {
		if errors.Is(err, wazero_runtime.ErrExportFunctionNotFound) {
			// Runtime doesn't yet support the api requested,
			// should execute anyway with default set of parameters.
			defaultExecutorParams := parachaintypes.NewExecutorParams()
			return &defaultExecutorParams, nil
		}
		return nil, err
	}

	if executorParams == nil {
		// should never happen
		panic(fmt.Sprintf("executor params for relay parent %s is nil", relayParent))
	}

	return executorParams, nil
}
