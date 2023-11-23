package dispute

import (
	"fmt"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	"sync"
	"sync/atomic"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// CandidateComparator comparator for ordering of disputes for candidate.
type CandidateComparator struct {
	relayParentBlockNumber *uint32
	candidateHash          common.Hash
}

// NewCandidateComparator creates a new CandidateComparator.
func NewCandidateComparator(relayParentBlockNumber *uint32,
	candidateHash common.Hash,
) CandidateComparator {
	return CandidateComparator{
		relayParentBlockNumber: relayParentBlockNumber,
		candidateHash:          candidateHash,
	}
}

// ParticipationRequest a dispute participation request
type ParticipationRequest struct {
	candidateHash    common.Hash
	candidateReceipt parachainTypes.CandidateReceipt
	session          parachainTypes.SessionIndex
	//TODO: requestTimer for metrics
}

// ParticipationRequestWithPriority a dispute participation request with priority
type ParticipationRequestWithPriority struct {
	request  ParticipationRequest
	priority ParticipationPriority
}

// ParticipationStatement is a statement as result of the validation process.
type ParticipationStatement struct {
	Session          parachainTypes.SessionIndex
	CandidateHash    common.Hash
	CandidateReceipt parachainTypes.CandidateReceipt
	Outcome          types.ParticipationOutcomeVDT
}

// Participation keeps track of the disputes we need to participate in.
type Participation interface {
	// Queue a dispute for the node to participate in
	Queue(overseerChannel chan<- any, request ParticipationRequest, priority ParticipationPriority) error

	// Clear clears a participation request. This is called when we have the dispute result.
	Clear(candidateHash common.Hash) error

	// ProcessActiveLeavesUpdate processes an active leaves update
	ProcessActiveLeavesUpdate(update overseer.ActiveLeavesUpdate)

	// BumpPriority bumps the priority for the given receipts
	BumpPriority(overseerChannel chan<- any, receipts []parachainTypes.CandidateReceipt)
}

type block struct {
	Number uint32
	Hash   common.Hash
}

// ParticipationHandler handles dispute participation.
type ParticipationHandler struct {
	runningParticipation sync.Map
	workers              atomic.Int32

	queue       Queue
	recentBlock *block

	runtime parachain.RuntimeInstance

	overseer chan<- any

	//TODO: metrics
}

const MaxParallelParticipation = 3

func (p *ParticipationHandler) Queue(overseerChannel chan<- any,
	request ParticipationRequest,
	priority ParticipationPriority,
) error {
	if _, ok := p.runningParticipation.Load(request.candidateHash); ok {
		return nil
	}

	// if we already have a recent block, participate right away
	if p.recentBlock != nil && p.numberOfWorkers() < MaxParallelParticipation {
		p.forkParticipation(&request, p.recentBlock.Hash)
		return nil
	}

	blockNumber, err := getBlockNumber(overseerChannel, request.candidateReceipt)
	if err != nil {
		return fmt.Errorf("get block number: %w", err)
	}

	candidateHash, err := request.candidateReceipt.Hash()
	if err != nil {
		return fmt.Errorf("hash candidate receipt: %w", err)
	}

	comparator := NewCandidateComparator(&blockNumber, candidateHash)
	if err := p.queue.Queue(comparator, &request, priority); err != nil {
		return fmt.Errorf("queue ParticipationHandler request: %w", err)
	}

	return nil
}

func (p *ParticipationHandler) Clear(candidateHash common.Hash) error {
	p.runningParticipation.Delete(candidateHash)
	p.workers.Add(-1)

	if p.recentBlock == nil {
		panic("we never ever reset recentBlock to nil and we already received a result, so it must have been set before. qed")
	}

	p.dequeueUntilCapacity(p.recentBlock.Hash)
	return nil
}

func (p *ParticipationHandler) ProcessActiveLeavesUpdate(update overseer.ActiveLeavesUpdate) {
	// TODO: to check if this is needed here
	// if it is being called in only one place, we could just add a check there itself
	if update.Activated == nil {
		return
	}

	if p.recentBlock != nil {
		if update.Activated.Number > p.recentBlock.Number {
			p.recentBlock.Number = update.Activated.Number
			p.recentBlock.Hash = update.Activated.Hash
		}
		return
	}

	p.recentBlock = &block{
		Number: update.Activated.Number,
		Hash:   update.Activated.Hash,
	}
	p.dequeueUntilCapacity(update.Activated.Hash)
}

func (p *ParticipationHandler) BumpPriority(overseerChannel chan<- any, receipts []parachainTypes.CandidateReceipt) {
	for _, receipt := range receipts {
		blockNumber, err := getBlockNumber(overseerChannel, receipt)
		if err != nil {
			logger.Errorf(
				"failed to get block number. CommitmentsHash: %s, Error: %s",
				receipt.CommitmentsHash.String(),
				err,
			)
			continue
		}

		candidateHash, err := receipt.Hash()
		if err != nil {
			logger.Errorf(
				"failed to hash candidate receipt. CommitmentsHash: %s, Error: %s",
				receipt.CommitmentsHash.String(),
				err,
			)
			continue
		}
		comparator := NewCandidateComparator(&blockNumber, candidateHash)

		if err := p.queue.PrioritiseIfPresent(comparator); err != nil {
			logger.Errorf(
				"failed to prioritise candidate. CommitmentsHash: %s, Error: %s",
				receipt.CommitmentsHash.String(),
				err,
			)
			continue
		}
	}
}

func (p *ParticipationHandler) numberOfWorkers() int {
	return int(p.workers.Load())
}

func (p *ParticipationHandler) dequeueUntilCapacity(recentHead common.Hash) {
	for p.numberOfWorkers() < MaxParallelParticipation {
		request := p.queue.Dequeue()
		if request == nil {
			break
		}

		p.forkParticipation(request.request, recentHead)
	}
}

func (p *ParticipationHandler) forkParticipation(request *ParticipationRequest, recentHead common.Hash) {
	_, ok := p.runningParticipation.LoadOrStore(request.candidateHash, nil)
	if ok {
		return
	}

	p.workers.Add(1)
	go func() {
		if err := p.participate(recentHead, *request); err != nil {
			logger.Debugf(
				"failed to participate in dispute. CandidateHash: %s, Error: %s",
				request.candidateHash.String(),
				err,
			)
		}
	}()
}

func (p *ParticipationHandler) participate(blockHash common.Hash, request ParticipationRequest) error {
	// get available data from the overseer
	respCh := make(chan any, 1)
	message := overseer.AvailabilityRecoveryMessage{
		CandidateReceipt: request.candidateReceipt,
		SessionIndex:     request.session,
		GroupIndex:       nil,
		ResponseChannel:  respCh,
	}
	res, err := call(p.overseer, message, message.ResponseChannel)
	if err != nil {
		return fmt.Errorf("send availability recovery message: %w", err)
	}

	availableData, ok := res.(overseer.AvailabilityRecoveryResponse)
	if !ok {
		return fmt.Errorf("unexpected response type: %T", res)
	}

	if availableData.Error != nil {
		switch *availableData.Error {
		case overseer.RecoveryErrorInvalid:
			sendResult(p.overseer, request, types.ParticipationOutcomeInvalid)
			return fmt.Errorf("invalid available data: %s", availableData.Error.String())
		case overseer.RecoveryErrorUnavailable:
			sendResult(p.overseer, request, types.ParticipationOutcomeUnAvailable)
			return fmt.Errorf("unavailable data: %s", availableData.Error.String())
		default:
			return fmt.Errorf("unexpected recovery error: %d", availableData.Error)
		}
	}

	validationCode, err := p.runtime.ParachainHostValidationCodeByHash(
		blockHash,
		request.candidateReceipt.Descriptor.ValidationCodeHash)
	if err != nil || validationCode == nil {
		sendResult(p.overseer, request, types.ParticipationOutcomeError)
		return fmt.Errorf("failed to get validation code: %w", err)
	}

	if len(*validationCode) == 0 {
		logger.Errorf(
			"validation code is empty. CandidateHash: %s",
			request.candidateHash.String(),
		)
		sendResult(p.overseer, request, types.ParticipationOutcomeError)
		return fmt.Errorf("validation code is empty")
	}

	// validate the request and send the result
	respChan := make(chan any, 1)
	validateMessage := overseer.ValidateFromChainState{
		CandidateReceipt:   request.candidateReceipt,
		PoV:                availableData.AvailableData.POV,
		PvfExecTimeoutKind: overseer.PvfExecTimeoutKindApproval,
		ResponseChannel:    respChan,
	}
	res, err = call(p.overseer, validateMessage, validateMessage.ResponseChannel)
	if err != nil {
		sendResult(p.overseer, request, types.ParticipationOutcomeError)
	}
	result, ok := res.(overseer.ValidationResult)
	if !ok {
		sendResult(p.overseer, request, types.ParticipationOutcomeError)
		return fmt.Errorf("unexpected response type: %T", res)
	}

	if result.Error != nil {
		// validation failed
		sendResult(p.overseer, request, types.ParticipationOutcomeError)
		return fmt.Errorf("validation failed: %s", result.Error)
	}
	if !result.IsValid {
		sendResult(p.overseer, request, types.ParticipationOutcomeInvalid)
		return fmt.Errorf("validation failed: %s", result.Error)
	}

	sendResult(p.overseer, request, types.ParticipationOutcomeValid)
	return nil
}

var _ Participation = (*ParticipationHandler)(nil)

func NewParticipation(overseer chan<- any, runtime parachain.RuntimeInstance) *ParticipationHandler {
	return &ParticipationHandler{
		runningParticipation: sync.Map{},
		queue:                NewQueue(),
		overseer:             overseer,
		runtime:              runtime,
	}
}
