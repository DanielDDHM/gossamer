// Copyright 2020 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package grandpa

import (
	"sync"

	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

// tracker keeps track of messages that have been received that have failed to validate with ErrBlockDoesNotExist
// these messages may be needed again in the case that we are slightly out of sync with the rest of the network
type tracker struct {
	blockState BlockState
	messages   map[common.Hash][]*networkVoteMessage // map of vote block hash -> array of VoteMessages for that hash
	mapLock    sync.Mutex
	in         chan *types.Block          // receive imported block from BlockState
	out        chan<- *networkVoteMessage // send a VoteMessage back to grandpa. corresponds to grandpa's in channel
	stopped    chan struct{}
}

func newTracker(bs BlockState, out chan<- *networkVoteMessage) (*tracker, error) {
	//in := make(chan *types.Block, 16)
	in, err := bs.GetNotifierChannel()
	// TODO (ed) import channel is unregistered and closed in tracker.stop()
	// todo ed, remove, and get channel makes channel 100, not 16
	//id, err := bs.RegisterImportedChannel(in)
	if err != nil {
		return nil, err
	}

	return &tracker{
		blockState: bs,
		messages:   make(map[common.Hash][]*networkVoteMessage),
		mapLock:    sync.Mutex{},
		in:         in,
		out:        out,
		stopped:    make(chan struct{}),
	}, nil
}

func (t *tracker) start() {
	go t.handleBlocks()
}

func (t *tracker) stop() {
	close(t.stopped)
	// todo ed remave and move close
	//t.blockState.UnregisterImportedChannel(t.chanID)
	t.blockState.FreeNotifierChannel(t.in)
	close(t.in)
}

func (t *tracker) add(v *networkVoteMessage) {
	if v.msg == nil {
		return
	}

	t.mapLock.Lock()
	// TODO: change to map of maps, this allows duplicates
	t.messages[v.msg.Message.Hash] = append(t.messages[v.msg.Message.Hash], v)
	t.mapLock.Unlock()
}

func (t *tracker) handleBlocks() {
	for {
		select {
		case b := <-t.in:
			if b == nil {
				continue
			}

			t.mapLock.Lock()

			h := b.Header.Hash()
			if t.messages[h] != nil {
				for _, v := range t.messages[h] {
					t.out <- v
				}
			}

			t.mapLock.Unlock()
		case <-t.stopped:
			return
		}
	}
}
