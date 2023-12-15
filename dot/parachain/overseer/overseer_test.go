// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package overseer

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	types "github.com/ChainSafe/gossamer/dot/types"
	gomock "github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type TestSubsystem struct {
	name string
}

func (s *TestSubsystem) Name() parachaintypes.SubSystemName {
	return parachaintypes.SubSystemName(s.name)
}

func (s *TestSubsystem) Run(ctx context.Context, OverseerToSubSystem chan any, SubSystemToOverseer chan any) error {
	fmt.Printf("%s run\n", s.name)
	counter := 0
	for {
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				fmt.Printf("%s ctx error: %v\n", s.name, err)
			}
			fmt.Printf("%s overseer stopping\n", s.name)
			return nil
		case overseerSignal := <-OverseerToSubSystem:
			fmt.Printf("%s received from overseer %v\n", s.name, overseerSignal)
		default:
			// simulate work, and sending messages to overseer
			r := rand.Intn(1000)
			time.Sleep(time.Duration(r) * time.Millisecond)
			SubSystemToOverseer <- fmt.Sprintf("hello from %v, i: %d", s.name, counter)
			counter++
		}
	}
}

func (s *TestSubsystem) ProcessOverseerSignals() {
	fmt.Printf("%s ProcessOverseerSignals\n", s.name)
}

func (s *TestSubsystem) String() parachaintypes.SubSystemName {
	return parachaintypes.SubSystemName(s.name)
}

func (s *TestSubsystem) Stop() {}

func TestStart2SubsytemsActivate1(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)
	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseerToSubSystem1 := overseer.RegisterSubsystem(subSystem1)
	overseerToSubSystem2 := overseer.RegisterSubsystem(subSystem2)

	go func() {
		<-overseerToSubSystem1
		<-overseerToSubSystem2
	}()

	err := overseer.Start()
	require.NoError(t, err)

	done := make(chan struct{})
	// listen for errors from overseer
	go func() {
		for errC := range overseer.errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	time.Sleep(1000 * time.Millisecond)
	activedLeaf := ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdateSignal{Activated: &activedLeaf}, subSystem1)

	// let subsystems run for a bit
	time.Sleep(4000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}

func TestStart2SubsytemsActivate2Different(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)
	overseer := NewOverseer(blockState)
	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseerToSubSystem1 := overseer.RegisterSubsystem(subSystem1)
	overseerToSubSystem2 := overseer.RegisterSubsystem(subSystem2)

	go func() {
		<-overseerToSubSystem1
		<-overseerToSubSystem2
	}()

	err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range overseer.errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	activedLeaf1 := ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	activedLeaf2 := ActivatedLeaf{
		Hash:   [32]byte{2},
		Number: 2,
	}
	time.Sleep(250 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdateSignal{Activated: &activedLeaf1}, subSystem1)
	time.Sleep(400 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdateSignal{Activated: &activedLeaf2}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(3000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}

func TestStart2SubsytemsActivate2Same(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)
	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseerToSubSystem1 := overseer.RegisterSubsystem(subSystem1)
	overseerToSubSystem2 := overseer.RegisterSubsystem(subSystem2)

	go func() {
		<-overseerToSubSystem1
		<-overseerToSubSystem2
	}()

	err := overseer.Start()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		for errC := range overseer.errChan {
			fmt.Printf("overseer start error: %v\n", errC)
		}
		close(done)
	}()

	activedLeaf := ActivatedLeaf{
		Hash:   [32]byte{1},
		Number: 1,
	}
	time.Sleep(300 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdateSignal{Activated: &activedLeaf}, subSystem1)
	time.Sleep(400 * time.Millisecond)
	overseer.sendActiveLeavesUpdate(ActiveLeavesUpdateSignal{Activated: &activedLeaf}, subSystem2)
	// let subsystems run for a bit
	time.Sleep(2000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	fmt.Printf("overseer stopped\n")
	<-done
}

func TestHandleBlockEvents(t *testing.T) {
	ctrl := gomock.NewController(t)

	blockState := NewMockBlockState(ctrl)

	finalizedNotifierChan := make(chan *types.FinalisationInfo)
	importedBlockNotiferChan := make(chan *types.Block)

	blockState.EXPECT().GetFinalisedNotifierChannel().Return(finalizedNotifierChan)
	blockState.EXPECT().GetImportedBlockNotifierChannel().Return(importedBlockNotiferChan)
	blockState.EXPECT().FreeFinalisedNotifierChannel(finalizedNotifierChan)
	blockState.EXPECT().FreeImportedBlockNotifierChannel(importedBlockNotiferChan)

	overseer := NewOverseer(blockState)

	require.NotNil(t, overseer)

	subSystem1 := &TestSubsystem{name: "subSystem1"}
	subSystem2 := &TestSubsystem{name: "subSystem2"}

	overseerToSubSystem1 := overseer.RegisterSubsystem(subSystem1)
	overseerToSubSystem2 := overseer.RegisterSubsystem(subSystem2)

	var finalizedCounter atomic.Int32
	var importedCounter atomic.Int32

	go func() {
		for {
			select {
			case msg := <-overseerToSubSystem1:
				if msg == nil {
					continue
				}

				_, ok := msg.(BlockFinalizedSignal)
				if ok {
					finalizedCounter.Add(1)
				}

				_, ok = msg.(ActiveLeavesUpdateSignal)
				if ok {
					importedCounter.Add(1)
				}
			case msg := <-overseerToSubSystem2:
				if msg == nil {
					continue
				}

				_, ok := msg.(BlockFinalizedSignal)
				if ok {
					finalizedCounter.Add(1)
				}

				_, ok = msg.(ActiveLeavesUpdateSignal)
				if ok {
					importedCounter.Add(1)
				}
			}

		}
	}()

	err := overseer.Start()
	require.NoError(t, err)
	finalizedNotifierChan <- &types.FinalisationInfo{}
	importedBlockNotiferChan <- &types.Block{}

	time.Sleep(1000 * time.Millisecond)

	err = overseer.Stop()
	require.NoError(t, err)

	require.Equal(t, int32(2), finalizedCounter.Load())
	require.Equal(t, int32(2), importedCounter.Load())
}
