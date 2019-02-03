// Copyright (c) 2016-2018 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaingen

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"time"

	"github.com/endurio/ndrd/chaincfg"
	"github.com/endurio/ndrd/chaincfg/chainhash"
	"github.com/endurio/ndrd/ndrutil"
	"github.com/endurio/ndrd/txscript"
	"github.com/endurio/ndrd/wire"
)

var (
	// hash256prngSeedConst is a constant derived from the hex
	// representation of pi and is used in conjunction with a caller-provided
	// seed when initializing the deterministic lottery prng.
	hash256prngSeedConst = []byte{0x24, 0x3f, 0x6a, 0x88, 0x85, 0xa3, 0x08,
		0xd3}

	// opTrueScript is a simple public key script that contains the OP_TRUE
	// opcode.  It is defined here to reduce garbage creation.
	opTrueScript = []byte{txscript.OP_TRUE}

	// opTrueRedeemScript is the signature script that can be used to redeem
	// a p2sh output to the opTrueScript.  It is defined here to reduce
	// garbage creation.
	opTrueRedeemScript = []byte{txscript.OP_DATA_1, txscript.OP_TRUE}

	// coinbaseSigScript is the signature script used by the tests when
	// creating standard coinbase transactions.  It is defined here to
	// reduce garbage creation.
	coinbaseSigScript = []byte{txscript.OP_0, txscript.OP_0}
)

const (
	// voteBitYes is the specific bit that is set in the vote bits to
	// indicate that the previous block is valid.
	voteBitYes = 0x01
)

// SpendableOut represents a transaction output that is spendable along with
// additional metadata such as the block its in and how much it pays.
type SpendableOut struct {
	prevOut     wire.OutPoint
	blockHeight uint32
	blockIndex  uint32
	amount      ndrutil.Amount
}

// PrevOut returns the outpoint associated with the spendable output.
func (s *SpendableOut) PrevOut() wire.OutPoint {
	return s.prevOut
}

// BlockHeight returns the block height of the block the spendable output is in.
func (s *SpendableOut) BlockHeight() uint32 {
	return s.blockHeight
}

// BlockIndex returns the offset into the block the spendable output is in.
func (s *SpendableOut) BlockIndex() uint32 {
	return s.blockIndex
}

// Amount returns the amount associated with the spendable output.
func (s *SpendableOut) Amount() ndrutil.Amount {
	return s.amount
}

// makeSpendableOutForTxInternal returns a spendable output for the given
// transaction block height, transaction index within the block, transaction
// tree and transaction output index within the transaction.
func makeSpendableOutForTxInternal(tx *wire.MsgTx, blockHeight, txIndex, txOutIndex uint32) SpendableOut {
	return SpendableOut{
		prevOut: wire.OutPoint{
			Hash:  *tx.CachedTxHash(),
			Index: txOutIndex,
		},
		blockHeight: blockHeight,
		blockIndex:  txIndex,
		amount:      ndrutil.Amount(tx.TxOut[txOutIndex].Value),
	}
}

// MakeSpendableOutForTx returns a spendable output for a regular transaction.
func MakeSpendableOutForTx(tx *wire.MsgTx, blockHeight, txIndex, txOutIndex uint32) SpendableOut {
	return makeSpendableOutForTxInternal(tx, blockHeight, txIndex, txOutIndex)
}

// MakeSpendableOutForSTx returns a spendable output for a stake transaction.
func MakeSpendableOutForSTx(tx *wire.MsgTx, blockHeight, txIndex, txOutIndex uint32) SpendableOut {
	return makeSpendableOutForTxInternal(tx, blockHeight, txIndex, txOutIndex)
}

// MakeSpendableOut returns a spendable output for the given block, transaction
// index within the block, and transaction output index within the transaction.
func MakeSpendableOut(block *wire.MsgBlock, txIndex, txOutIndex uint32) SpendableOut {
	tx := block.Transactions[txIndex]
	return MakeSpendableOutForTx(tx, block.Header.Height, txIndex, txOutIndex)
}

// Generator houses state used to ease the process of generating test blocks
// that build from one another along with housing other useful things such as
// available spendable outputs and generic payment scripts used throughout the
// tests.
type Generator struct {
	params           *chaincfg.Params
	tip              *wire.MsgBlock
	tipName          string
	blocks           map[chainhash.Hash]*wire.MsgBlock
	blockHeights     map[chainhash.Hash]uint32
	blocksByName     map[string]*wire.MsgBlock
	p2shOpTrueAddr   ndrutil.Address
	p2shOpTrueScript []byte

	// Used for tracking spendable coinbase outputs.
	spendableOuts     [][]SpendableOut
	prevCollectedHash chainhash.Hash

	// Used for tracking the live ticket pool and revocations.
	originalParents map[chainhash.Hash]chainhash.Hash
}

// MakeGenerator returns a generator instance initialized with the genesis block
// as the tip as well as a cached generic pay-to-script-hash script for OP_TRUE.
func MakeGenerator(params *chaincfg.Params) (Generator, error) {
	// Generate a generic pay-to-script-hash script that is a simple
	// OP_TRUE.  This allows the tests to avoid needing to generate and
	// track actual public keys and signatures.
	p2shOpTrueAddr, err := ndrutil.NewAddressScriptHash(opTrueScript, params)
	if err != nil {
		return Generator{}, err
	}
	p2shOpTrueScript, err := txscript.PayToAddrScript(p2shOpTrueAddr)
	if err != nil {
		return Generator{}, err
	}

	genesis := params.GenesisBlock
	genesisHash := genesis.BlockHash()
	return Generator{
		params:           params,
		tip:              genesis,
		tipName:          "genesis",
		blocks:           map[chainhash.Hash]*wire.MsgBlock{genesisHash: genesis},
		blockHeights:     map[chainhash.Hash]uint32{genesis.BlockHash(): 0},
		blocksByName:     map[string]*wire.MsgBlock{"genesis": genesis},
		p2shOpTrueAddr:   p2shOpTrueAddr,
		p2shOpTrueScript: p2shOpTrueScript,
		originalParents:  make(map[chainhash.Hash]chainhash.Hash),
	}, nil
}

// Params returns the chain params associated with the generator instance.
func (g *Generator) Params() *chaincfg.Params {
	return g.params
}

// Tip returns the current tip block of the generator instance.
func (g *Generator) Tip() *wire.MsgBlock {
	return g.tip
}

// TipName returns the name of the current tip block of the generator instance.
func (g *Generator) TipName() string {
	return g.tipName
}

// P2shOpTrueAddr returns the generator p2sh script that is composed with
// a single OP_TRUE.
func (g *Generator) P2shOpTrueAddr() ndrutil.Address {
	return g.p2shOpTrueAddr
}

// BlockByName returns the block associated with the provided block name.  It
// will panic if the specified block name does not exist.
func (g *Generator) BlockByName(blockName string) *wire.MsgBlock {
	block, ok := g.blocksByName[blockName]
	if !ok {
		panic(fmt.Sprintf("block name %s does not exist", blockName))
	}
	return block
}

// BlockByHash returns the block associated with the provided block hash.  It
// will panic if the specified block hash does not exist.
func (g *Generator) BlockByHash(hash *chainhash.Hash) *wire.MsgBlock {
	block, ok := g.blocks[*hash]
	if !ok {
		panic(fmt.Sprintf("block with hash %s does not exist", hash))
	}
	return block
}

// blockHeight returns the block height associated with the provided block hash.
// It will panic if the specified block hash does not exist.
func (g *Generator) blockHeight(hash chainhash.Hash) uint32 {
	height, ok := g.blockHeights[hash]
	if !ok {
		panic(fmt.Sprintf("no block height found for block %s", hash))
	}
	return height
}

// opReturnScript returns a provably-pruneable OP_RETURN script with the
// provided data.
func opReturnScript(data []byte) []byte {
	builder := txscript.NewScriptBuilder()
	script, err := builder.AddOp(txscript.OP_RETURN).AddData(data).Script()
	if err != nil {
		panic(err)
	}
	return script
}

// UniqueOpReturnScript returns a standard provably-pruneable OP_RETURN script
// with a random uint64 encoded as the data.
func UniqueOpReturnScript() []byte {
	rand, err := wire.RandomUint64()
	if err != nil {
		panic(err)
	}

	data := make([]byte, 8)
	binary.LittleEndian.PutUint64(data[0:8], rand)
	return opReturnScript(data)
}

// calcFullSubsidy returns the full block subsidy for the given block height.
//
// NOTE: This and the other subsidy calculation funcs intentionally are not
// using the blockchain code since the intent is to be able to generate known
// good tests which exercise that code, so it wouldn't make sense to use the
// same code to generate them.
func (g *Generator) calcFullSubsidy(blockHeight uint32) ndrutil.Amount {
	iterations := int64(blockHeight) / g.params.SubsidyReductionInterval
	subsidy := g.params.BaseSubsidy
	for i := int64(0); i < iterations; i++ {
		subsidy *= g.params.MulSubsidy
		subsidy /= g.params.DivSubsidy
	}
	return ndrutil.Amount(subsidy)
}

// calcPoWSubsidy returns the proof-of-work subsidy portion from a given full
// subsidy, block height, and number of votes that will be included in the
// block.
//
// NOTE: This and the other subsidy calculation funcs intentionally are not
// using the blockchain code since the intent is to be able to generate known
// good tests which exercise that code, so it wouldn't make sense to use the
// same code to generate them.
func (g *Generator) calcPoWSubsidy(fullSubsidy ndrutil.Amount, blockHeight uint32, numVotes uint16) ndrutil.Amount {
	powProportion := ndrutil.Amount(g.params.WorkRewardProportion)
	totalProportions := ndrutil.Amount(g.params.TotalSubsidyProportions())
	powSubsidy := (fullSubsidy * powProportion) / totalProportions
	return powSubsidy
}

// calcPoSSubsidy returns the proof-of-stake subsidy portion for a given block
// height being voted on.
//
// NOTE: This and the other subsidy calculation funcs intentionally are not
// using the blockchain code since the intent is to be able to generate known
// good tests which exercise that code, so it wouldn't make sense to use the
// same code to generate them.
func (g *Generator) calcPoSSubsidy(heightVotedOn uint32) ndrutil.Amount {
	if int64(heightVotedOn+1) < g.params.StakeValidationHeight {
		return 0
	}

	fullSubsidy := g.calcFullSubsidy(heightVotedOn)
	posProportion := ndrutil.Amount(g.params.StakeRewardProportion)
	totalProportions := ndrutil.Amount(g.params.TotalSubsidyProportions())
	return (fullSubsidy * posProportion) / totalProportions
}

// calcDevSubsidy returns the dev org subsidy portion from a given full subsidy.
//
// NOTE: This and the other subsidy calculation funcs intentionally are not
// using the blockchain code since the intent is to be able to generate known
// good tests which exercise that code, so it wouldn't make sense to use the
// same code to generate them.
func (g *Generator) calcDevSubsidy(fullSubsidy ndrutil.Amount, blockHeight uint32, numVotes uint16) ndrutil.Amount {
	devProportion := ndrutil.Amount(g.params.BlockTaxProportion)
	totalProportions := ndrutil.Amount(g.params.TotalSubsidyProportions())
	devSubsidy := (fullSubsidy * devProportion) / totalProportions
	return devSubsidy
}

// standardCoinbaseOpReturnScript returns a standard script suitable for use as
// the second output of a standard coinbase transaction of a new block.  In
// particular, the serialized data used with the OP_RETURN starts with the block
// height and is followed by 32 bytes which are treated as 4 uint64 extra
// nonces.  This implementation puts a cryptographically random value into the
// final extra nonce position.  The actual format of the data after the block
// height is not defined however this effectively mirrors the actual mining code
// at the time it was written.
func standardCoinbaseOpReturnScript(blockHeight uint32) []byte {
	rand, err := wire.RandomUint64()
	if err != nil {
		panic(err)
	}

	data := make([]byte, 36)
	binary.LittleEndian.PutUint32(data[0:4], blockHeight)
	binary.LittleEndian.PutUint64(data[28:36], rand)
	return opReturnScript(data)
}

// addCoinbaseTxOutputs adds the following outputs to the provided transaction
// which is assumed to be a coinbase transaction:
// - First output pays the development subsidy portion to the dev org
// - Second output is a standard provably prunable data-only coinbase output
// - Third and subsequent outputs pay the pow subsidy portion to the generic
//   OP_TRUE p2sh script hash
func (g *Generator) addCoinbaseTxOutputs(tx *wire.MsgTx, blockHeight uint32, devSubsidy, powSubsidy ndrutil.Amount) {
	// First output is the developer subsidy.
	tx.AddTxOut(&wire.TxOut{
		Value:    int64(devSubsidy),
		Version:  g.params.OrganizationPkScriptVersion,
		PkScript: g.params.OrganizationPkScript,
	})

	// Second output is a provably prunable data-only output that is used
	// to ensure the coinbase is unique.
	tx.AddTxOut(wire.NewTxOut(0, standardCoinbaseOpReturnScript(blockHeight)))

	// Final outputs are the proof-of-work subsidy split into more than one
	// output.  These are in turn used throughout the tests as inputs to
	// other transactions such as ticket purchases and additional spend
	// transactions.
	const numPoWOutputs = 6
	amount := powSubsidy / numPoWOutputs
	for i := 0; i < numPoWOutputs; i++ {
		if i == numPoWOutputs-1 {
			amount = powSubsidy - amount*(numPoWOutputs-1)
		}
		tx.AddTxOut(wire.NewTxOut(int64(amount), g.p2shOpTrueScript))
	}
}

// CreateCoinbaseTx returns a coinbase transaction paying an appropriate
// subsidy based on the passed block height and number of votes to the dev org
// and proof-of-work miner.
//
// See the addCoinbaseTxOutputs documentation for a breakdown of the outputs
// the transaction contains.
func (g *Generator) CreateCoinbaseTx(blockHeight uint32, numVotes uint16) *wire.MsgTx {
	// Calculate the subsidy proportions based on the block height and the
	// number of votes the block will include.
	fullSubsidy := g.calcFullSubsidy(blockHeight)
	devSubsidy := g.calcDevSubsidy(fullSubsidy, blockHeight, numVotes)
	powSubsidy := g.calcPoWSubsidy(fullSubsidy, blockHeight, numVotes)

	tx := wire.NewMsgTx()
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex, wire.TxTreeRegular),
		Sequence:        wire.MaxTxInSequenceNum,
		ValueIn:         int64(devSubsidy + powSubsidy),
		BlockHeight:     wire.NullBlockHeight,
		BlockIndex:      wire.NullBlockIndex,
		SignatureScript: coinbaseSigScript,
	})

	g.addCoinbaseTxOutputs(tx, blockHeight, devSubsidy, powSubsidy)

	return tx
}

// ancestorBlock returns the ancestor block at the provided height by following
// the chain backwards from the given block.  The returned block will be nil
// when a height is requested that is after the height of the passed block.
// Also, a callback can optionally be provided that is invoked with each block
// as it traverses.
func (g *Generator) ancestorBlock(block *wire.MsgBlock, height uint32, f func(*wire.MsgBlock)) *wire.MsgBlock {
	// Nothing to do if the requested height is outside of the valid
	// range.
	if block == nil || height > block.Header.Height {
		return nil
	}

	// Iterate backwards until the requested height is reached.
	for block != nil && block.Header.Height > height {
		block = g.blocks[block.Header.PrevBlock]
		if f != nil && block != nil {
			f(block)
		}
	}

	return block
}

// mergeDifficulty takes an original stake difficulty and two new, scaled
// stake difficulties, merges the new difficulties, and outputs a new
// merged stake difficulty.
func mergeDifficulty(oldDiff int64, newDiff1 int64, newDiff2 int64) int64 {
	newDiff1Big := big.NewInt(newDiff1)
	newDiff2Big := big.NewInt(newDiff2)
	newDiff2Big.Lsh(newDiff2Big, 32)

	oldDiffBig := big.NewInt(oldDiff)
	oldDiffBigLSH := big.NewInt(oldDiff)
	oldDiffBigLSH.Lsh(oldDiffBig, 32)

	newDiff1Big.Div(oldDiffBigLSH, newDiff1Big)
	newDiff2Big.Div(newDiff2Big, oldDiffBig)

	// Combine the two changes in difficulty.
	summedChange := big.NewInt(0)
	summedChange.Set(newDiff2Big)
	summedChange.Lsh(summedChange, 32)
	summedChange.Div(summedChange, newDiff1Big)
	summedChange.Mul(summedChange, oldDiffBig)
	summedChange.Rsh(summedChange, 32)

	return summedChange.Int64()
}

// limitRetarget clamps the passed new difficulty to the old one adjusted by the
// factor specified in the chain parameters.  This ensures the difficulty can
// only move up or down by a limited amount.
func (g *Generator) limitRetarget(oldDiff, newDiff int64) int64 {
	maxRetarget := g.params.RetargetAdjustmentFactor
	switch {
	case newDiff == 0:
		fallthrough
	case (oldDiff / newDiff) > (maxRetarget - 1):
		return oldDiff / maxRetarget
	case (newDiff / oldDiff) > (maxRetarget - 1):
		return oldDiff * maxRetarget
	}

	return newDiff
}

// CalcNextRequiredDifficulty returns the required proof-of-work difficulty for
// the block after the current tip block the generator is associated with.
//
// An overview of the algorithm is as follows:
// 1) Use the proof-of-work limit for all blocks before the first retarget
//    window
// 2) Use the previous block's difficulty if the next block is not at a retarget
//    interval
// 3) Calculate the ideal retarget difficulty for each window based on the
//    actual timespan of the window versus the target timespan and exponentially
//    weight each difficulty such that the most recent window has the highest
//    weight
// 4) Calculate the final retarget difficulty based on the exponential weighted
//    average and ensure it is limited to the max retarget adjustment factor
func (g *Generator) CalcNextRequiredDifficulty() uint32 {
	// Target difficulty before the first retarget interval is the pow
	// limit.
	nextHeight := g.tip.Header.Height + 1
	windowSize := g.params.WorkDiffWindowSize
	if int64(nextHeight) < windowSize {
		return g.params.PowLimitBits
	}

	// Return the previous block's difficulty requirements if the next block
	// is not at a difficulty retarget interval.
	curDiff := int64(g.tip.Header.Bits)
	if int64(nextHeight)%windowSize != 0 {
		return uint32(curDiff)
	}

	// Calculate the ideal retarget difficulty for each window based on the
	// actual time between blocks versus the target time and exponentially
	// weight them.
	adjustedTimespan := big.NewInt(0)
	tempBig := big.NewInt(0)
	weightedTimespanSum, weightSum := big.NewInt(0), big.NewInt(0)
	targetTimespan := int64(g.params.TargetTimespan)
	targetTimespanBig := big.NewInt(targetTimespan)
	numWindows := g.params.WorkDiffWindows
	weightAlpha := g.params.WorkDiffAlpha
	block := g.tip
	finalWindowTime := block.Header.Timestamp.UnixNano()
	for i := int64(0); i < numWindows; i++ {
		// Get the timestamp of the block at the start of the window and
		// calculate the actual timespan accordingly.  Use the target
		// timespan if there are not yet enough blocks left to cover the
		// window.
		actualTimespan := targetTimespan
		if int64(block.Header.Height) > windowSize {
			for j := int64(0); j < windowSize; j++ {
				block = g.blocks[block.Header.PrevBlock]
			}
			startWindowTime := block.Header.Timestamp.UnixNano()
			actualTimespan = finalWindowTime - startWindowTime

			// Set final window time for the next window.
			finalWindowTime = startWindowTime
		}

		// Calculate the ideal retarget difficulty for the window based
		// on the actual timespan and weight it exponentially by
		// multiplying it by 2^(window_number) such that the most recent
		// window receives the most weight.
		//
		// Also, since integer division is being used, shift up the
		// number of new tickets 32 bits to avoid losing precision.
		//
		//   windowWeightShift = ((numWindows - i) * weightAlpha)
		//   adjustedTimespan = (actualTimespan << 32) / targetTimespan
		//   weightedTimespanSum += adjustedTimespan << windowWeightShift
		//   weightSum += 1 << windowWeightShift
		windowWeightShift := uint((numWindows - i) * weightAlpha)
		adjustedTimespan.SetInt64(actualTimespan)
		adjustedTimespan.Lsh(adjustedTimespan, 32)
		adjustedTimespan.Div(adjustedTimespan, targetTimespanBig)
		adjustedTimespan.Lsh(adjustedTimespan, windowWeightShift)
		weightedTimespanSum.Add(weightedTimespanSum, adjustedTimespan)
		weight := tempBig.SetInt64(1)
		weight.Lsh(weight, windowWeightShift)
		weightSum.Add(weightSum, weight)
	}

	// Calculate the retarget difficulty based on the exponential weighted
	// average and shift the result back down 32 bits to account for the
	// previous shift up in order to avoid losing precision.  Then, limit it
	// to the maximum allowed retarget adjustment factor.
	//
	//   nextDiff = (weightedTimespanSum/weightSum * curDiff) >> 32
	curDiffBig := tempBig.SetInt64(curDiff)
	weightedTimespanSum.Div(weightedTimespanSum, weightSum)
	weightedTimespanSum.Mul(weightedTimespanSum, curDiffBig)
	weightedTimespanSum.Rsh(weightedTimespanSum, 32)
	nextDiff := weightedTimespanSum.Int64()
	nextDiff = g.limitRetarget(curDiff, nextDiff)

	if nextDiff > int64(g.params.PowLimitBits) {
		return g.params.PowLimitBits
	}
	return uint32(nextDiff)
}

// hash256prng is a determinstic pseudorandom number generator that uses a
// 256-bit secure hashing function to generate random uint32s starting from
// an initial seed.
type hash256prng struct {
	seed       chainhash.Hash // Initialization seed
	idx        uint64         // Hash iterator index
	cachedHash chainhash.Hash // Most recently generated hash
	hashOffset int            // Offset into most recently generated hash
}

// newHash256PRNG creates a pointer to a newly created hash256PRNG.
func newHash256PRNG(seed []byte) *hash256prng {
	// The provided seed is initialized by appending a constant derived from
	// the hex representation of pi and hashing the result to give 32 bytes.
	// This ensures the PRNG is always doing a short number of rounds
	// regardless of input since it will only need to hash small messages
	// (less than 64 bytes).
	seedHash := chainhash.HashFunc(append(seed, hash256prngSeedConst...))
	return &hash256prng{
		seed:       seedHash,
		idx:        0,
		cachedHash: seedHash,
	}
}

// State returns a hash that represents the current state of the deterministic
// PRNG.
func (hp *hash256prng) State() chainhash.Hash {
	// The final state is the hash of the most recently generated hash
	// concatenated with both the hash iterator index and the offset into
	// the hash.
	//
	//   hash(hp.cachedHash || hp.idx || hp.hashOffset)
	finalState := make([]byte, len(hp.cachedHash)+4+1)
	copy(finalState, hp.cachedHash[:])
	offset := len(hp.cachedHash)
	binary.BigEndian.PutUint32(finalState[offset:], uint32(hp.idx))
	offset += 4
	finalState[offset] = byte(hp.hashOffset)
	return chainhash.HashH(finalState)
}

// Hash256Rand returns a uint32 random number using the pseudorandom number
// generator and updates the state.
func (hp *hash256prng) Hash256Rand() uint32 {
	offset := hp.hashOffset * 4
	r := binary.BigEndian.Uint32(hp.cachedHash[offset : offset+4])
	hp.hashOffset++

	// Generate a new hash and reset the hash position index once it would
	// overflow the available bytes in the most recently generated hash.
	if hp.hashOffset > 7 {
		// Hash of the seed concatenated with the hash iterator index.
		//   hash(hp.seed || hp.idx)
		data := make([]byte, len(hp.seed)+4)
		copy(data, hp.seed[:])
		binary.BigEndian.PutUint32(data[len(hp.seed):], uint32(hp.idx))
		hp.cachedHash = chainhash.HashH(data)
		hp.idx++
		hp.hashOffset = 0
	}

	// Roll over the entire PRNG by re-hashing the seed when the hash
	// iterator index overlows a uint32.
	if hp.idx > math.MaxUint32 {
		hp.seed = chainhash.HashH(hp.seed[:])
		hp.cachedHash = hp.seed
		hp.idx = 0
	}

	return r
}

// uniformRandom returns a random in the range [0, upperBound) while avoiding
// modulo bias to ensure a normal distribution within the specified range.
func (hp *hash256prng) uniformRandom(upperBound uint32) uint32 {
	if upperBound < 2 {
		return 0
	}

	// (2^32 - (x*2)) % x == 2^32 % x when x <= 2^31
	min := ((math.MaxUint32 - (upperBound * 2)) + 1) % upperBound
	if upperBound > 0x80000000 {
		min = 1 + ^upperBound
	}

	r := hp.Hash256Rand()
	for r < min {
		r = hp.Hash256Rand()
	}
	return r % upperBound
}

// winningTickets returns a slice of tickets that are required to vote for the
// given block being voted on and live ticket pool and the associated underlying
// deterministic prng state hash.
func winningTickets(voteBlock *wire.MsgBlock, liveTickets []*stakeTicket, numVotes uint16) ([]*stakeTicket, chainhash.Hash, error) {
	// Serialize the parent block header used as the seed to the
	// deterministic pseudo random number generator for vote selection.
	var buf bytes.Buffer
	buf.Grow(wire.MaxBlockHeaderPayload)
	if err := voteBlock.Header.Serialize(&buf); err != nil {
		return nil, chainhash.Hash{}, err
	}

	// Ensure the number of live tickets is within the allowable range.
	numLiveTickets := uint32(len(liveTickets))
	if numLiveTickets > math.MaxUint32 {
		return nil, chainhash.Hash{}, fmt.Errorf("live ticket pool "+
			"has %d tickets which is more than the max allowed of "+
			"%d", len(liveTickets), uint32(math.MaxUint32))
	}
	if uint32(numVotes) > numLiveTickets {
		return nil, chainhash.Hash{}, fmt.Errorf("live ticket pool "+
			"has %d tickets, while %d are needed to vote",
			len(liveTickets), numVotes)
	}

	// Construct list of winners by generating successive values from the
	// deterministic prng and using them as indices into the sorted live
	// ticket pool while skipping any duplicates that might occur.
	prng := newHash256PRNG(buf.Bytes())
	winners := make([]*stakeTicket, 0, numVotes)
	usedOffsets := make(map[uint32]struct{})
	for uint16(len(winners)) < numVotes {
		ticketIndex := prng.uniformRandom(numLiveTickets)
		if _, exists := usedOffsets[ticketIndex]; !exists {
			usedOffsets[ticketIndex] = struct{}{}
			winners = append(winners, liveTickets[ticketIndex])
		}
	}
	return winners, prng.State(), nil
}

// calcFinalLotteryState calculates the final lottery state for a set of winning
// tickets and the associated deterministic prng state hash after selecting the
// winners.  It is the first 6 bytes of:
//   blake256(firstTicketHash || ... || lastTicketHash || prngStateHash)
func calcFinalLotteryState(winners []*stakeTicket, prngStateHash chainhash.Hash) [6]byte {
	data := make([]byte, (len(winners)+1)*chainhash.HashSize)
	for i := 0; i < len(winners); i++ {
		h := winners[i].tx.CachedTxHash()
		copy(data[chainhash.HashSize*i:], h[:])
	}
	copy(data[chainhash.HashSize*len(winners):], prngStateHash[:])
	dataHash := chainhash.HashH(data)

	var finalState [6]byte
	copy(finalState[:], dataHash[0:6])
	return finalState
}

// nextPowerOfTwo returns the next highest power of two from a given number if
// it is not already a power of two.  This is a helper function used during the
// calculation of a merkle tree.
func nextPowerOfTwo(n int) int {
	// Return the number if it's already a power of 2.
	if n&(n-1) == 0 {
		return n
	}

	// Figure out and return the next power of two.
	exponent := uint(math.Log2(float64(n))) + 1
	return 1 << exponent // 2^exponent
}

// hashMerkleBranches takes two hashes, treated as the left and right tree
// nodes, and returns the hash of their concatenation.  This is a helper
// function used to aid in the generation of a merkle tree.
func hashMerkleBranches(left *chainhash.Hash, right *chainhash.Hash) *chainhash.Hash {
	// Concatenate the left and right nodes.
	var hash [chainhash.HashSize * 2]byte
	copy(hash[:chainhash.HashSize], left[:])
	copy(hash[chainhash.HashSize:], right[:])

	newHash := chainhash.HashH(hash[:])
	return &newHash
}

// buildMerkleTreeStore creates a merkle tree from a slice of transactions,
// stores it using a linear array, and returns a slice of the backing array.  A
// linear array was chosen as opposed to an actual tree structure since it uses
// about half as much memory.  The following describes a merkle tree and how it
// is stored in a linear array.
//
// A merkle tree is a tree in which every non-leaf node is the hash of its
// children nodes.  A diagram depicting how this works for Decred transactions
// where h(x) is a blake256 hash follows:
//
//	         root = h1234 = h(h12 + h34)
//	        /                           \
//	  h12 = h(h1 + h2)            h34 = h(h3 + h4)
//	   /            \              /            \
//	h1 = h(tx1)  h2 = h(tx2)    h3 = h(tx3)  h4 = h(tx4)
//
// The above stored as a linear array is as follows:
//
// 	[h1 h2 h3 h4 h12 h34 root]
//
// As the above shows, the merkle root is always the last element in the array.
//
// The number of inputs is not always a power of two which results in a
// balanced tree structure as above.  In that case, parent nodes with no
// children are also zero and parent nodes with only a single left node
// are calculated by concatenating the left node with itself before hashing.
// Since this function uses nodes that are pointers to the hashes, empty nodes
// will be nil.
func buildMerkleTreeStore(transactions []*ndrutil.Tx) []*chainhash.Hash {
	// If there's an empty stake tree, return totally zeroed out merkle tree root
	// only.
	if len(transactions) == 0 {
		merkles := make([]*chainhash.Hash, 1)
		merkles[0] = &chainhash.Hash{}
		return merkles
	}

	// Calculate how many entries are required to hold the binary merkle
	// tree as a linear array and create an array of that size.
	nextPoT := nextPowerOfTwo(len(transactions))
	arraySize := nextPoT*2 - 1
	merkles := make([]*chainhash.Hash, arraySize)

	// Create the base transaction hashes and populate the array with them.
	for i, tx := range transactions {
		msgTx := tx.MsgTx()
		txHashFull := msgTx.TxHashFull()
		merkles[i] = &txHashFull
	}

	// Start the array offset after the last transaction and adjusted to the
	// next power of two.
	offset := nextPoT
	for i := 0; i < arraySize-1; i += 2 {
		switch {
		// When there is no left child node, the parent is nil too.
		case merkles[i] == nil:
			merkles[offset] = nil

		// When there is no right child, the parent is generated by
		// hashing the concatenation of the left child with itself.
		case merkles[i+1] == nil:
			newHash := hashMerkleBranches(merkles[i], merkles[i])
			merkles[offset] = newHash

		// The normal case sets the parent node to the hash of the
		// concatentation of the left and right children.
		default:
			newHash := hashMerkleBranches(merkles[i], merkles[i+1])
			merkles[offset] = newHash
		}
		offset++
	}

	return merkles
}

// calcMerkleRoot creates a merkle tree from the slice of transactions and
// returns the root of the tree.
func calcMerkleRoot(txns []*wire.MsgTx) chainhash.Hash {
	utilTxns := make([]*ndrutil.Tx, 0, len(txns))
	for _, tx := range txns {
		utilTxns = append(utilTxns, ndrutil.NewTx(tx))
	}
	merkles := buildMerkleTreeStore(utilTxns)
	return *merkles[len(merkles)-1]
}

// hashToBig converts a chainhash.Hash into a big.Int that can be used to
// perform math comparisons.
func hashToBig(hash *chainhash.Hash) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// compactToBig converts a compact representation of a whole number N to an
// unsigned 32-bit number.  The representation is similar to IEEE754 floating
// point numbers.
//
// Like IEEE754 floating point, there are three basic components: the sign,
// the exponent, and the mantissa.  They are broken out as follows:
//
//	* the most significant 8 bits represent the unsigned base 256 exponent
// 	* bit 23 (the 24th bit) represents the sign bit
//	* the least significant 23 bits represent the mantissa
//
//	-------------------------------------------------
//	|   Exponent     |    Sign    |    Mantissa     |
//	-------------------------------------------------
//	| 8 bits [31-24] | 1 bit [23] | 23 bits [22-00] |
//	-------------------------------------------------
//
// The formula to calculate N is:
// 	N = (-1^sign) * mantissa * 256^(exponent-3)
//
// This compact form is only used in Decred to encode unsigned 256-bit numbers
// which represent difficulty targets, thus there really is not a need for a
// sign bit, but it is implemented here to stay consistent with bitcoind.
func compactToBig(compact uint32) *big.Int {
	// Extract the mantissa, sign bit, and exponent.
	mantissa := compact & 0x007fffff
	isNegative := compact&0x00800000 != 0
	exponent := uint(compact >> 24)

	// Since the base for the exponent is 256, the exponent can be treated
	// as the number of bytes to represent the full 256-bit number.  So,
	// treat the exponent as the number of bytes and shift the mantissa
	// right or left accordingly.  This is equivalent to:
	// N = mantissa * 256^(exponent-3)
	var bn *big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		bn.Lsh(bn, 8*(exponent-3))
	}

	// Make it negative if the sign bit is set.
	if isNegative {
		bn = bn.Neg(bn)
	}

	return bn
}

// IsSolved returns whether or not the header hashes to a value that is less
// than or equal to the target difficulty as specified by its bits field.
func IsSolved(header *wire.BlockHeader) bool {
	targetDifficulty := compactToBig(header.Bits)
	hash := header.BlockHash()
	return hashToBig(&hash).Cmp(targetDifficulty) <= 0
}

// solveBlock attempts to find a nonce which makes the passed block header hash
// to a value less than the target difficulty.  When a successful solution is
// found, true is returned and the nonce field of the passed header is updated
// with the solution.  False is returned if no solution exists.
//
// NOTE: This function will never solve blocks with a nonce of 0.  This is done
// so the 'NextBlock' function can properly detect when a nonce was modified by
// a munge function.
func solveBlock(header *wire.BlockHeader) bool {
	// sbResult is used by the solver goroutines to send results.
	type sbResult struct {
		found bool
		nonce uint32
	}

	// solver accepts a block header and a nonce range to test. It is
	// intended to be run as a goroutine.
	targetDifficulty := compactToBig(header.Bits)
	quit := make(chan bool)
	results := make(chan sbResult)
	solver := func(hdr wire.BlockHeader, startNonce, stopNonce uint32) {
		// We need to modify the nonce field of the header, so make sure
		// we work with a copy of the original header.
		for i := startNonce; i >= startNonce && i <= stopNonce; i++ {
			select {
			case <-quit:
				results <- sbResult{false, 0}
				return
			default:
				hdr.Nonce = i
				hash := hdr.BlockHash()
				if hashToBig(&hash).Cmp(
					targetDifficulty) <= 0 {

					results <- sbResult{true, i}
					return
				}
			}
		}
		results <- sbResult{false, 0}
	}

	startNonce := uint32(1)
	stopNonce := uint32(math.MaxUint32)
	numCores := uint32(runtime.NumCPU())
	noncesPerCore := (stopNonce - startNonce) / numCores
	for i := uint32(0); i < numCores; i++ {
		rangeStart := startNonce + (noncesPerCore * i)
		rangeStop := startNonce + (noncesPerCore * (i + 1)) - 1
		if i == numCores-1 {
			rangeStop = stopNonce
		}
		go solver(*header, rangeStart, rangeStop)
	}
	var foundResult bool
	for i := uint32(0); i < numCores; i++ {
		result := <-results
		if !foundResult && result.found {
			close(quit)
			header.Nonce = result.nonce
			foundResult = true
		}
	}

	return foundResult
}

// ReplaceWithNVotes returns a function that itself takes a block and modifies
// it by replacing the votes in the stake tree with specified number of votes.
//
// NOTE: This must only be used as a munger to the 'NextBlock' function or it
// will lead to an invalid live ticket pool.  To help safeguard against improper
// usage, it will panic if called with a block that does not connect to the
// current tip block.
func (g *Generator) ReplaceWithNVotes(numVotes uint16) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		// Attempt to prevent misuse of this function by ensuring the
		// provided block connects to the current tip.
		if b.Header.PrevBlock != g.tip.BlockHash() {
			panic(fmt.Sprintf("attempt to replace number of votes "+
				"for block %s with parent %s that is not the "+
				"current tip %s", b.BlockHash(),
				b.Header.PrevBlock, g.tip.BlockHash()))
		}

		// Get the winning tickets for the specified number of votes.
		parentBlock := g.tip
		winners, _, err := winningTickets(parentBlock, g.liveTickets,
			numVotes)
		if err != nil {
			panic(err)
		}

		// Generate vote transactions for the winning tickets.
		defaultNumVotes := int(g.params.TicketsPerBlock)
		numExisting := len(b.STransactions) - defaultNumVotes
		stakeTxns := make([]*wire.MsgTx, 0, numExisting+int(numVotes))
		for _, ticket := range winners {
			voteTx := g.createVoteTxFromTicket(parentBlock, ticket)
			stakeTxns = append(stakeTxns, voteTx)
		}

		// Add back the original stake transactions other than the
		// original stake votes that have been replaced.
		stakeTxns = append(stakeTxns, b.STransactions[defaultNumVotes:]...)

		// Update the block with the new stake transactions and the
		// header with the new number of votes.
		b.STransactions = stakeTxns
		b.Header.Voters = numVotes

		// Recalculate the coinbase amount based on the number of new
		// votes and update the coinbase so that the adjustment in
		// subsidy is accounted for.
		height := b.Header.Height
		fullSubsidy := g.calcFullSubsidy(height)
		devSubsidy := g.calcDevSubsidy(fullSubsidy, height, numVotes)
		powSubsidy := g.calcPoWSubsidy(fullSubsidy, height, numVotes)
		cbTx := b.Transactions[0]
		cbTx.TxIn[0].ValueIn = int64(devSubsidy + powSubsidy)
		cbTx.TxOut = nil
		g.addCoinbaseTxOutputs(cbTx, height, devSubsidy, powSubsidy)
	}
}

// ReplaceVoteBitsN returns a function that itself takes a block and modifies
// it by replacing the vote bits of the vote located at the provided index.  It
// will panic if the stake transaction at the provided index is not already a
// vote.
//
// NOTE: This must only be used as a munger to the 'NextBlock' function or it
// will lead to an invalid live ticket pool.
func (g *Generator) ReplaceVoteBitsN(voteNum int, voteBits uint16) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		// Attempt to prevent misuse of this function by ensuring the
		// provided stake transaction number is actually a vote.
		stx := b.STransactions[voteNum]
		if !isVoteTx(stx) {
			panic(fmt.Sprintf("attempt to replace non-vote "+
				"transaction #%d for for block %s", voteNum,
				b.BlockHash()))
		}

		// Extract the existing vote version.
		existingScript := stx.TxOut[1].PkScript
		var voteVersion uint32
		if len(existingScript) >= 8 {
			voteVersion = binary.LittleEndian.Uint32(existingScript[4:8])
		}

		stx.TxOut[1].PkScript = voteBitsScript(voteBits, voteVersion)
	}
}

// ReplaceBlockVersion returns a function that itself takes a block and modifies
// it by replacing the stake version of the header.
func ReplaceBlockVersion(newVersion int32) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		b.Header.Version = newVersion
	}
}

// ReplaceStakeVersion returns a function that itself takes a block and modifies
// it by replacing the stake version of the header.
func ReplaceStakeVersion(newVersion uint32) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		b.Header.StakeVersion = newVersion
	}
}

// ReplaceVoteVersions returns a function that itself takes a block and modifies
// it by replacing the voter version of the stake transactions.
//
// NOTE: This must only be used as a munger to the 'NextBlock' function or it
// will lead to an invalid live ticket pool.
func ReplaceVoteVersions(newVersion uint32) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		for _, stx := range b.STransactions {
			if isVoteTx(stx) {
				stx.TxOut[1].PkScript = voteBitsScript(
					voteBitYes, newVersion)
			}
		}
	}
}

// ReplaceVotes returns a function that itself takes a block and modifies it by
// replacing the voter version and bits of the stake transactions.
//
// NOTE: This must only be used as a munger to the 'NextBlock' function or it
// will lead to an invalid live ticket pool.
func ReplaceVotes(voteBits uint16, newVersion uint32) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		for _, stx := range b.STransactions {
			if isVoteTx(stx) {
				stx.TxOut[1].PkScript = voteBitsScript(voteBits,
					newVersion)
			}
		}
	}
}

// CreateSpendTx creates a transaction that spends from the provided spendable
// output and includes an additional unique OP_RETURN output to ensure the
// transaction ends up with a unique hash.  The public key script is a simple
// OP_TRUE p2sh script which avoids the need to track addresses and signature
// scripts in the tests.  The signature script is the opTrueRedeemScript.
func (g *Generator) CreateSpendTx(spend *SpendableOut, fee ndrutil.Amount) *wire.MsgTx {
	spendTx := wire.NewMsgTx()
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: spend.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		ValueIn:          int64(spend.amount),
		BlockHeight:      spend.blockHeight,
		BlockIndex:       spend.blockIndex,
		SignatureScript:  opTrueRedeemScript,
	})
	spendTx.AddTxOut(wire.NewTxOut(int64(spend.amount-fee),
		g.p2shOpTrueScript))
	spendTx.AddTxOut(wire.NewTxOut(0, UniqueOpReturnScript()))
	return spendTx
}

// CreateSpendTxForTx creates a transaction that spends from the first output of
// the provided transaction and includes an additional unique OP_RETURN output
// to ensure the transaction ends up with a unique hash.  The public key script
// is a simple OP_TRUE p2sh script which avoids the need to track addresses and
// signature scripts in the tests.  This signature script the
// opTrueRedeemScript.
func (g *Generator) CreateSpendTxForTx(tx *wire.MsgTx, blockHeight, txIndex uint32, fee ndrutil.Amount) *wire.MsgTx {
	spend := MakeSpendableOutForTx(tx, blockHeight, txIndex, 0)
	return g.CreateSpendTx(&spend, fee)
}

// originalParent returns the original block the passed block was built from.
// This is necessary because callers might change the previous block hash in a
// munger which would cause the like ticket pool to be reconstructed improperly.
func (g *Generator) originalParent(b *wire.MsgBlock) *wire.MsgBlock {
	parentHash, ok := g.originalParents[b.BlockHash()]
	if !ok {
		parentHash = b.Header.PrevBlock
	}
	return g.BlockByHash(&parentHash)
}

// SetTip changes the tip of the instance to the block with the provided name.
// This is useful since the tip is used for things such as generating subsequent
// blocks.
func (g *Generator) SetTip(blockName string) {
	// Nothing to do if already the tip.
	if blockName == g.tipName {
		return
	}

	newTip := g.blocksByName[blockName]
	if newTip == nil {
		panic(fmt.Sprintf("tip block name %s does not exist", blockName))
	}

	// Create a list of blocks to disconnect and blocks to connect in order
	// to switch to the new tip.
	var connect, disconnect []*wire.MsgBlock
	oldBranch, newBranch := g.tip, newTip
	for oldBranch != newBranch {
		oldBranchHeight := g.blockHeight(oldBranch.BlockHash())
		newBranchHeight := g.blockHeight(newBranch.BlockHash())
		if oldBranchHeight > newBranchHeight {
			disconnect = append(disconnect, oldBranch)
			oldBranch = g.originalParent(oldBranch)
			continue
		} else if newBranchHeight > oldBranchHeight {
			connect = append(connect, newBranch)
			newBranch = g.originalParent(newBranch)
			continue
		}

		// At this point the two branches have the same height, so add
		// each tip to the appropriate connect or disconnect list and
		// the tips to their previous block.
		disconnect = append(disconnect, oldBranch)
		oldBranch = g.originalParent(oldBranch)
		connect = append(connect, newBranch)
		newBranch = g.originalParent(newBranch)
	}

	// Update the live ticket pool and associated data structs by
	// disconnecting all blocks back to the fork point.
	for _, block := range disconnect {
		g.disconnectBlockTickets(block)
		g.tip = g.originalParent(block)
	}

	// Update the live ticket pool and associated data structs by connecting
	// all blocks after the fork point up to the new tip.  The list of
	// blocks to connect is iterated in reverse order, because it was
	// constructed in reverse, and the blocks need to be connected in the
	// order in which they build the chain.
	for i := len(connect) - 1; i >= 0; i-- {
		block := connect[i]
		g.connectBlockTickets(block)
		g.tip = block
	}

	// Ensure the tip is the expected new tip and set the associated name.
	if g.tip != newTip {
		panic(fmt.Sprintf("tip %s is not expected new tip %s",
			g.tip.BlockHash(), newTip.BlockHash()))
	}
	g.tipName = blockName
}

// updateVoteCommitments updates all of the votes in the passed block to commit
// to the previous block hash and previous height based on the values specified
// in the header.
func updateVoteCommitments(block *wire.MsgBlock) {
	for _, stx := range block.STransactions {
		if !isVoteTx(stx) {
			continue
		}

		stx.TxOut[0].PkScript = VoteCommitmentScript(block.Header.PrevBlock,
			block.Header.Height-1)
	}
}

// NextBlock builds a new block that extends the current tip associated with the
// generator and updates the generator's tip to the newly generated block.
//
// The block will include the following:
// - A coinbase with the following outputs:
//   - One that pays the required 10% subsidy to the dev org
//   - One that contains a standard coinbase OP_RETURN script
//   - Six that pay the required 60% subsidy to an OP_TRUE p2sh script
// - When a spendable output is provided:
//   - A transaction that spends from the provided output the following outputs:
//     - One that pays the inputs amount minus 1 atom to an OP_TRUE p2sh script
// - Once the coinbase maturity has been reached:
//   - A ticket purchase transaction (sstx) for each provided ticket spendable
//     output with the following outputs:
//     - One OP_SSTX output that grants voting rights to an OP_TRUE p2sh script
//     - One OP_RETURN output that contains the required commitment and pays
//       the subsidy to an OP_TRUE p2sh script
//     - One OP_SSTXCHANGE output that sends change to an OP_TRUE p2sh script
// - Once the stake validation height has been reached:
//   - 5 vote transactions (ssgen) as required according to the live ticket
//     pool and vote selection rules with the following outputs:
//     - One OP_RETURN followed by the block hash and height being voted on
//     - One OP_RETURN followed by the vote bits
//     - One or more OP_SSGEN outputs with the payouts according to the original
//       ticket commitments
//   - Revocation transactions (ssrtx) as required according to any missed votes
//     with the following outputs:
//     - One or more OP_SSRTX outputs with the payouts according to the original
//       ticket commitments
//
// Additionally, if one or more munge functions are specified, they will be
// invoked with the block prior to solving it.  This provides callers with the
// opportunity to modify the block which is especially useful for testing.
//
// In order to simply the logic in the munge functions, the following rules are
// applied after all munge functions have been invoked:
// - All votes will have their commitments updated if the previous hash or
//   height was manually changed after stake validation height has been reached
// - The merkle root will be recalculated unless it was manually changed
// - The stake root will be recalculated unless it was manually changed
// - The size of the block will be recalculated unless it was manually changed
// - The block will be solved unless the nonce was changed
func (g *Generator) NextBlock(blockName string, spend *SpendableOut, ticketSpends []SpendableOut, mungers ...func(*wire.MsgBlock)) *wire.MsgBlock {
	// Prevent block name collisions.
	if g.blocksByName[blockName] != nil {
		panic(fmt.Sprintf("block name %s already exists", blockName))
	}

	// Calculate the next required stake difficulty (aka ticket price).
	ticketPrice := ndrutil.Amount(g.CalcNextRequiredStakeDifficulty())

	// Generate the appropriate votes and ticket purchases based on the
	// current tip block and provided ticket spendable outputs.
	var ticketWinners []*stakeTicket
	var stakeTxns []*wire.MsgTx
	var finalState [6]byte
	nextHeight := g.tip.Header.Height + 1
	if nextHeight > uint32(g.params.CoinbaseMaturity) {
		// Generate votes once the stake validation height has been
		// reached.
		if int64(nextHeight) >= g.params.StakeValidationHeight {
			// Generate and add the vote transactions for the
			// winning tickets to the stake tree.
			numVotes := g.params.TicketsPerBlock
			winners, stateHash, err := winningTickets(g.tip,
				g.liveTickets, numVotes)
			if err != nil {
				panic(err)
			}
			ticketWinners = winners
			for _, ticket := range winners {
				voteTx := g.createVoteTxFromTicket(g.tip, ticket)
				stakeTxns = append(stakeTxns, voteTx)
			}

			// Calculate the final lottery state hash for use in the
			// block header.
			finalState = calcFinalLotteryState(winners, stateHash)
		}

		// Generate ticket purchases (sstx) using the provided spendable
		// outputs.
		if ticketSpends != nil {
			const ticketFee = ndrutil.Amount(2)
			for i := 0; i < len(ticketSpends); i++ {
				out := &ticketSpends[i]
				purchaseTx := g.CreateTicketPurchaseTx(out,
					ticketPrice, ticketFee)
				stakeTxns = append(stakeTxns, purchaseTx)
			}
		}

		// Generate and add revocations for any missed tickets.
		for _, missedVote := range g.missedVotes {
			revocationTx := g.createRevocationTxFromTicket(missedVote)
			stakeTxns = append(stakeTxns, revocationTx)
		}
	}

	// Count the ticket purchases (sstx), votes (ssgen,  and ticket revocations
	// (ssrtx),  and calculate the total PoW fees generated by the stake
	// transactions.
	var numVotes uint16
	var numTicketRevocations uint8
	var numTicketPurchases uint8
	var stakeTreeFees ndrutil.Amount
	for _, tx := range stakeTxns {
		switch {
		case isVoteTx(tx):
			numVotes++
		case isTicketPurchaseTx(tx):
			numTicketPurchases++
		case isRevocationTx(tx):
			numTicketRevocations++
		}

		// Calculate any fees for the transaction.
		var inputSum, outputSum ndrutil.Amount
		for _, txIn := range tx.TxIn {
			inputSum += ndrutil.Amount(txIn.ValueIn)
		}
		for _, txOut := range tx.TxOut {
			outputSum += ndrutil.Amount(txOut.Value)
		}
		stakeTreeFees += (inputSum - outputSum)
	}

	// Create a standard coinbase and spending transaction.
	var regularTxns []*wire.MsgTx
	{
		// Create coinbase transaction for the block with no additional
		// dev or pow subsidy.
		coinbaseTx := g.CreateCoinbaseTx(nextHeight, numVotes)
		regularTxns = []*wire.MsgTx{coinbaseTx}

		// Increase the PoW subsidy to account for any fees in the stake
		// tree.
		coinbaseTx.TxOut[2].Value += int64(stakeTreeFees)

		// Create a transaction to spend the provided utxo if needed.
		if spend != nil {
			// Create the transaction with a fee of 1 atom for the
			// miner and increase the PoW subsidy accordingly.
			fee := ndrutil.Amount(1)
			coinbaseTx.TxOut[2].Value += int64(fee)

			// Create a transaction that spends from the provided
			// spendable output and includes an additional unique
			// OP_RETURN output to ensure the transaction ends up
			// with a unique hash, then add it to the list of
			// transactions to include in the block.  The script is
			// a simple OP_TRUE p2sh script in order to avoid the
			// need to track addresses and signature scripts in the
			// tests.
			spendTx := g.CreateSpendTx(spend, fee)
			regularTxns = append(regularTxns, spendTx)
		}
	}

	// Use a timestamp that is 7/8 of target timespan after the previous
	// block unless this is the first block in which case the current time
	// is used or the proof-of-work difficulty parameters have been adjusted
	// such that it's greater than the max 2 hours worth of blocks that can
	// be tested in which case one second is used.  This helps maintain the
	// retarget difficulty low as needed.  Also, ensure the timestamp is
	// limited to one second precision.
	var ts time.Time
	if nextHeight == 1 {
		ts = time.Now()
	} else {
		if g.params.WorkDiffWindowSize > 7200 {
			ts = g.tip.Header.Timestamp.Add(time.Second)
		} else {
			addDuration := g.params.TargetTimespan * 7 / 8
			ts = g.tip.Header.Timestamp.Add(addDuration)
		}
	}
	ts = time.Unix(ts.Unix(), 0)

	// Create the unsolved block.
	prevHash := g.tip.BlockHash()
	block := wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:      1,
			PrevBlock:    prevHash,
			MerkleRoot:   calcMerkleRoot(regularTxns),
			StakeRoot:    calcMerkleRoot(stakeTxns),
			VoteBits:     1,
			FinalState:   finalState,
			Voters:       numVotes,
			FreshStake:   numTicketPurchases,
			Revocations:  numTicketRevocations,
			PoolSize:     uint32(len(g.liveTickets)),
			Bits:         g.CalcNextRequiredDifficulty(),
			SBits:        int64(ticketPrice),
			Height:       nextHeight,
			Size:         0, // Filled in below.
			Timestamp:    ts,
			Nonce:        0, // To be solved.
			ExtraData:    [32]byte{},
			StakeVersion: 0,
		},
		Transactions:  regularTxns,
		STransactions: stakeTxns,
	}
	block.Header.Size = uint32(block.SerializeSize())

	// Perform any block munging just before solving.  Once stake validation
	// height has been reached, update the vote commitments accordingly if the
	// header height or previous hash was manually changed by a munge function.
	// Also, only recalculate the merkle roots and block size if they weren't
	// manually changed by a munge function.
	curMerkleRoot := block.Header.MerkleRoot
	curStakeRoot := block.Header.StakeRoot
	curSize := block.Header.Size
	curNonce := block.Header.Nonce
	for _, f := range mungers {
		f(&block)
	}
	if block.Header.Height != nextHeight || block.Header.PrevBlock != prevHash {
		if int64(nextHeight) >= g.params.StakeValidationHeight {
			updateVoteCommitments(&block)
		}
	}
	if block.Header.MerkleRoot == curMerkleRoot {
		block.Header.MerkleRoot = calcMerkleRoot(block.Transactions)
	}
	if block.Header.StakeRoot == curStakeRoot {
		block.Header.StakeRoot = calcMerkleRoot(block.STransactions)
	}
	if block.Header.Size == curSize {
		block.Header.Size = uint32(block.SerializeSize())
	}

	// Only solve the block if the nonce wasn't manually changed by a munge
	// function.
	if block.Header.Nonce == curNonce && !solveBlock(&block.Header) {
		panic(fmt.Sprintf("unable to solve block at height %d",
			block.Header.Height))
	}

	// Create stake tickets for the ticket purchases (sstx) in the block.  This
	// is done after the mungers to ensure all changes are accurately accounted
	// for.
	var ticketPurchases []*stakeTicket
	for txIdx, tx := range block.STransactions {
		if isTicketPurchaseTx(tx) {
			ticket := &stakeTicket{tx, nextHeight, uint32(txIdx)}
			ticketPurchases = append(ticketPurchases, ticket)
		}
	}

	// Update generator state and return the block.
	blockHash := block.BlockHash()
	if block.Header.PrevBlock != prevHash {
		// Save the orignal block this one was built from if it was
		// manually changed in a munger so the code which deals with
		// updating the live tickets when changing the tip has access to
		// it.
		g.originalParents[blockHash] = prevHash
	}
	g.addMissedVotes(&blockHash, block.STransactions, ticketWinners)
	g.connectRevocations(&blockHash, block.STransactions)
	g.connectLiveTickets(&blockHash, nextHeight, ticketWinners,
		ticketPurchases)
	g.blocks[blockHash] = &block
	g.blockHeights[blockHash] = nextHeight
	g.blocksByName[blockName] = &block
	g.tip = &block
	g.tipName = blockName
	return &block
}

// CreatePremineBlock generates the first block of the chain with the required
// premine payouts.  The additional amount parameter can be used to create a
// block that is otherwise a completely valid premine block except it adds the
// extra amount to each payout and thus create a block that violates consensus.
func (g *Generator) CreatePremineBlock(blockName string, additionalAmount ndrutil.Amount, mungers ...func(*wire.MsgBlock)) *wire.MsgBlock {
	coinbaseTx := wire.NewMsgTx()
	coinbaseTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex, wire.TxTreeRegular),
		Sequence:        wire.MaxTxInSequenceNum,
		ValueIn:         0, // Updated below.
		BlockHeight:     wire.NullBlockHeight,
		BlockIndex:      wire.NullBlockIndex,
		SignatureScript: coinbaseSigScript,
	})

	// Add each required output and tally the total payouts for the coinbase
	// in order to set the input value appropriately.
	var totalSubsidy ndrutil.Amount
	for _, payout := range g.params.BlockOneLedger {
		payoutAddr, err := ndrutil.DecodeAddress(payout.Address)
		if err != nil {
			panic(err)
		}
		pkScript, err := txscript.PayToAddrScript(payoutAddr)
		if err != nil {
			panic(err)
		}
		coinbaseTx.AddTxOut(&wire.TxOut{
			Value:    payout.Amount + int64(additionalAmount),
			Version:  0,
			PkScript: pkScript,
		})

		totalSubsidy += ndrutil.Amount(payout.Amount)
	}
	coinbaseTx.TxIn[0].ValueIn = int64(totalSubsidy)

	// Generate the block with the specially created regular transactions.
	munger := func(b *wire.MsgBlock) {
		b.Transactions = []*wire.MsgTx{coinbaseTx}
	}
	mungers = append([]func(*wire.MsgBlock){munger}, mungers...)
	return g.NextBlock(blockName, nil, nil, mungers...)
}

// UpdateBlockState manually updates the generator state to remove all internal
// map references to a block via its old hash and insert new ones for the new
// block hash.  This is useful if the test code has to manually change a block
// after 'NextBlock' has returned.
func (g *Generator) UpdateBlockState(oldBlockName string, oldBlockHash chainhash.Hash, newBlockName string, newBlock *wire.MsgBlock) {
	// Remove existing entries.
	wonTickets := g.wonTickets[oldBlockHash]
	existingHeight := g.blockHeights[oldBlockHash]
	delete(g.blocks, oldBlockHash)
	delete(g.blockHeights, oldBlockHash)
	delete(g.blocksByName, oldBlockName)
	delete(g.wonTickets, oldBlockHash)

	// Add new entries.
	newBlockHash := newBlock.BlockHash()
	g.blocks[newBlockHash] = newBlock
	g.blockHeights[newBlockHash] = existingHeight
	g.blocksByName[newBlockName] = newBlock
	g.wonTickets[newBlockHash] = wonTickets
}

// OldestCoinbaseOuts removes the oldest set of coinbase proof-of-work outputs
// that was previously saved to the generator and returns the set as a slice.
func (g *Generator) OldestCoinbaseOuts() []SpendableOut {
	outs := g.spendableOuts[0]
	g.spendableOuts = g.spendableOuts[1:]
	return outs
}

// NumSpendableCoinbaseOuts returns the number of proof-of-work outputs that
// were previously saved to the generated but have not yet been collected.
func (g *Generator) NumSpendableCoinbaseOuts() int {
	return len(g.spendableOuts)
}

// saveCoinbaseOuts adds the proof-of-work outputs of the coinbase tx in the
// passed block to the list of spendable outputs.
func (g *Generator) saveCoinbaseOuts(b *wire.MsgBlock) {
	g.spendableOuts = append(g.spendableOuts, []SpendableOut{
		MakeSpendableOut(b, 0, 2),
		MakeSpendableOut(b, 0, 3),
		MakeSpendableOut(b, 0, 4),
		MakeSpendableOut(b, 0, 5),
		MakeSpendableOut(b, 0, 6),
		MakeSpendableOut(b, 0, 7),
	})
	g.prevCollectedHash = b.BlockHash()
}

// SaveTipCoinbaseOuts adds the proof-of-work outputs of the coinbase tx in the
// current tip block to the list of spendable outputs.
func (g *Generator) SaveTipCoinbaseOuts() {
	g.saveCoinbaseOuts(g.tip)
}

// SaveSpendableCoinbaseOuts adds all proof-of-work coinbase outputs starting
// from the block after the last block that had its coinbase outputs collected
// and ending at the current tip.  This is useful to batch the collection of the
// outputs once the tests reach a stable point so they don't have to manually
// add them for the right tests which will ultimately end up being the best
// chain.
func (g *Generator) SaveSpendableCoinbaseOuts() {
	// Loop through the ancestors of the current tip until the
	// reaching the block that has already had the coinbase outputs
	// collected.
	var collectBlocks []*wire.MsgBlock
	for b := g.tip; b != nil; b = g.blocks[b.Header.PrevBlock] {
		if b.BlockHash() == g.prevCollectedHash {
			break
		}
		collectBlocks = append(collectBlocks, b)
	}
	for i := range collectBlocks {
		g.saveCoinbaseOuts(collectBlocks[len(collectBlocks)-1-i])
	}
}

// AssertTipHeight panics if the current tip block associated with the generator
// does not have the specified height.
func (g *Generator) AssertTipHeight(expected uint32) {
	height := g.tip.Header.Height
	if height != expected {
		panic(fmt.Sprintf("height for block %q is %d instead of "+
			"expected %d", g.tipName, height, expected))
	}
}

// AssertScriptSigOpsCount panics if the provided script does not have the
// specified number of signature operations.
func (g *Generator) AssertScriptSigOpsCount(script []byte, expected int) {
	numSigOps := txscript.GetSigOpCount(script)
	if numSigOps != expected {
		_, file, line, _ := runtime.Caller(1)
		panic(fmt.Sprintf("assertion failed at %s:%d: generated number "+
			"of sigops for script is %d instead of expected %d",
			file, line, numSigOps, expected))
	}
}

// countBlockSigOps returns the number of legacy signature operations in the
// scripts in the passed block.
func countBlockSigOps(block *wire.MsgBlock) int {
	totalSigOps := 0
	for _, tx := range block.Transactions {
		for _, txIn := range tx.TxIn {
			numSigOps := txscript.GetSigOpCount(txIn.SignatureScript)
			totalSigOps += numSigOps
		}
		for _, txOut := range tx.TxOut {
			numSigOps := txscript.GetSigOpCount(txOut.PkScript)
			totalSigOps += numSigOps
		}
	}

	return totalSigOps
}

// AssertTipBlockSigOpsCount panics if the current tip block associated with the
// generator does not have the specified number of signature operations.
func (g *Generator) AssertTipBlockSigOpsCount(expected int) {
	numSigOps := countBlockSigOps(g.tip)
	if numSigOps != expected {
		panic(fmt.Sprintf("generated number of sigops for block %q "+
			"(height %d) is %d instead of expected %d", g.tipName,
			g.tip.Header.Height, numSigOps, expected))
	}
}

// AssertTipBlockSize panics if the if the current tip block associated with the
// generator does not have the specified size when serialized.
func (g *Generator) AssertTipBlockSize(expected int) {
	serializeSize := g.tip.SerializeSize()
	if serializeSize != expected {
		panic(fmt.Sprintf("block size of block %q (height %d) is %d "+
			"instead of expected %d", g.tipName,
			g.tip.Header.Height, serializeSize, expected))
	}
}

// AssertTipBlockNumTxns panics if the number of transactions in the current tip
// block associated with the generator does not match the specified value.
func (g *Generator) AssertTipBlockNumTxns(expected int) {
	numTxns := len(g.tip.Transactions)
	if numTxns != expected {
		panic(fmt.Sprintf("number of txns in block %q (height %d) is "+
			"%d instead of expected %d", g.tipName,
			g.tip.Header.Height, numTxns, expected))
	}
}

// AssertTipBlockHash panics if the current tip block associated with the
// generator does not match the specified hash.
func (g *Generator) AssertTipBlockHash(expected chainhash.Hash) {
	hash := g.tip.BlockHash()
	if hash != expected {
		panic(fmt.Sprintf("block hash of block %q (height %d) is %v "+
			"instead of expected %v", g.tipName,
			g.tip.Header.Height, hash, expected))
	}
}

// AssertTipBlockMerkleRoot panics if the merkle root in header of the current
// tip block associated with the generator does not match the specified hash.
func (g *Generator) AssertTipBlockMerkleRoot(expected chainhash.Hash) {
	hash := g.tip.Header.MerkleRoot
	if hash != expected {
		panic(fmt.Sprintf("merkle root of block %q (height %d) is %v "+
			"instead of expected %v", g.tipName,
			g.tip.Header.Height, hash, expected))
	}
}

// AssertTipBlockStakeRoot panics if the stake root in header of the current
// tip block associated with the generator does not match the specified hash.
func (g *Generator) AssertTipBlockStakeRoot(expected chainhash.Hash) {
	hash := g.tip.Header.StakeRoot
	if hash != expected {
		panic(fmt.Sprintf("stake root of block %q (height %d) is %v "+
			"instead of expected %v", g.tipName,
			g.tip.Header.Height, hash, expected))
	}
}

// AssertTipBlockTxOutOpReturn panics if the current tip block associated with
// the generator does not have an OP_RETURN script for the transaction output at
// the provided tx index and output index.
func (g *Generator) AssertTipBlockTxOutOpReturn(txIndex, txOutIndex uint32) {
	if txIndex >= uint32(len(g.tip.Transactions)) {
		panic(fmt.Sprintf("transaction index %d in block %q "+
			"(height %d) does not exist", txIndex, g.tipName,
			g.tip.Header.Height))
	}

	tx := g.tip.Transactions[txIndex]
	if txOutIndex >= uint32(len(tx.TxOut)) {
		panic(fmt.Sprintf("transaction index %d output %d in block %q "+
			"(height %d) does not exist", txIndex, txOutIndex,
			g.tipName, g.tip.Header.Height))
	}

	txOut := tx.TxOut[txOutIndex]
	if txOut.PkScript[0] != txscript.OP_RETURN {
		panic(fmt.Sprintf("transaction index %d output %d in block %q "+
			"(height %d) is not an OP_RETURN", txIndex, txOutIndex,
			g.tipName, g.tip.Header.Height))
	}
}

// AssertStakeVersion panics if the current tip block associated with the
// generator does not have the specified stake version in the header.
func (g *Generator) AssertStakeVersion(expected uint32) {
	stakeVersion := g.tip.Header.StakeVersion
	if stakeVersion != expected {
		panic(fmt.Sprintf("stake version for block %q is %d instead of "+
			"expected %d", g.tipName, stakeVersion, expected))
	}
}

// AssertBlockVersion panics if the current tip block associated with the
// generator does not have the specified block version in the header.
func (g *Generator) AssertBlockVersion(expected int32) {
	blockVersion := g.tip.Header.Version
	if blockVersion != expected {
		panic(fmt.Sprintf("block version for block %q is %d instead of "+
			"expected %d", g.tipName, blockVersion, expected))
	}
}

// AssertPoolSize panics if the current tip block associated with the generator
// does not indicate the specified pool size.
func (g *Generator) AssertPoolSize(expected uint32) {
	poolSize := g.tip.Header.PoolSize
	if poolSize != expected {
		panic(fmt.Sprintf("pool size for block %q is %d instead of expected "+
			"%d", g.tipName, poolSize, expected))
	}
}

// AssertBlockRevocationTx panics if the current tip block associated with the
// generator does not have a revocation at the specified transaction index
// provided.
func (g *Generator) AssertBlockRevocationTx(b *wire.MsgBlock, txIndex uint32) {
	if !isRevocationTx(b.STransactions[txIndex]) {
		panic(fmt.Sprintf("stake transaction at index %d in block %q is "+
			" not a revocation", txIndex, g.tipName))
	}
}

// AssertTipNumRevocations panics if the number of revocations in header of the
// current tip block associated with the generator does not match the specified
// value.
func (g *Generator) AssertTipNumRevocations(expected uint8) {
	numRevocations := g.tip.Header.Revocations
	if numRevocations != expected {
		panic(fmt.Sprintf("number of revocations in block %q (height "+
			"%d) is %d instead of expected %d", g.tipName,
			g.tip.Header.Height, numRevocations, expected))
	}
}

// AssertTipDisapprovesPrevious panics if the current tip block associated with
// the generator does not disapprove the previous block.
func (g *Generator) AssertTipDisapprovesPrevious() {
	if g.tip.Header.VoteBits&voteBitYes == 1 {
		panic(fmt.Sprintf("block %q (height %d) does not disapprove prev block",
			g.tipName, g.tip.Header.Height))
	}
}
