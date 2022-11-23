// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/internal/trie/tracking"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

var (
	// ErrDecodeStorageValue is defined since no sentinel error is defined
	// in the scale package.
	// TODO remove once the following issue is done:
	// https://github.com/ChainSafe/gossamer/issues/2631 .
	ErrDecodeStorageValue = errors.New("cannot decode storage value")
	ErrReadChildrenBitmap = errors.New("cannot read children bitmap")
	// ErrDecodeChildHash is defined since no sentinel error is defined
	// in the scale package.
	// TODO remove once the following issue is done:
	// https://github.com/ChainSafe/gossamer/issues/2631 .
	ErrDecodeChildHash = errors.New("cannot decode child hash")
)

// Decode decodes a node from a reader.
// The encoding format is documented in the README.md
// of this package, and specified in the Polkadot spec at
// https://spec.polkadot.network/#sect-state-storage
// For branch decoding, see the comments on decodeBranch.
// For leaf decoding, see the comments on decodeLeaf.
func Decode(reader io.Reader, database Putter) (n *Node, err error) {
	variant, partialKeyLength, err := decodeHeader(reader)
	if err != nil {
		return nil, fmt.Errorf("decoding header: %w", err)
	}

	switch variant {
	case leafVariant.bits:
		n, err = decodeLeaf(reader, partialKeyLength, database)
		if err != nil {
			return nil, fmt.Errorf("cannot decode leaf: %w", err)
		}
		return n, nil
	case branchVariant.bits, branchWithValueVariant.bits:
		n, err = decodeBranch(reader, variant, partialKeyLength, database)
		if err != nil {
			return nil, fmt.Errorf("cannot decode branch: %w", err)
		}
		return n, nil
	default:
		// this is a programming error, an unknown node variant
		// should be caught by decodeHeader.
		panic(fmt.Sprintf("not implemented for node variant %08b", variant))
	}
}

var noopDeltaTracker = tracking.NewNoop()

// decodeBranch reads from a reader and decodes to a node branch.
// Note that since the encoded branch stores the hash of the children nodes, we are not
// reconstructing the child nodes from the encoding. This function instead stubs where the
// children are known to be with an empty leaf. The children nodes hashes are then used to
// find other storage values using the persistent database.
func decodeBranch(reader io.Reader, variant byte, partialKeyLength uint16,
	database Putter) (node *Node, err error) {
	node = &Node{
		Children: make([]*Node, ChildrenCapacity),
	}

	node.PartialKey, err = decodeKey(reader, partialKeyLength)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	childrenBitmap := make([]byte, 2)
	_, err = reader.Read(childrenBitmap)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrReadChildrenBitmap, err)
	}

	sd := scale.NewDecoder(reader)

	if variant == branchWithValueVariant.bits {
		var storageValue []byte
		err := sd.Decode(&storageValue)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
		}

		// node storage value is nil so there is no need to track deltas.
		err = node.SetStorageValue(storageValue, database, noopDeltaTracker)
		if err != nil {
			return nil, fmt.Errorf("setting storage value: %w", err)
		}
	}

	for i := 0; i < ChildrenCapacity; i++ {
		if (childrenBitmap[i/8]>>(i%8))&1 != 1 {
			continue
		}

		var hash []byte
		err := sd.Decode(&hash)
		if err != nil {
			return nil, fmt.Errorf("%w: at index %d: %s",
				ErrDecodeChildHash, i, err)
		}

		const hashLength = 32
		childNode := &Node{
			MerkleValue: hash,
		}
		if len(hash) < hashLength {
			// Handle inlined nodes
			reader = bytes.NewReader(hash)
			childNode, err = Decode(reader, database)
			if err != nil {
				return nil, fmt.Errorf("decoding inlined child at index %d: %w", i, err)
			}
			node.Descendants += childNode.Descendants
		}

		node.Descendants++
		node.Children[i] = childNode
	}

	return node, nil
}

// decodeLeaf reads from a reader and decodes to a leaf node.
func decodeLeaf(reader io.Reader, partialKeyLength uint16, database Putter) (
	node *Node, err error) {
	node = &Node{}

	node.PartialKey, err = decodeKey(reader, partialKeyLength)
	if err != nil {
		return nil, fmt.Errorf("cannot decode key: %w", err)
	}

	sd := scale.NewDecoder(reader)
	var storageValue []byte
	err = sd.Decode(&storageValue)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrDecodeStorageValue, err)
	}

	err = node.SetStorageValue(storageValue, database, noopDeltaTracker)
	if err != nil {
		return nil, fmt.Errorf("setting node storage value: %w", err)
	}

	return node, nil
}
