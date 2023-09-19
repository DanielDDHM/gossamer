package scraping

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	lrucache "github.com/ChainSafe/gossamer/lib/utils/lru-cache"
)

const (
	// DisputeCandidateLifetimeAfterFinalization How many blocks after finalisation an information about
	// backed/included candidate should be preloaded (when scraping onchain votes) and kept locally (when pruning).
	DisputeCandidateLifetimeAfterFinalization = uint32(10)

	// AncestryChunkSize Limits the number of ancestors received for a single request
	AncestryChunkSize = uint32(10)

	// AncestrySizeLimit Limits the overall number of ancestors walked through for a given head.
	AncestrySizeLimit = int(500) // TODO: This should be MaxFinalityLag

	// LRUObservedBlocksCapacity Number of hashes to keep in the LRU cache.
	LRUObservedBlocksCapacity = 20
)

// ChainScraper Scrapes non finalized chain in order to collect information from blocks
type ChainScraper struct {
	// All candidates we have seen included, which not yet have been finalized.
	IncludedCandidates ScrapedCandidates
	// All candidates we have seen backed
	BackedCandidates ScrapedCandidates
	// Latest relay blocks observed by the provider.
	LastObservedBlocks *lrucache.LRUCache[common.Hash, *uint32]
	// Maps included candidate hashes to one or more relay block heights and hashes.
	Inclusions Inclusions
	// Runtime instance
	Runtime parachain.RuntimeInstance
}

// IsCandidateIncluded Check whether we have seen a candidate included on any chain.
func (cs *ChainScraper) IsCandidateIncluded(candidateHash common.Hash) bool {
	return cs.IncludedCandidates.Contains(candidateHash)
}

// IsCandidateBacked Check whether the candidate is backed
func (cs *ChainScraper) IsCandidateBacked(candidateHash common.Hash) bool {
	return cs.BackedCandidates.Contains(candidateHash)
}

// GetBlocksIncludingCandidate Get blocks including the given candidate hash
func (cs *ChainScraper) GetBlocksIncludingCandidate(candidateHash common.Hash) []Inclusion {
	return cs.Inclusions.Get(candidateHash)
}

// ProcessActiveLeavesUpdate Process active leaves update
func (cs *ChainScraper) ProcessActiveLeavesUpdate(
	sender overseer.Sender,
	update overseer.ActiveLeavesUpdate,
) (*parachainTypes.ScrapedUpdates, error) {
	if update.Activated == nil {
		return &parachainTypes.ScrapedUpdates{}, nil
	}

	ancestors, err := cs.GetRelevantBlockAncestors(sender, update.Activated.Hash, update.Activated.Number)
	if err != nil {
		return nil, fmt.Errorf("getting relevant block ancestors: %w", err)
	}

	earliestBlockNumber := update.Activated.Number - uint32(len(ancestors))
	var blockNumbers []uint32
	for i := update.Activated.Number; i >= earliestBlockNumber; i-- {
		blockNumbers = append(blockNumbers, i)
	}
	blockHashes := append([]common.Hash{update.Activated.Hash}, ancestors...)

	var scrapedUpdates parachainTypes.ScrapedUpdates
	for i, blockNumber := range blockNumbers {
		blockHash := blockHashes[i]

		receiptsForBlock, err := cs.ProcessCandidateEvents(blockNumber, blockHash)
		if err != nil {
			return nil, fmt.Errorf("processing candidate events: %w", err)
		}
		scrapedUpdates.IncludedReceipts = append(scrapedUpdates.IncludedReceipts, receiptsForBlock...)

		onChainVotes, err := cs.Runtime.ParachainHostOnChainVotes(blockHash)
		if err != nil {
			return nil, fmt.Errorf("getting onchain votes: %w", err)
		}

		if onChainVotes != nil {
			scrapedUpdates.OnChainVotes = append(scrapedUpdates.OnChainVotes, *onChainVotes)
		}
	}

	cs.LastObservedBlocks.Put(update.Activated.Hash, &update.Activated.Number)
	return &scrapedUpdates, nil
}

// ProcessFinalisedBlock prune finalised candidates.
func (cs *ChainScraper) ProcessFinalisedBlock(finalisedBlock uint32) {
	if finalisedBlock < DisputeCandidateLifetimeAfterFinalization-1 {
		// Nothing to prune. We are still in the beginning of the chain and there are not
		// enough finalized blocks yet.
		return
	}
	keyToPrune := finalisedBlock - (DisputeCandidateLifetimeAfterFinalization - 1)

	cs.BackedCandidates.RemoveUptoHeight(keyToPrune)
	candidatesModified := cs.IncludedCandidates.RemoveUptoHeight(keyToPrune)
	cs.Inclusions.RemoveUpToHeight(keyToPrune, candidatesModified)
}

// ProcessCandidateEvents Process candidate events
func (cs *ChainScraper) ProcessCandidateEvents(
	blockNumber uint32,
	blockHash common.Hash,
) ([]parachainTypes.CandidateReceipt, error) {
	var (
		candidateEvents  []parachainTypes.CandidateEvent
		includedReceipts []parachainTypes.CandidateReceipt
	)

	events, err := cs.Runtime.ParachainHostCandidateEvents(blockHash)
	if err != nil {
		return nil, fmt.Errorf("getting candidate events: %w", err)
	}

	for _, event := range events.Types {
		candidateEvents = append(candidateEvents, parachainTypes.CandidateEvent(event))
	}

	for _, candidateEvent := range candidateEvents {
		e, err := candidateEvent.Value()
		if err != nil {
			return nil, fmt.Errorf("getting candidate event value: %w", err)
		}
		switch event := e.(type) {
		case parachainTypes.CandidateIncluded:
			candidateHash, err := event.CandidateReceipt.Hash()
			if err != nil {
				return nil, fmt.Errorf("getting candidate receipt hash: %w", err)
			}

			cs.IncludedCandidates.Insert(blockNumber, candidateHash)
			cs.Inclusions.Insert(candidateHash, blockHash, blockNumber)
			includedReceipts = append(includedReceipts, event.CandidateReceipt)
		case parachainTypes.CandidateBacked:
			candidateHash, err := event.CandidateReceipt.Hash()
			if err != nil {
				return nil, fmt.Errorf("getting candidate receipt hash: %w", err)
			}

			cs.BackedCandidates.Insert(blockNumber, candidateHash)
		default:
			// skip the rest
		}
	}

	return includedReceipts, nil
}

// GetRelevantBlockAncestors Get relevant block ancestors
func (cs *ChainScraper) GetRelevantBlockAncestors(
	sender overseer.Sender,
	head common.Hash,
	headNumber uint32,
) ([]common.Hash, error) {
	targetAncestor, err := getFinalisedBlockNumber(sender)
	if err != nil {
		return nil, fmt.Errorf("getting finalised block number: %w", err)
	}
	targetAncestor = saturatingSub(targetAncestor, DisputeCandidateLifetimeAfterFinalization)
	var ancestors []common.Hash

	// If headNumber <= targetAncestor + 1 the ancestry will be empty.
	if observedBlock := cs.LastObservedBlocks.Get(head); observedBlock != nil || headNumber <= targetAncestor+1 {
		return ancestors, nil
	}

	for {
		hashes, err := getBlockAncestors(sender, head, AncestryChunkSize)
		if err != nil {
			return nil, fmt.Errorf("getting block ancestors: %w", err)
		}

		if len(hashes) == 0 {
			break
		}

		earliestBlockNumber := saturatingSub(headNumber, uint32(len(hashes)))
		var blockNumbers []uint32
		// The reversed order is parent, grandparent, etc. excluding the head.
		startIndex := int(headNumber - 1)
		endIndex := int(earliestBlockNumber)
		for i := startIndex; i >= endIndex; i-- {
			blockNumbers = append(blockNumbers, uint32(i))
		}

		for i, blockNumber := range blockNumbers {
			// Return if we either met target/cached block or hit the size limit for the returned ancestry of head.
			if cs.LastObservedBlocks.Get(hashes[i]) != nil ||
				blockNumber <= targetAncestor ||
				len(ancestors) >= AncestrySizeLimit {
				return ancestors, nil
			}
			ancestors = append(ancestors, hashes[i])
		}

		head = hashes[len(hashes)-1]
		headNumber = earliestBlockNumber
	}

	return ancestors, nil
}

// IsPotentialSpam Check whether the vote state is a potential spam
func (cs *ChainScraper) IsPotentialSpam(voteState types.CandidateVoteState, candidateHash common.Hash) (bool, error) {
	isDisputed := voteState.IsDisputed()
	isIncluded := cs.IsCandidateIncluded(candidateHash)
	isBacked := cs.IsCandidateBacked(candidateHash)
	isConfirmed, err := voteState.IsConfirmed()
	if err != nil {
		return false, fmt.Errorf("checking if the vote state is confirmed: %w", err)
	}

	return isDisputed && !isIncluded && !isBacked && !isConfirmed, nil
}

// NewChainScraper New chain scraper
func NewChainScraper(
	sender overseer.Sender,
	runtime parachain.RuntimeInstance,
	initialHead *overseer.ActivatedLeaf,
) (*ChainScraper, *parachainTypes.ScrapedUpdates, error) {
	chainScraper := &ChainScraper{
		IncludedCandidates: NewScrapedCandidates(),
		BackedCandidates:   NewScrapedCandidates(),
		LastObservedBlocks: lrucache.NewLRUCache[common.Hash, *uint32](LRUObservedBlocksCapacity),
		Inclusions:         NewInclusions(),
		Runtime:            runtime,
	}

	update := overseer.ActiveLeavesUpdate{
		Activated: initialHead,
	}
	updates, err := chainScraper.ProcessActiveLeavesUpdate(sender, update)
	if err != nil {
		return nil, nil, fmt.Errorf("processing active leaves update: %w", err)
	}

	return chainScraper, updates, nil
}

// getFinalisedBlockNumber sends a message to the overseer to get the finalised block number.
func getFinalisedBlockNumber(sender overseer.Sender) (uint32, error) {
	tx := make(chan overseer.FinalizedBlockNumberResponse, 1)
	message := overseer.FinalizedBlockNumberRequest{
		ResponseChannel: tx,
	}
	err := sender.SendMessage(message)
	if err != nil {
		return 0, fmt.Errorf("sending message to get finalised block number: %w", err)
	}

	response := <-tx
	if response.Err != nil {
		return 0, fmt.Errorf("getting finalised block number: %w", response.Err)
	}

	return response.Number, nil
}

// getBlockAncestors sends a message to the overseer to get the ancestors of a block.
func getBlockAncestors(
	sender overseer.Sender,
	head common.Hash,
	numAncestors uint32,
) ([]common.Hash, error) {
	tx := make(chan overseer.AncestorsResponse, 1)
	message := overseer.AncestorsRequest{
		Hash:            head,
		K:               numAncestors,
		ResponseChannel: tx,
	}
	err := sender.SendMessage(message)
	if err != nil {
		return nil, fmt.Errorf("sending message to get block ancestors: %w", err)
	}

	response := <-tx
	if response.Error != nil {
		return nil, fmt.Errorf("getting block ancestors: %w", response.Error)
	}

	return response.Ancestors, nil
}

// saturatingSub returns the result of a - b, saturating at 0.
func saturatingSub(a, b uint32) uint32 {
	result := int(a) - int(b)
	if result < 0 {
		return 0
	}
	return uint32(result)
}
