// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"math/big"
	"testing"
	"time"

	"github.com/endurio/ndrd/chaincfg"
)

func TestBigToCompact(t *testing.T) {
	tests := []struct {
		in  int64
		out uint32
	}{
		{0, 0},
		{-1, 25231360},
	}

	for x, test := range tests {
		n := big.NewInt(test.in)
		r := BigToCompact(n)
		if r != test.out {
			t.Errorf("TestBigToCompact test #%d failed: got %d want %d\n",
				x, r, test.out)
			return
		}
	}
}

func TestCompactToBig(t *testing.T) {
	tests := []struct {
		in  uint32
		out int64
	}{
		{10000000, 0},
	}

	for x, test := range tests {
		n := CompactToBig(test.in)
		want := big.NewInt(test.out)
		if n.Cmp(want) != 0 {
			t.Errorf("TestCompactToBig test #%d failed: got %d want %d\n",
				x, n.Int64(), want.Int64())
			return
		}
	}
}

func TestCalcWork(t *testing.T) {
	tests := []struct {
		in  uint32
		out int64
	}{
		{10000000, 0},
	}

	for x, test := range tests {
		bits := uint32(test.in)

		r := CalcWork(bits)
		if r.Int64() != test.out {
			t.Errorf("TestCalcWork test #%d failed: got %v want %d\n",
				x, r.Int64(), test.out)
			return
		}
	}
}

// TestMinDifficultyReduction ensures the code which results in reducing the
// minimum required difficulty, when the network params allow it, works as
// expected.
func TestMinDifficultyReduction(t *testing.T) {
	// Create chain params based on regnet params, but set the fields related to
	// proof-of-work difficulty to specific values expected by the tests.
	params := chaincfg.RegNetParams
	params.ReduceMinDifficulty = true
	params.TargetTimePerBlock = time.Minute * 2
	params.MinDiffReductionTime = time.Minute * 10 // ~99.3% chance to be mined
	params.WorkDiffAlpha = 1
	params.WorkDiffWindowSize = 144
	params.WorkDiffWindows = 20
	params.TargetTimespan = params.TargetTimePerBlock *
		time.Duration(params.WorkDiffWindowSize)
	params.RetargetAdjustmentFactor = 4

	tests := []struct {
		name           string
		timeAdjustment func(i int) time.Duration
		numBlocks      int64
		expectedDiff   func(i int) uint32
	}{
		{
			name:           "genesis block",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      1,
			expectedDiff:   func(i int) uint32 { return params.PowLimitBits },
		},
		{
			name:           "create difficulty spike - part 1",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize - 2,
			expectedDiff:   func(i int) uint32 { return 545259519 },
		},
		{
			name:           "create difficulty spike - part 2",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 545259519 },
		},
		{
			name:           "create difficulty spike - part 3",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 541100164 },
		},
		{
			name:           "create difficulty spike - part 4",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 537954654 },
		},
		{
			name:           "create difficulty spike - part 5",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 537141847 },
		},
		{
			name:           "create difficulty spike - part 6",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 536938645 },
		},
		{
			name:           "create difficulty spike - part 7",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 524428608 },
		},
		{
			name:           "create difficulty spike - part 8",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 521177424 },
		},
		{
			name:           "create difficulty spike - part 9",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 520364628 },
		},
		{
			name:           "create difficulty spike - part 10",
			timeAdjustment: func(i int) time.Duration { return time.Second },
			numBlocks:      params.WorkDiffWindowSize,
			expectedDiff:   func(i int) uint32 { return 520161429 },
		},
		{
			name: "alternate min diff blocks",
			timeAdjustment: func(i int) time.Duration {
				if i%2 == 0 {
					return params.MinDiffReductionTime + time.Second
				}
				return params.TargetTimePerBlock
			},
			numBlocks: params.WorkDiffWindowSize,
			expectedDiff: func(i int) uint32 {
				if i%2 == 0 && i != 0 {
					return params.PowLimitBits
				}
				return 507651392
			},
		},
		{
			name: "interval of blocks taking twice the target time - part 1",
			timeAdjustment: func(i int) time.Duration {
				return params.TargetTimePerBlock * 2
			},
			numBlocks:    params.WorkDiffWindowSize,
			expectedDiff: func(i int) uint32 { return 509850141 },
		},
		{
			name: "interval of blocks taking twice the target time - part 2",
			timeAdjustment: func(i int) time.Duration {
				return params.TargetTimePerBlock * 2
			},
			numBlocks:    params.WorkDiffWindowSize,
			expectedDiff: func(i int) uint32 { return 520138451 },
		},
		{
			name: "interval of blocks taking twice the target time - part 3",
			timeAdjustment: func(i int) time.Duration {
				return params.TargetTimePerBlock * 2
			},
			numBlocks:    params.WorkDiffWindowSize,
			expectedDiff: func(i int) uint32 { return 520177692 },
		},
	}

	bc := newFakeChain(&params)
	node := bc.bestChain.Tip()
	blockTime := time.Unix(node.timestamp, 0)
	for _, test := range tests {
		for i := 0; i < int(test.numBlocks); i++ {
			// Update the block time according to the test data and calculate
			// the difficulty for the next block.
			blockTime = blockTime.Add(test.timeAdjustment(i))
			diff, err := bc.calcNextRequiredDifficulty(node, blockTime)
			if err != nil {
				t.Fatalf("calcNextRequiredDifficulty: unexpected err: %v", err)
			}

			// Ensure the calculated difficulty matches the expected value.
			expectedDiff := test.expectedDiff(i)
			if diff != expectedDiff {
				t.Fatalf("calcNextRequiredDifficulty (%s): did not get "+
					"expected difficulty -- got %d, want %d", test.name, diff,
					expectedDiff)
			}

			node = newFakeNode(node, 1, 1, diff, blockTime)
			bc.index.AddNode(node)
			bc.bestChain.SetTip(node)
		}
	}
}
