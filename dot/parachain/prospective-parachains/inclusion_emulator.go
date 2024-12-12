package prospectiveparachains

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ethereum/go-ethereum/common/math"
)

// ProspectiveCandidate includes key informations that represents a candidate
// without pinning it to a particular session. For example, commitments are
// represented here, but the erasure-root is not. This means that, prospective
// candidates are not correlated to any session in particular.
type ProspectiveCandidate struct {
	Commitments             parachaintypes.CandidateCommitments
	PersistedValidationData parachaintypes.PersistedValidationData
	PoVHash                 common.Hash
	ValidationCodeHash      parachaintypes.ValidationCodeHash
}

type ErrDisallowedHrmpWatermark struct {
	BlockNumber uint
}

func (e *ErrDisallowedHrmpWatermark) Error() string {
	return fmt.Sprintf("DisallowedHrmpWatermark(BlockNumber: %d)", e.BlockNumber)
}

type ErrNoSuchHrmpChannel struct {
	paraID parachaintypes.ParaID
}

func (e *ErrNoSuchHrmpChannel) Error() string {
	return fmt.Sprintf("NoSuchHrmpChannel(ParaId: %d)", e.paraID)
}

type ErrHrmpMessagesOverflow struct {
	paraID            parachaintypes.ParaID
	messagesRemaining uint32
	messagesSubmitted uint32
}

func (e *ErrHrmpMessagesOverflow) Error() string {
	return fmt.Sprintf("HrmpMessagesOverflow(ParaId: %d, MessagesRemaining: %d, MessagesSubmitted: %d)",
		e.paraID, e.messagesRemaining, e.messagesSubmitted)
}

type ErrHrmpBytesOverflow struct {
	paraID         parachaintypes.ParaID
	bytesRemaining uint32
	bytesSubmitted uint32
}

func (e *ErrHrmpBytesOverflow) Error() string {
	return fmt.Sprintf("HrmpBytesOverflow(ParaId: %d, BytesRemaining: %d, BytesSubmitted: %d)",
		e.paraID, e.bytesRemaining, e.bytesSubmitted)
}

type ErrUmpMessagesOverflow struct {
	messagesRemaining uint32
	messagesSubmitted uint32
}

func (e *ErrUmpMessagesOverflow) Error() string {
	return fmt.Sprintf("UmpMessagesOverflow(MessagesRemaining: %d, MessagesSubmitted: %d)",
		e.messagesRemaining, e.messagesSubmitted)
}

type ErrUmpBytesOverflow struct {
	bytesRemaining uint32
	bytesSubmitted uint32
}

func (e *ErrUmpBytesOverflow) Error() string {
	return fmt.Sprintf("UmpBytesOverflow(BytesRemaining: %d, BytesSubmitted: %d)", e.bytesRemaining, e.bytesSubmitted)
}

type ErrDmpMessagesUnderflow struct {
	messagesRemaining uint32
	messagesProcessed uint32
}

func (e *ErrDmpMessagesUnderflow) Error() string {
	return fmt.Sprintf("DmpMessagesUnderflow(MessagesRemaining: %d, MessagesProcessed: %d)",
		e.messagesRemaining, e.messagesProcessed)
}

var (
	ErrAppliedNonexistentCodeUpgrade = errors.New("AppliedNonexistentCodeUpgrade()")
	ErrDmpAdvancementRule            = errors.New("DmpAdvancementRule()")
	ErrCodeUpgradeRestricted         = errors.New("CodeUpgradeRestricted()")
)

type ErrValidationCodeMismatch struct {
	expected parachaintypes.ValidationCodeHash
	got      parachaintypes.ValidationCodeHash
}

func (e *ErrValidationCodeMismatch) Error() string {
	return fmt.Sprintf("ValidationCodeMismatch(Expected: %v, Got: %v)", e.expected, e.got)
}

type ErrOutputsInvalid struct {
	ModificationError error
}

func (e *ErrOutputsInvalid) Error() string {
	return fmt.Sprintf("OutputsInvalid(ModificationError: %v)", e.ModificationError)
}

type ErrCodeSizeTooLarge struct {
	maxAllowed uint32
	newSize    uint32
}

func (e *ErrCodeSizeTooLarge) Error() string {
	return fmt.Sprintf("CodeSizeTooLarge(MaxAllowed: %d, NewSize: %d)", e.maxAllowed, e.newSize)
}

type ErrRelayParentTooOld struct {
	minAllowed uint
	current    uint
}

func (e *ErrRelayParentTooOld) Error() string {
	return fmt.Sprintf("RelayParentTooOld(MinAllowed: %d, Current: %d)", e.minAllowed, e.current)
}

type ErrUmpMessagesPerCandidateOverflow struct {
	messagesAllowed   uint32
	messagesSubmitted uint32
}

func (e *ErrUmpMessagesPerCandidateOverflow) Error() string {
	return fmt.Sprintf("UmpMessagesPerCandidateOverflow(MessagesAllowed: %d, MessagesSubmitted: %d)",
		e.messagesAllowed, e.messagesSubmitted)
}

type ErrHrmpMessagesPerCandidateOverflow struct {
	messagesAllowed   uint32
	messagesSubmitted uint32
}

func (e *ErrHrmpMessagesPerCandidateOverflow) Error() string {
	return fmt.Sprintf("HrmpMessagesPerCandidateOverflow(MessagesAllowed: %d, MessagesSubmitted: %d)",
		e.messagesAllowed, e.messagesSubmitted)
}

type ErrHrmpMessagesDescendingOrDuplicate struct {
	index uint
}

func (e *ErrHrmpMessagesDescendingOrDuplicate) Error() string {
	return fmt.Sprintf("HrmpMessagesDescendingOrDuplicate(Index: %d)", e.index)
}

// RelayChainBlockInfo contains minimum information about a relay-chain block.
type RelayChainBlockInfo struct {
	Hash        common.Hash
	StorageRoot common.Hash
	Number      uint
}

func CheckModifications(c *parachaintypes.Constraints, modifications *ConstraintModifications) error {
	if modifications.HrmpWatermark != nil && modifications.HrmpWatermark.Type == Trunk {
		if !slices.Contains(c.HrmpInbound.ValidWatermarks, modifications.HrmpWatermark.Watermark()) {
			return &ErrDisallowedHrmpWatermark{BlockNumber: modifications.HrmpWatermark.Watermark()}
		}
	}

	for id, outboundHrmpMod := range modifications.OutboundHrmp {
		outbound, ok := c.HrmpChannelsOut[id]
		if !ok {
			return &ErrNoSuchHrmpChannel{paraID: id}
		}

		_, overflow := math.SafeSub(uint64(outbound.BytesRemaining), uint64(outboundHrmpMod.BytesSubmitted))
		if overflow {
			return &ErrHrmpBytesOverflow{
				paraID:         id,
				bytesRemaining: outbound.BytesRemaining,
				bytesSubmitted: outboundHrmpMod.BytesSubmitted,
			}
		}

		_, overflow = math.SafeSub(uint64(outbound.MessagesRemaining), uint64(outboundHrmpMod.MessagesSubmitted))
		if overflow {
			return &ErrHrmpMessagesOverflow{
				paraID:            id,
				messagesRemaining: outbound.MessagesRemaining,
				messagesSubmitted: outboundHrmpMod.MessagesSubmitted,
			}
		}
	}

	_, overflow := math.SafeSub(uint64(c.UmpRemaining), uint64(modifications.UmpMessagesSent))
	if overflow {
		return &ErrUmpMessagesOverflow{
			messagesRemaining: c.UmpRemaining,
			messagesSubmitted: modifications.UmpMessagesSent,
		}
	}

	_, overflow = math.SafeSub(uint64(c.UmpRemainingBytes), uint64(modifications.UmpBytesSent))
	if overflow {
		return &ErrUmpBytesOverflow{
			bytesRemaining: c.UmpRemainingBytes,
			bytesSubmitted: modifications.UmpBytesSent,
		}
	}

	_, overflow = math.SafeSub(uint64(len(c.DmpRemainingMessages)), uint64(modifications.DmpMessagesProcessed))
	if overflow {
		return &ErrDmpMessagesUnderflow{
			messagesRemaining: uint32(len(c.DmpRemainingMessages)),
			messagesProcessed: modifications.DmpMessagesProcessed,
		}
	}

	if c.FutureValidationCode == nil && modifications.CodeUpgradeApplied {
		return ErrAppliedNonexistentCodeUpgrade
	}

	return nil
}

func ApplyModifications(c *parachaintypes.Constraints, modifications *ConstraintModifications) (
	*parachaintypes.Constraints, error) {
	newConstraints := c.Clone()

	if modifications.RequiredParent != nil {
		newConstraints.RequiredParent = *modifications.RequiredParent
	}

	if modifications.HrmpWatermark != nil {
		pos, found := slices.BinarySearch(
			newConstraints.HrmpInbound.ValidWatermarks,
			modifications.HrmpWatermark.Watermark())

		if found {
			// Exact match, so this is OK in all cases.
			newConstraints.HrmpInbound.ValidWatermarks = newConstraints.HrmpInbound.ValidWatermarks[pos+1:]
		} else {
			switch modifications.HrmpWatermark.Type {
			case Head:
				// Updates to Head are always OK.
				newConstraints.HrmpInbound.ValidWatermarks = newConstraints.HrmpInbound.ValidWatermarks[pos:]
			case Trunk:
				// Trunk update landing on disallowed watermark is not OK.
				return nil, &ErrDisallowedHrmpWatermark{BlockNumber: modifications.HrmpWatermark.Block}
			}
		}
	}

	for id, outboundHrmpMod := range modifications.OutboundHrmp {
		outbound, ok := newConstraints.HrmpChannelsOut[id]
		if !ok {
			return nil, &ErrNoSuchHrmpChannel{id}
		}

		if outboundHrmpMod.BytesSubmitted > outbound.BytesRemaining {
			return nil, &ErrHrmpBytesOverflow{
				paraID:         id,
				bytesRemaining: outbound.BytesRemaining,
				bytesSubmitted: outboundHrmpMod.BytesSubmitted,
			}
		}

		if outboundHrmpMod.MessagesSubmitted > outbound.MessagesRemaining {
			return nil, &ErrHrmpMessagesOverflow{
				paraID:            id,
				messagesRemaining: outbound.MessagesRemaining,
				messagesSubmitted: outboundHrmpMod.MessagesSubmitted,
			}
		}

		outbound.BytesRemaining -= outboundHrmpMod.BytesSubmitted
		outbound.MessagesRemaining -= outboundHrmpMod.MessagesSubmitted
	}

	if modifications.UmpMessagesSent > newConstraints.UmpRemaining {
		return nil, &ErrUmpMessagesOverflow{
			messagesRemaining: newConstraints.UmpRemaining,
			messagesSubmitted: modifications.UmpMessagesSent,
		}
	}
	newConstraints.UmpRemaining -= modifications.UmpMessagesSent

	if modifications.UmpBytesSent > newConstraints.UmpRemainingBytes {
		return nil, &ErrUmpBytesOverflow{
			bytesRemaining: newConstraints.UmpRemainingBytes,
			bytesSubmitted: modifications.UmpBytesSent,
		}
	}
	newConstraints.UmpRemainingBytes -= modifications.UmpBytesSent

	if modifications.DmpMessagesProcessed > uint32(len(newConstraints.DmpRemainingMessages)) {
		return nil, &ErrDmpMessagesUnderflow{
			messagesRemaining: uint32(len(newConstraints.DmpRemainingMessages)),
			messagesProcessed: modifications.DmpMessagesProcessed,
		}
	} else {
		newConstraints.DmpRemainingMessages = newConstraints.DmpRemainingMessages[modifications.DmpMessagesProcessed:]
	}

	if modifications.CodeUpgradeApplied {
		if newConstraints.FutureValidationCode == nil {
			return nil, ErrAppliedNonexistentCodeUpgrade
		}

		newConstraints.ValidationCodeHash = newConstraints.FutureValidationCode.ValidationCodeHash
	}

	return newConstraints, nil
}

// OutboundHrmpChannelModification represents modifications to outbound HRMP channels.
type OutboundHrmpChannelModification struct {
	BytesSubmitted    uint32
	MessagesSubmitted uint32
}

// HrmpWatermarkUpdate represents an update to the HRMP Watermark.
type HrmpWatermarkUpdate struct {
	Type  HrmpWatermarkUpdateType
	Block uint
}

// HrmpWatermarkUpdateType defines the type of HrmpWatermarkUpdate.
type HrmpWatermarkUpdateType int

const (
	Head HrmpWatermarkUpdateType = iota
	Trunk
)

// Watermark returns the block number of the HRMP Watermark update.
func (h HrmpWatermarkUpdate) Watermark() uint {
	return h.Block
}

// ConstraintModifications represents modifications to constraints as a result of prospective candidates.
type ConstraintModifications struct {
	// The required parent head to build upon.
	RequiredParent *parachaintypes.HeadData
	// The new HRMP watermark.
	HrmpWatermark *HrmpWatermarkUpdate
	// Outbound HRMP channel modifications.
	OutboundHrmp map[parachaintypes.ParaID]OutboundHrmpChannelModification
	// The amount of UMP XCM messages sent. `UMPSignal` and separator are excluded.
	UmpMessagesSent uint32
	// The amount of UMP XCM bytes sent. `UMPSignal` and separator are excluded.
	UmpBytesSent uint32
	// The amount of DMP messages processed.
	DmpMessagesProcessed uint32
	// Whether a pending code upgrade has been applied.
	CodeUpgradeApplied bool
}

func (cm *ConstraintModifications) Clone() *ConstraintModifications {
	return &ConstraintModifications{
		RequiredParent:       cm.RequiredParent,
		HrmpWatermark:        cm.HrmpWatermark,
		OutboundHrmp:         maps.Clone(cm.OutboundHrmp),
		UmpMessagesSent:      cm.UmpMessagesSent,
		UmpBytesSent:         cm.UmpBytesSent,
		DmpMessagesProcessed: cm.DmpMessagesProcessed,
		CodeUpgradeApplied:   cm.CodeUpgradeApplied,
	}
}

// Identity returns the 'identity' modifications: these can be applied to
// any constraints and yield the exact same result.
func NewConstraintModificationsIdentity() *ConstraintModifications {
	return &ConstraintModifications{
		OutboundHrmp: make(map[parachaintypes.ParaID]OutboundHrmpChannelModification),
	}
}

// Stack stacks other modifications on top of these. This does no sanity-checking, so if
// `other` is garbage relative to `self`, then the new value will be garbage as well.
// This is an addition which is not commutative.
func (cm *ConstraintModifications) Stack(other *ConstraintModifications) {
	if other.RequiredParent != nil {
		cm.RequiredParent = other.RequiredParent
	}

	if other.HrmpWatermark != nil {
		cm.HrmpWatermark = other.HrmpWatermark
	}

	for id, mods := range other.OutboundHrmp {
		record, ok := cm.OutboundHrmp[id]
		if !ok {
			record = OutboundHrmpChannelModification{}
		}

		record.BytesSubmitted += mods.BytesSubmitted
		record.MessagesSubmitted += mods.MessagesSubmitted
		cm.OutboundHrmp[id] = record
	}

	cm.UmpMessagesSent += other.UmpMessagesSent
	cm.UmpBytesSent += other.UmpBytesSent
	cm.DmpMessagesProcessed += other.DmpMessagesProcessed
	cm.CodeUpgradeApplied = cm.CodeUpgradeApplied || other.CodeUpgradeApplied
}

// Fragment represents another prospective parachain block
// This is a type which guarantees that the candidate is valid under the operating constraints
type Fragment struct {
	relayParent          *RelayChainBlockInfo
	operatingConstraints *parachaintypes.Constraints
	candidate            *ProspectiveCandidate
	modifications        *ConstraintModifications
}

func (f *Fragment) RelayParent() *RelayChainBlockInfo {
	return f.relayParent
}

func (f *Fragment) Candidate() *ProspectiveCandidate {
	return f.candidate
}

func (f *Fragment) ConstraintModifications() *ConstraintModifications {
	return f.modifications
}

// NewFragment creates a new Fragment. This fails if the fragment isnt in line
// with the operating constraints. That is, either its inputs or outputs fail
// checks against the constraints.
// This does not check that the collator signature is valid or whether the PoV is
// small enough.
func NewFragment(
	relayParent *RelayChainBlockInfo,
	operatingConstraints *parachaintypes.Constraints,
	candidate *ProspectiveCandidate) (*Fragment, error) {

	modifications, err := CheckAgainstConstraints(
		relayParent,
		operatingConstraints,
		candidate.Commitments,
		candidate.ValidationCodeHash,
		candidate.PersistedValidationData,
	)
	if err != nil {
		return nil, err
	}

	return &Fragment{
		relayParent:          relayParent,
		operatingConstraints: operatingConstraints,
		candidate:            candidate,
		modifications:        modifications,
	}, nil
}

func CheckAgainstConstraints(
	relayParent *RelayChainBlockInfo,
	operatingConstraints *parachaintypes.Constraints,
	commitments parachaintypes.CandidateCommitments,
	validationCodeHash parachaintypes.ValidationCodeHash,
	persistedValidationData parachaintypes.PersistedValidationData,
) (*ConstraintModifications, error) {
	upwardMessages := make([]parachaintypes.UpwardMessage, 0)
	// filter UMP signals
	for upwardMessage := range skipUmpSignals(commitments.UpwardMessages) {
		upwardMessages = append(upwardMessages, upwardMessage)
	}

	umpMessagesSent := len(upwardMessages)
	umpBytesSent := 0
	for _, message := range upwardMessages {
		umpBytesSent += len(message)
	}

	hrmpWatermark := HrmpWatermarkUpdate{
		Type:  Trunk,
		Block: uint(commitments.HrmpWatermark),
	}

	if uint(commitments.HrmpWatermark) == relayParent.Number {
		hrmpWatermark.Type = Head
	}

	outboundHrmp := make(map[parachaintypes.ParaID]OutboundHrmpChannelModification)
	var lastRecipient *parachaintypes.ParaID

	for i, message := range commitments.HorizontalMessages {
		if lastRecipient != nil && *lastRecipient >= parachaintypes.ParaID(message.Recipient) {
			return nil, &ErrHrmpMessagesDescendingOrDuplicate{index: uint(i)}
		}

		recipientParaID := parachaintypes.ParaID(message.Recipient)
		lastRecipient = &recipientParaID
		record, ok := outboundHrmp[recipientParaID]
		if !ok {
			record = OutboundHrmpChannelModification{}
		}

		record.BytesSubmitted += uint32(len(message.Data))
		record.MessagesSubmitted++
		outboundHrmp[recipientParaID] = record
	}

	codeUpgradeApplied := false
	if operatingConstraints.FutureValidationCode != nil {
		codeUpgradeApplied = relayParent.Number >= operatingConstraints.FutureValidationCode.BlockNumber
	}

	modifications := &ConstraintModifications{
		RequiredParent:       &commitments.HeadData,
		HrmpWatermark:        &hrmpWatermark,
		OutboundHrmp:         outboundHrmp,
		UmpMessagesSent:      uint32(umpMessagesSent),
		UmpBytesSent:         uint32(umpBytesSent),
		DmpMessagesProcessed: commitments.ProcessedDownwardMessages,
		CodeUpgradeApplied:   codeUpgradeApplied,
	}

	err := validateAgainstConstraints(
		operatingConstraints,
		relayParent,
		commitments,
		persistedValidationData,
		validationCodeHash,
		modifications,
	)
	if err != nil {
		return nil, err
	}

	return modifications, nil
}

// UmpSeparator is a constant used to separate UMP signals.
var UmpSeparator = []byte{}

// skipUmpSignals is a utility function for skipping the UMP signals.
func skipUmpSignals(upwardMessages []parachaintypes.UpwardMessage) iter.Seq[parachaintypes.UpwardMessage] {
	return func(yield func(parachaintypes.UpwardMessage) bool) {
		for _, message := range upwardMessages {
			if !bytes.Equal([]byte(message), UmpSeparator) {
				if !yield([]byte(message)) {
					return
				}
			}

			return //nolint:staticcheck
		}
	}
}

func validateAgainstConstraints(
	constraints *parachaintypes.Constraints,
	relayParent *RelayChainBlockInfo,
	commitments parachaintypes.CandidateCommitments,
	persistedValidationData parachaintypes.PersistedValidationData,
	validationCodeHash parachaintypes.ValidationCodeHash,
	modifications *ConstraintModifications,
) error {
	expectedPVD := parachaintypes.PersistedValidationData{
		ParentHead:             constraints.RequiredParent,
		RelayParentNumber:      uint32(relayParent.Number),
		RelayParentStorageRoot: relayParent.StorageRoot,
		MaxPovSize:             constraints.MaxPoVSize,
	}

	if !expectedPVD.Equal(persistedValidationData) {
		return fmt.Errorf("%w, expected: %v, got: %v",
			ErrPersistedValidationDataMismatch, expectedPVD, persistedValidationData)
	}

	if constraints.ValidationCodeHash != validationCodeHash {
		return &ErrValidationCodeMismatch{
			expected: constraints.ValidationCodeHash,
			got:      validationCodeHash,
		}
	}

	if relayParent.Number < constraints.MinRelayParentNumber {
		return &ErrRelayParentTooOld{
			minAllowed: constraints.MinRelayParentNumber,
			current:    relayParent.Number,
		}
	}

	if commitments.NewValidationCode != nil {
		restriction, err := constraints.UpgradeRestriction.Value()
		if err != nil {
			return fmt.Errorf("while getting upgrade restriction: %w", err)
		}

		switch restriction.(type) {
		case *parachaintypes.Present:
			return ErrCodeUpgradeRestricted
		}
	}

	announcedCodeSize := 0
	if commitments.NewValidationCode != nil {
		announcedCodeSize = len(*commitments.NewValidationCode)
	}

	if uint32(announcedCodeSize) > constraints.MaxCodeSize {
		return &ErrCodeSizeTooLarge{
			maxAllowed: constraints.MaxCodeSize,
			newSize:    uint32(announcedCodeSize),
		}
	}

	if modifications.DmpMessagesProcessed == 0 {
		if len(constraints.DmpRemainingMessages) > 0 && constraints.DmpRemainingMessages[0] <= relayParent.Number {
			return ErrDmpAdvancementRule
		}
	}

	if len(commitments.HorizontalMessages) > int(constraints.MaxHrmpNumPerCandidate) {
		return &ErrHrmpMessagesPerCandidateOverflow{
			messagesAllowed:   constraints.MaxHrmpNumPerCandidate,
			messagesSubmitted: uint32(len(commitments.HorizontalMessages)),
		}
	}

	if modifications.UmpMessagesSent > constraints.MaxUmpNumPerCandidate {
		return &ErrUmpMessagesPerCandidateOverflow{
			messagesAllowed:   constraints.MaxUmpNumPerCandidate,
			messagesSubmitted: modifications.UmpMessagesSent,
		}
	}

	if err := CheckModifications(constraints, modifications); err != nil {
		return &ErrOutputsInvalid{ModificationError: err}
	}

	return nil
}
