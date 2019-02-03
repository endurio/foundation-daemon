// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2018 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/endurio/ndrd/chaincfg"
	"github.com/endurio/ndrd/chaincfg/chainhash"
	"github.com/endurio/ndrd/database"
	"github.com/endurio/ndrd/ndrutil"
	"github.com/endurio/ndrd/txscript"
	"github.com/endurio/ndrd/wire"
)

const (
	// MaxSigOpsPerBlock is the maximum number of signature operations
	// allowed for a block.  This really should be based upon the max
	// allowed block size for a network and any votes that might change it,
	// however, since it was not updated to be based upon it before
	// release, it will require a hard fork and associated vote agenda to
	// change it.  The original max block size for the protocol was 1MiB,
	// so that is what this is based on.
	MaxSigOpsPerBlock = 1000000 / 200

	// MaxTimeOffsetSeconds is the maximum number of seconds a block time
	// is allowed to be ahead of the current time.  This is currently 2
	// hours.
	MaxTimeOffsetSeconds = 2 * 60 * 60

	// MinCoinbaseScriptLen is the minimum length a coinbase script can be.
	MinCoinbaseScriptLen = 2

	// MaxCoinbaseScriptLen is the maximum length a coinbase script can be.
	MaxCoinbaseScriptLen = 100

	// medianTimeBlocks is the number of previous blocks which should be
	// used to calculate the median time used to validate block timestamps.
	medianTimeBlocks = 11

	// earlyVoteBitsValue is the only value of VoteBits allowed in a block
	// header before stake validation height.
	earlyVoteBitsValue = 0x0001

	// maxRevocationsPerBlock is the maximum number of revocations that are
	// allowed per block.
	maxRevocationsPerBlock = 255

	// A ticket commitment output is an OP_RETURN script with a 30-byte data
	// push that consists of a 20-byte hash for the payment hash, 8 bytes
	// for the amount to commit to (with the upper bit flag set to indicate
	// the hash is for a pay-to-script-hash address, otherwise the hash is a
	// pay-to-pubkey-hash), and 2 bytes for the fee limits.  Thus, 1 byte
	// for the OP_RETURN + 1 byte for the data push + 20 bytes for the
	// payment hash means the encoded amount is at offset 22.  Then, 8 bytes
	// for the amount means the encoded fee limits are at offset 30.
	commitHashStartIdx     = 2
	commitHashEndIdx       = commitHashStartIdx + 20
	commitAmountStartIdx   = commitHashEndIdx
	commitAmountEndIdx     = commitAmountStartIdx + 8
	commitFeeLimitStartIdx = commitAmountEndIdx
	commitFeeLimitEndIdx   = commitFeeLimitStartIdx + 2

	// commitP2SHFlag specifies the bitmask to apply to an amount decoded from
	// a ticket commitment in order to determine if it is a pay-to-script-hash
	// commitment.  The value is derived from the fact it is encoded as the most
	// significant bit in the amount.
	commitP2SHFlag = uint64(1 << 63)

	// submissionOutputIdx is the index of the stake submission output of a
	// ticket transaction.
	submissionOutputIdx = 0
)

var (
	// zeroHash is the zero value for a chainhash.Hash and is defined as a
	// package level variable to avoid the need to create a new instance
	// every time a check is needed.
	zeroHash = &chainhash.Hash{}

	// earlyFinalState is the only value of the final state allowed in a
	// block header before stake validation height.
	earlyFinalState = [6]byte{0x00}
)

// isNullOutpoint determines whether or not a previous transaction output point
// is set.
func isNullOutpoint(outpoint *wire.OutPoint) bool {
	if outpoint.Index == math.MaxUint32 &&
		outpoint.Hash.IsEqual(zeroHash) {
		return true
	}
	return false
}

// isNullFraudProof determines whether or not a previous transaction fraud
// proof is set.
func isNullFraudProof(txIn *wire.TxIn) bool {
	switch {
	case txIn.BlockHeight != wire.NullBlockHeight:
		return false
	case txIn.BlockIndex != wire.NullBlockIndex:
		return false
	}

	return true
}

// IsCoinBaseTx determines whether or not a transaction is a coinbase.  A
// coinbase is a special transaction created by miners that has no inputs.
// This is represented in the block chain by a transaction with a single input
// that has a previous output transaction index set to the maximum value along
// with a zero hash.
//
// This function only differs from IsCoinBase in that it works with a raw wire
// transaction as opposed to a higher level util transaction.
func IsCoinBaseTx(msgTx *wire.MsgTx) bool {
	// A coin base must only have one transaction input.
	if len(msgTx.TxIn) != 1 {
		return false
	}

	// The previous output of a coin base must have a max value index and a
	// zero hash.
	prevOut := &msgTx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || !prevOut.Hash.IsEqual(zeroHash) {
		return false
	}

	return true
}

// IsCoinBase determines whether or not a transaction is a coinbase.  A
// coinbase is a special transaction created by miners that has no inputs.
// This is represented in the block chain by a transaction with a single input
// that has a previous output transaction index set to the maximum value along
// with a zero hash.
//
// This function only differs from IsCoinBaseTx in that it works with a higher
// level util transaction as opposed to a raw wire transaction.
func IsCoinBase(tx *ndrutil.Tx) bool {
	return IsCoinBaseTx(tx.MsgTx())
}

// IsExpiredTx returns where or not the passed transaction is expired according
// to the given block height.
//
// This function only differs from IsExpired in that it works with a raw wire
// transaction as opposed to a higher level util transaction.
func IsExpiredTx(tx *wire.MsgTx, blockHeight int64) bool {
	expiry := tx.Expiry
	return expiry != wire.NoExpiryValue && blockHeight >= int64(expiry)
}

// IsExpired returns where or not the passed transaction is expired according to
// the given block height.
//
// This function only differs from IsExpiredTx in that it works with a higher
// level util transaction as opposed to a raw wire transaction.
func IsExpired(tx *ndrutil.Tx, blockHeight int64) bool {
	return IsExpiredTx(tx.MsgTx(), blockHeight)
}

// SequenceLockActive determines if all of the inputs to a given transaction
// have achieved a relative age that surpasses the requirements specified by
// their respective sequence locks as calculated by CalcSequenceLock.  A single
// sequence lock is sufficient because the calculated lock selects the minimum
// required time and block height from all of the non-disabled inputs after
// which the transaction can be included.
func SequenceLockActive(lock *SequenceLock, blockHeight int64, medianTime time.Time) bool {
	// The transaction is not yet mature if it has not yet reached the
	// required minimum time and block height according to its sequence
	// locks.
	if blockHeight <= lock.MinHeight || medianTime.Unix() <= lock.MinTime {
		return false
	}

	return true
}

// IsFinalizedTransaction determines whether or not a transaction is finalized.
func IsFinalizedTransaction(tx *ndrutil.Tx, blockHeight int64, blockTime time.Time) bool {
	// Lock time of zero means the transaction is finalized.
	msgTx := tx.MsgTx()
	lockTime := msgTx.LockTime
	if lockTime == 0 {
		return true
	}

	// The lock time field of a transaction is either a block height at
	// which the transaction is finalized or a timestamp depending on if the
	// value is before the txscript.LockTimeThreshold.  When it is under the
	// threshold it is a block height.
	var blockTimeOrHeight int64
	if lockTime < txscript.LockTimeThreshold {
		blockTimeOrHeight = blockHeight
	} else {
		blockTimeOrHeight = blockTime.Unix()
	}
	if int64(lockTime) < blockTimeOrHeight {
		return true
	}

	// At this point, the transaction's lock time hasn't occurred yet, but
	// the transaction might still be finalized if the sequence number
	// for all transaction inputs is maxed out.
	for _, txIn := range msgTx.TxIn {
		if txIn.Sequence != math.MaxUint32 {
			return false
		}
	}
	return true
}

// CheckTransactionSanity performs some preliminary checks on a transaction to
// ensure it is sane.  These checks are context free.
func CheckTransactionSanity(tx *wire.MsgTx, params *chaincfg.Params) error {
	// A transaction must have at least one input.
	if len(tx.TxIn) == 0 {
		return ruleError(ErrNoTxInputs, "transaction has no inputs")
	}

	// A transaction must have at least one output.
	if len(tx.TxOut) == 0 {
		return ruleError(ErrNoTxOutputs, "transaction has no outputs")
	}

	// A transaction must not exceed the maximum allowed size when
	// serialized.
	serializedTxSize := tx.SerializeSize()
	if serializedTxSize > params.MaxTxSize {
		str := fmt.Sprintf("serialized transaction is too big - got "+
			"%d, max %d", serializedTxSize, params.MaxTxSize)
		return ruleError(ErrTxTooBig, str)
	}

	// Coinbase script length must be between min and max length.
	if IsCoinBaseTx(tx) {
		// The referenced outpoint must be null.
		if !isNullOutpoint(&tx.TxIn[0].PreviousOutPoint) {
			str := fmt.Sprintf("coinbase transaction did not use " +
				"a null outpoint")
			return ruleError(ErrBadCoinbaseOutpoint, str)
		}

		// The fraud proof must also be null.
		if !isNullFraudProof(tx.TxIn[0]) {
			str := fmt.Sprintf("coinbase transaction fraud proof " +
				"was non-null")
			return ruleError(ErrBadCoinbaseFraudProof, str)
		}

		slen := len(tx.TxIn[0].SignatureScript)
		if slen < MinCoinbaseScriptLen || slen > MaxCoinbaseScriptLen {
			str := fmt.Sprintf("coinbase transaction script "+
				"length of %d is out of range (min: %d, max: "+
				"%d)", slen, MinCoinbaseScriptLen,
				MaxCoinbaseScriptLen)
			return ruleError(ErrBadCoinbaseScriptLen, str)
		}
	} else {
		// Previous transaction outputs referenced by the inputs to
		// this transaction must not be null except in the case of
		// stakebases for votes.
		for _, txIn := range tx.TxIn {
			prevOut := &txIn.PreviousOutPoint
			if isNullOutpoint(prevOut) {
				return ruleError(ErrBadTxInput, "transaction "+
					"input refers to previous output that "+
					"is null")
			}
		}
	}

	var totalAtom int64
	for _, txOut := range tx.TxOut {
		atom := txOut.Value
		if atom < 0 {
			str := fmt.Sprintf("transaction output has negative value of %v",
				atom)
			return ruleError(ErrBadTxOutValue, str)
		}
		if atom > ndrutil.MaxAmount {
			str := fmt.Sprintf("transaction output value of %v is higher than "+
				"max allowed value of %v", atom, ndrutil.MaxAmount)
			return ruleError(ErrBadTxOutValue, str)
		}

		// Two's complement int64 overflow guarantees that any overflow is
		// detected and reported.  This is impossible for Decred, but perhaps
		// possible if an alt increases the total money supply.
		totalAtom += atom
		if totalAtom < 0 {
			str := fmt.Sprintf("total value of all transaction outputs "+
				"exceeds max allowed value of %v", ndrutil.MaxAmount)
			return ruleError(ErrBadTxOutValue, str)
		}
		if totalAtom > ndrutil.MaxAmount {
			str := fmt.Sprintf("total value of all transaction outputs is %v "+
				"which is higher than max allowed value of %v", totalAtom,
				ndrutil.MaxAmount)
			return ruleError(ErrBadTxOutValue, str)
		}
	}

	// Check for duplicate transaction inputs.
	existingTxOut := make(map[wire.OutPoint]struct{})
	for _, txIn := range tx.TxIn {
		if _, exists := existingTxOut[txIn.PreviousOutPoint]; exists {
			return ruleError(ErrDuplicateTxInputs, "transaction "+
				"contains duplicate inputs")
		}
		existingTxOut[txIn.PreviousOutPoint] = struct{}{}
	}

	return nil
}

// checkProofOfWork ensures the block header bits which indicate the target
// difficulty is in min/max range and that the block hash is less than the
// target difficulty as claimed.
//
// The flags modify the behavior of this function as follows:
//  - BFNoPoWCheck: The check to ensure the block hash is less than the target
//    difficulty is not performed.
func checkProofOfWork(header *wire.BlockHeader, powLimit *big.Int, flags BehaviorFlags) error {
	// The target difficulty must be larger than zero.
	target := CompactToBig(header.Bits)
	if target.Sign() <= 0 {
		str := fmt.Sprintf("block target difficulty of %064x is too "+
			"low", target)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

	// The target difficulty must be less than the maximum allowed.
	if target.Cmp(powLimit) > 0 {
		str := fmt.Sprintf("block target difficulty of %064x is "+
			"higher than max of %064x", target, powLimit)
		return ruleError(ErrUnexpectedDifficulty, str)
	}

	// The block hash must be less than the claimed target unless the flag
	// to avoid proof of work checks is set.
	if flags&BFNoPoWCheck != BFNoPoWCheck {
		// The block hash must be less than the claimed target.
		hash := header.BlockHash()
		hashNum := HashToBig(&hash)
		if hashNum.Cmp(target) > 0 {
			str := fmt.Sprintf("block hash of %064x is higher than"+
				" expected max of %064x", hashNum, target)
			return ruleError(ErrHighHash, str)
		}
	}

	return nil
}

// CheckProofOfWork ensures the block header bits which indicate the target
// difficulty is in min/max range and that the block hash is less than the
// target difficulty as claimed.
func CheckProofOfWork(header *wire.BlockHeader, powLimit *big.Int) error {
	return checkProofOfWork(header, powLimit, BFNone)
}

// checkBlockHeaderSanity performs some preliminary checks on a block header to
// ensure it is sane before continuing with processing.  These checks are
// context free.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to checkProofOfWork.
func checkBlockHeaderSanity(header *wire.BlockHeader, timeSource MedianTimeSource, flags BehaviorFlags, chainParams *chaincfg.Params) error {
	// Ensure the proof of work bits in the block header is in min/max
	// range and the block hash is less than the target value described by
	// the bits.
	err := checkProofOfWork(header, chainParams.PowLimit, flags)
	if err != nil {
		return err
	}

	// A block timestamp must not have a greater precision than one second.
	// This check is necessary because Go time.Time values support
	// nanosecond precision whereas the consensus rules only apply to
	// seconds and it's much nicer to deal with standard Go time values
	// instead of converting to seconds everywhere.
	if !header.Timestamp.Equal(time.Unix(header.Timestamp.Unix(), 0)) {
		str := fmt.Sprintf("block timestamp of %v has a higher "+
			"precision than one second", header.Timestamp)
		return ruleError(ErrInvalidTime, str)
	}

	// Ensure the block time is not too far in the future.
	maxTimestamp := timeSource.AdjustedTime().Add(time.Second *
		MaxTimeOffsetSeconds)
	if header.Timestamp.After(maxTimestamp) {
		str := fmt.Sprintf("block timestamp of %v is too far in the "+
			"future", header.Timestamp)
		return ruleError(ErrTimeTooNew, str)
	}

	return nil
}

// checkBlockSanity performs some preliminary checks on a block to ensure it is
// sane before continuing with block processing.  These checks are context
// free.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to checkBlockHeaderSanity.
func checkBlockSanity(block *ndrutil.Block, timeSource MedianTimeSource, flags BehaviorFlags, chainParams *chaincfg.Params) error {
	msgBlock := block.MsgBlock()
	header := &msgBlock.Header
	err := checkBlockHeaderSanity(header, timeSource, flags, chainParams)
	if err != nil {
		return err
	}

	// A block must have at least one regular transaction.
	numTx := len(msgBlock.Transactions)
	if numTx == 0 {
		return ruleError(ErrNoTransactions, "block does not contain "+
			"any transactions")
	}

	// A block must not exceed the maximum allowed block payload when
	// serialized.
	//
	// This is a quick and context-free sanity check of the maximum block
	// size according to the wire protocol.  Even though the wire protocol
	// already prevents blocks bigger than this limit, there are other
	// methods of receiving a block that might not have been checked
	// already.  A separate block size is enforced later that takes into
	// account the network-specific block size and the results of block
	// size votes.  Typically that block size is more restrictive than this
	// one.
	serializedSize := msgBlock.SerializeSize()
	if serializedSize > wire.MaxBlockPayload {
		str := fmt.Sprintf("serialized block is too big - got %d, "+
			"max %d", serializedSize, wire.MaxBlockPayload)
		return ruleError(ErrBlockTooBig, str)
	}

	// The first transaction in a block's regular tree must be a coinbase.
	transactions := block.Transactions()
	if !IsCoinBaseTx(transactions[0].MsgTx()) {
		return ruleError(ErrFirstTxNotCoinbase, "first transaction in "+
			"block is not a coinbase")
	}

	// A block must not have more than one coinbase.
	for i, tx := range transactions[1:] {
		if IsCoinBaseTx(tx.MsgTx()) {
			str := fmt.Sprintf("block contains second coinbase at "+
				"index %d", i+1)
			return ruleError(ErrMultipleCoinbases, str)
		}
	}

	// Do some preliminary checks on each regular transaction to ensure they
	// are sane before continuing.
	for _, tx := range transactions {
		// A block must not have stake transactions in the regular
		// transaction tree.
		msgTx := tx.MsgTx()

		err := CheckTransactionSanity(msgTx, chainParams)
		if err != nil {
			return err
		}
	}

	// Build merkle tree and ensure the calculated merkle root matches the
	// entry in the block header.  This also has the effect of caching all
	// of the transaction hashes in the block to speed up future hash
	// checks.  Bitcoind builds the tree here and checks the merkle root
	// after the following checks, but there is no reason not to check the
	// merkle root matches here.
	merkles := BuildMerkleTreeStore(block.Transactions())
	calculatedMerkleRoot := merkles[len(merkles)-1]
	if !header.MerkleRoot.IsEqual(calculatedMerkleRoot) {
		str := fmt.Sprintf("block merkle root is invalid - block "+
			"header indicates %v, but calculated value is %v",
			header.MerkleRoot, calculatedMerkleRoot)
		return ruleError(ErrBadMerkleRoot, str)
	}

	// Check for duplicate transactions.  This check will be fairly quick
	// since the transaction hashes are already cached due to building the
	// merkle trees above.
	existingTxHashes := make(map[chainhash.Hash]struct{})
	allTransactions := transactions

	for _, tx := range allTransactions {
		hash := tx.Hash()
		if _, exists := existingTxHashes[*hash]; exists {
			str := fmt.Sprintf("block contains duplicate "+
				"transaction %v", hash)
			return ruleError(ErrDuplicateTx, str)
		}
		existingTxHashes[*hash] = struct{}{}
	}

	// The number of signature operations must be less than the maximum
	// allowed per block.
	totalSigOps := 0
	for _, tx := range allTransactions {
		// We could potentially overflow the accumulator so check for
		// overflow.
		lastSigOps := totalSigOps

		msgTx := tx.MsgTx()
		isCoinBase := IsCoinBaseTx(msgTx)
		totalSigOps += CountSigOps(tx, isCoinBase)
		if totalSigOps < lastSigOps || totalSigOps > MaxSigOpsPerBlock {
			str := fmt.Sprintf("block contains too many signature "+
				"operations - got %v, max %v", totalSigOps,
				MaxSigOpsPerBlock)
			return ruleError(ErrTooManySigOps, str)
		}
	}

	return nil
}

// CheckBlockSanity performs some preliminary checks on a block to ensure it is
// sane before continuing with block processing.  These checks are context
// free.
func CheckBlockSanity(block *ndrutil.Block, timeSource MedianTimeSource, chainParams *chaincfg.Params) error {
	return checkBlockSanity(block, timeSource, BFNone, chainParams)
}

// checkBlockHeaderPositional performs several validation checks on the block
// header which depend on its position within the block chain and having the
// headers of all ancestors available.  These checks do not, and must not, rely
// on having the full block data of all ancestors available.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: All checks except those involving comparing the header against
//    the checkpoints and expected height are not performed.
//
// This function MUST be called with the chain state lock held (for reads).
func (b *BlockChain) checkBlockHeaderPositional(header *wire.BlockHeader, prevNode *blockNode, flags BehaviorFlags) error {
	// The genesis block is valid by definition.
	if prevNode == nil {
		return nil
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		// Ensure the difficulty specified in the block header matches
		// the calculated difficulty based on the previous block and
		// difficulty retarget rules.
		expDiff, err := b.calcNextRequiredDifficulty(prevNode,
			header.Timestamp)
		if err != nil {
			return err
		}
		blockDifficulty := header.Bits
		if blockDifficulty != expDiff {
			str := fmt.Sprintf("block difficulty of %d is not the"+
				" expected value of %d", blockDifficulty,
				expDiff)
			return ruleError(ErrUnexpectedDifficulty, str)
		}

		// Ensure the timestamp for the block header is after the
		// median time of the last several blocks (medianTimeBlocks).
		medianTime := prevNode.CalcPastMedianTime()
		if !header.Timestamp.After(medianTime) {
			str := "block timestamp of %v is not after expected %v"
			str = fmt.Sprintf(str, header.Timestamp, medianTime)
			return ruleError(ErrTimeTooOld, str)
		}
	}

	// The height of this block is one more than the referenced previous
	// block.
	blockHeight := prevNode.height + 1

	// Ensure chain matches up to predetermined checkpoints.
	blockHash := header.BlockHash()
	if !b.verifyCheckpoint(blockHeight, &blockHash) {
		str := fmt.Sprintf("block at height %d does not match "+
			"checkpoint hash", blockHeight)
		return ruleError(ErrBadCheckpoint, str)
	}

	// Find the previous checkpoint and prevent blocks which fork the main
	// chain before it.  This prevents storage of new, otherwise valid,
	// blocks which build off of old blocks that are likely at a much easier
	// difficulty and therefore could be used to waste cache and disk space.
	checkpointNode, err := b.findPreviousCheckpoint()
	if err != nil {
		return err
	}
	if checkpointNode != nil && blockHeight < checkpointNode.height {
		str := fmt.Sprintf("block at height %d forks the main chain "+
			"before the previous checkpoint at height %d",
			blockHeight, checkpointNode.height)
		return ruleError(ErrForkTooOld, str)
	}

	if !fastAdd {
		// Reject version 5 blocks for networks other than the main
		// network once a majority of the network has upgraded.
		if b.chainParams.Net != wire.MainNet && header.Version < 6 &&
			b.isMajorityVersion(6, prevNode, b.chainParams.BlockRejectNumRequired) {

			str := "new blocks with version %d are no longer valid"
			str = fmt.Sprintf(str, header.Version)
			return ruleError(ErrBlockVersionTooOld, str)
		}

		// Reject version 4 blocks once a majority of the network has
		// upgraded.
		if header.Version < 5 && b.isMajorityVersion(5, prevNode,
			b.chainParams.BlockRejectNumRequired) {

			str := "new blocks with version %d are no longer valid"
			str = fmt.Sprintf(str, header.Version)
			return ruleError(ErrBlockVersionTooOld, str)
		}

		// Reject version 3 blocks once a majority of the network has
		// upgraded.
		if header.Version < 4 && b.isMajorityVersion(4, prevNode,
			b.chainParams.BlockRejectNumRequired) {

			str := "new blocks with version %d are no longer valid"
			str = fmt.Sprintf(str, header.Version)
			return ruleError(ErrBlockVersionTooOld, str)
		}

		// Reject version 2 blocks once a majority of the network has
		// upgraded.
		if header.Version < 3 && b.isMajorityVersion(3, prevNode,
			b.chainParams.BlockRejectNumRequired) {

			str := "new blocks with version %d are no longer valid"
			str = fmt.Sprintf(str, header.Version)
			return ruleError(ErrBlockVersionTooOld, str)
		}

		// Reject version 1 blocks once a majority of the network has
		// upgraded.
		if header.Version < 2 && b.isMajorityVersion(2, prevNode,
			b.chainParams.BlockRejectNumRequired) {

			str := "new blocks with version %d are no longer valid"
			str = fmt.Sprintf(str, header.Version)
			return ruleError(ErrBlockVersionTooOld, str)
		}
	}

	return nil
}

// checkBlockPositional performs several validation checks on the block which
// depend on its position within the block chain and having the headers of all
// ancestors available.  These checks do not, and must not, rely on having the
// full block data of all ancestors available.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: The transactions are not checked to see if they are expired and
//    the coinbae height check is not performed.
//
// The flags are also passed to checkBlockHeaderPositional.  See its
// documentation for how the flags modify its behavior.
func (b *BlockChain) checkBlockPositional(block *ndrutil.Block, prevNode *blockNode, flags BehaviorFlags) error {
	// The genesis block is valid by definition.
	if prevNode == nil {
		return nil
	}

	// Perform all block header related validation checks that depend on its
	// position within the block chain and having the headers of all
	// ancestors available, but do not rely on having the full block data of
	// all ancestors available.
	header := &block.MsgBlock().Header
	err := b.checkBlockHeaderPositional(header, prevNode, flags)
	if err != nil {
		return err
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		// The height of this block is one more than the referenced
		// previous block.
		blockHeight := prevNode.height + 1

		// Ensure all transactions in the block are not expired.
		for _, tx := range block.Transactions() {
			if IsExpired(tx, blockHeight) {
				errStr := fmt.Sprintf("block contains expired regular "+
					"transaction %v (expiration height %d)", tx.Hash(),
					tx.MsgTx().Expiry)
				return ruleError(ErrExpiredTx, errStr)
			}
		}

		// Check that the coinbase contains at minimum the block
		// height in output 1.
		if blockHeight > 1 {
			err := checkCoinbaseUniqueHeight(blockHeight, block)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// checkBlockHeaderContext performs several validation checks on the block
// header which depend on having the full block data for all of its ancestors
// available.  This includes checks which depend on tallying the results of
// votes, because votes are part of the block data.
//
// It should be noted that rule changes that have become buried deep enough
// typically will eventually be transitioned to using well-known activation
// points for efficiency purposes at which point the associated checks no longer
// require having direct access to the historical votes, and therefore may be
// transitioned to checkBlockHeaderPositional at that time.  Conversely, any
// checks in that function which become conditional based on the results of a
// vote will necessarily need to be transitioned to this function.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: No check are performed.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) checkBlockHeaderContext(header *wire.BlockHeader, prevNode *blockNode, flags BehaviorFlags) error {
	// The genesis block is valid by definition.
	if prevNode == nil {
		return nil
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		// removed stake check
	}

	return nil
}

// checkCoinbaseUniqueHeight checks to ensure that for all blocks height > 1 the
// coinbase contains the height encoding to make coinbase hash collisions
// impossible.
func checkCoinbaseUniqueHeight(blockHeight int64, block *ndrutil.Block) error {
	// Coinbase TxOut[0] is always tax, TxOut[1] is always
	// height + extranonce, so at least two outputs must
	// exist.
	if len(block.MsgBlock().Transactions[0].TxOut) < 2 {
		str := fmt.Sprintf("block %v is missing necessary coinbase "+
			"outputs", block.Hash())
		return ruleError(ErrFirstTxNotCoinbase, str)
	}

	// Only version 0 scripts are currently valid.
	nullDataOut := block.MsgBlock().Transactions[0].TxOut[1]
	if nullDataOut.Version != 0 {
		str := fmt.Sprintf("block %v output 1 has wrong script version",
			block.Hash())
		return ruleError(ErrFirstTxNotCoinbase, str)
	}

	// The first 4 bytes of the null data output must be the encoded height
	// of the block, so that every coinbase created has a unique transaction
	// hash.
	nullData, err := txscript.ExtractCoinbaseNullData(nullDataOut.PkScript)
	if err != nil {
		str := fmt.Sprintf("block %v output 1 has wrong script type",
			block.Hash())
		return ruleError(ErrFirstTxNotCoinbase, str)
	}
	if len(nullData) < 4 {
		str := fmt.Sprintf("block %v output 1 data push too short to "+
			"contain height", block.Hash())
		return ruleError(ErrFirstTxNotCoinbase, str)
	}

	// Check the height and ensure it is correct.
	cbHeight := binary.LittleEndian.Uint32(nullData[0:4])
	if cbHeight != uint32(blockHeight) {
		prevBlock := block.MsgBlock().Header.PrevBlock
		str := fmt.Sprintf("block %v output 1 has wrong height in "+
			"coinbase; want %v, got %v; prevBlock %v, header height %v",
			block.Hash(), blockHeight, cbHeight, prevBlock,
			block.MsgBlock().Header.Height)
		return ruleError(ErrCoinbaseHeight, str)
	}

	return nil
}

// checkBlockContext performs several validation checks on the block which depend
// on having the full block data for all of its ancestors available.  This
// includes checks which depend on tallying the results of votes, because votes
// are part of the block data.
//
// It should be noted that rule changes that have become buried deep enough
// typically will eventually be transitioned to using well-known activation
// points for efficiency purposes at which point the associated checks no longer
// require having direct access to the historical votes, and therefore may be
// transitioned to checkBlockPositional at that time.  Conversely, any checks
// in that function which become conditional based on the results of a vote will
// necessarily need to be transitioned to this function.
//
// The flags modify the behavior of this function as follows:
//  - BFFastAdd: The max block size is not checked, transactions are not checked
//    to see if they are finalized, and the included votes and revocations are
//    not verified to be allowed.
//
// The flags are also passed to checkBlockHeaderContext.  See its documentation
// for how the flags modify its behavior.
func (b *BlockChain) checkBlockContext(block *ndrutil.Block, prevNode *blockNode, flags BehaviorFlags) error {
	// The genesis block is valid by definition.
	if prevNode == nil {
		return nil
	}

	// Perform all block header related validation checks which depend on
	// having the full block data for all of its ancestors available.
	header := &block.MsgBlock().Header
	err := b.checkBlockHeaderContext(header, prevNode, flags)
	if err != nil {
		return err
	}

	fastAdd := flags&BFFastAdd == BFFastAdd
	if !fastAdd {
		// A block must not exceed the maximum allowed size as defined
		// by the network parameters and the current status of any hard
		// fork votes to change it when serialized.
		maxBlockSize, err := b.maxBlockSize(prevNode)
		if err != nil {
			return err
		}
		serializedSize := int64(block.MsgBlock().SerializeSize())
		if serializedSize > maxBlockSize {
			str := fmt.Sprintf("serialized block is too big - "+
				"got %d, max %d", serializedSize,
				maxBlockSize)
			return ruleError(ErrBlockTooBig, str)
		}

		blockTime := prevNode.CalcPastMedianTime()

		// The height of this block is one more than the referenced
		// previous block.
		blockHeight := prevNode.height + 1

		// Ensure all transactions in the block are finalized.
		for _, tx := range block.Transactions() {
			if !IsFinalizedTransaction(tx, blockHeight, blockTime) {
				str := fmt.Sprintf("block contains unfinalized regular "+
					"transaction %v", tx.Hash())
				return ruleError(ErrUnfinalizedTx, str)
			}
		}
	}

	return nil
}

// checkDupTxs ensures blocks do not contain duplicate transactions which
// 'overwrite' older transactions that are not fully spent.  This prevents an
// attack where a coinbase and all of its dependent transactions could be
// duplicated to effectively revert the overwritten transactions to a single
// confirmation thereby making them vulnerable to a double spend.
//
// For more details, see https://en.bitcoin.it/wiki/BIP_0030 and
// http://r6.ca/blog/20120206T005236Z.html.
//
// Decred: Check the stake transactions to make sure they don't have this txid
// too.
func (b *BlockChain) checkDupTxs(txSet []*ndrutil.Tx, view *UtxoViewpoint) error {
	if !chaincfg.CheckForDuplicateHashes {
		return nil
	}

	// Fetch utxo details for all of the transactions in this block.
	// Typically, there will not be any utxos for any of the transactions.
	filteredSet := make(viewFilteredSet)
	for _, tx := range txSet {
		filteredSet.add(view, tx.Hash())
	}
	err := view.fetchUtxosMain(b.db, filteredSet)
	if err != nil {
		return err
	}

	// Duplicate transactions are only allowed if the previous transaction
	// is fully spent.
	for _, tx := range txSet {
		txEntry := view.LookupEntry(tx.Hash())
		if txEntry != nil && !txEntry.IsFullySpent() {
			str := fmt.Sprintf("tried to overwrite transaction %v "+
				"at block height %d that is not fully spent",
				tx.Hash(), txEntry.BlockHeight())
			return ruleError(ErrOverwriteTx, str)
		}
	}

	return nil
}

// CheckTransactionInputs performs a series of checks on the inputs to a
// transaction to ensure they are valid.  An example of some of the checks
// include verifying all inputs exist, ensuring the coinbase seasoning
// requirements are met, detecting double spends, validating all values and
// fees are in the legal range and the total output amount doesn't exceed the
// input amount, and verifying the signatures to prove the spender was the
// owner of the Decred and therefore allowed to spend them.  As it checks the
// inputs, it also calculates the total fees for the transaction and returns
// that value.
//
// NOTE: The transaction MUST have already been sanity checked with the
// CheckTransactionSanity function prior to calling this function.
func CheckTransactionInputs(subsidyCache *SubsidyCache, tx *ndrutil.Tx, txHeight int64, view *UtxoViewpoint, checkFraudProof bool, chainParams *chaincfg.Params) (int64, error) {
	// Coinbase transactions have no inputs.
	if IsCoinBase(tx) {
		return 0, nil
	}

	// -------------------------------------------------------------------
	// Decred stake transaction testing.
	// -------------------------------------------------------------------

	// Perform additional checks on ticket purchase transactions such as
	// ensuring the input type requirements are met and the output commitments
	// coincide with the inputs.
	msgTx := tx.MsgTx()

	// -------------------------------------------------------------------
	// Decred general transaction testing (and a few stake exceptions).
	// -------------------------------------------------------------------

	txHash := tx.Hash()
	var totalAtomIn int64
	for idx, txIn := range msgTx.TxIn {
		txInHash := &txIn.PreviousOutPoint.Hash
		originTxIndex := txIn.PreviousOutPoint.Index
		utxoEntry := view.LookupEntry(txInHash)
		if utxoEntry == nil || utxoEntry.IsOutputSpent(originTxIndex) {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				txHash, idx)
			return 0, ruleError(ErrMissingTxOut, str)
		}

		// Check fraud proof witness data.

		// Using zero value outputs as inputs is banned.
		if utxoEntry.AmountByIndex(originTxIndex) == 0 {
			str := fmt.Sprintf("tried to spend zero value output "+
				"from input %v, idx %v", txInHash,
				originTxIndex)
			return 0, ruleError(ErrZeroValueOutputSpend, str)
		}

		if checkFraudProof {
			if txIn.ValueIn !=
				utxoEntry.AmountByIndex(originTxIndex) {
				str := fmt.Sprintf("bad fraud check value in "+
					"(expected %v, given %v) for txIn %v",
					utxoEntry.AmountByIndex(originTxIndex),
					txIn.ValueIn, idx)
				return 0, ruleError(ErrFraudAmountIn, str)
			}

			if int64(txIn.BlockHeight) != utxoEntry.BlockHeight() {
				str := fmt.Sprintf("bad fraud check block "+
					"height (expected %v, given %v) for "+
					"txIn %v", utxoEntry.BlockHeight(),
					txIn.BlockHeight, idx)
				return 0, ruleError(ErrFraudBlockHeight, str)
			}

			if txIn.BlockIndex != utxoEntry.BlockIndex() {
				str := fmt.Sprintf("bad fraud check block "+
					"index (expected %v, given %v) for "+
					"txIn %v", utxoEntry.BlockIndex(),
					txIn.BlockIndex, idx)
				return 0, ruleError(ErrFraudBlockIndex, str)
			}
		}

		// Ensure the transaction is not spending coins which have not
		// yet reached the required coinbase maturity.
		coinbaseMaturity := int64(chainParams.CoinbaseMaturity)
		if utxoEntry.IsCoinBase() {
			originHeight := utxoEntry.BlockHeight()
			blocksSincePrev := txHeight - originHeight
			if blocksSincePrev < coinbaseMaturity {
				str := fmt.Sprintf("tx %v tried to spend "+
					"coinbase transaction %v from height "+
					"%v at height %v before required "+
					"maturity of %v blocks", txHash,
					txInHash, originHeight, txHeight,
					coinbaseMaturity)
				return 0, ruleError(ErrImmatureSpend, str)
			}
		}

		// Ensure that the transaction is not spending coins from a
		// transaction that included an expiry but which has not yet
		// reached coinbase maturity many blocks.
		if utxoEntry.HasExpiry() {
			originHeight := utxoEntry.BlockHeight()
			blocksSincePrev := txHeight - originHeight
			if blocksSincePrev < coinbaseMaturity {
				str := fmt.Sprintf("tx %v tried to spend "+
					"transaction %v including an expiry "+
					"from height %v at height %v before "+
					"required maturity of %v blocks",
					txHash, txInHash, originHeight,
					txHeight, coinbaseMaturity)
				return 0, ruleError(ErrExpiryTxSpentEarly, str)
			}
		}

		// Ensure the transaction amounts are in range.  Each of the
		// output values of the input transactions must not be negative
		// or more than the max allowed per transaction.  All amounts
		// in a transaction are in a unit value known as an atom.  One
		// Decred is a quantity of atoms as defined by the AtomPerCoin
		// constant.
		originTxAtom := utxoEntry.AmountByIndex(originTxIndex)
		if originTxAtom < 0 {
			str := fmt.Sprintf("transaction output has negative "+
				"value of %v", originTxAtom)
			return 0, ruleError(ErrBadTxOutValue, str)
		}
		if originTxAtom > ndrutil.MaxAmount {
			str := fmt.Sprintf("transaction output value of %v is "+
				"higher than max allowed value of %v",
				originTxAtom, ndrutil.MaxAmount)
			return 0, ruleError(ErrBadTxOutValue, str)
		}

		// The total of all outputs must not be more than the max
		// allowed per transaction.  Also, we could potentially
		// overflow the accumulator so check for overflow.
		lastAtomIn := totalAtomIn
		totalAtomIn += originTxAtom
		if totalAtomIn < lastAtomIn ||
			totalAtomIn > ndrutil.MaxAmount {
			str := fmt.Sprintf("total value of all transaction "+
				"inputs is %v which is higher than max "+
				"allowed value of %v", totalAtomIn,
				ndrutil.MaxAmount)
			return 0, ruleError(ErrBadTxOutValue, str)
		}
	}

	// Calculate the total output amount for this transaction.  It is safe
	// to ignore overflow and out of range errors here because those error
	// conditions would have already been caught by checkTransactionSanity.
	var totalAtomOut int64
	for _, txOut := range tx.MsgTx().TxOut {
		totalAtomOut += txOut.Value
	}

	// Ensure the transaction does not spend more than its inputs.
	if totalAtomIn < totalAtomOut {
		str := fmt.Sprintf("total value of all transaction inputs for "+
			"transaction %v is %v which is less than the amount "+
			"spent of %v", txHash, totalAtomIn, totalAtomOut)
		return 0, ruleError(ErrSpendTooHigh, str)
	}

	txFeeInAtom := totalAtomIn - totalAtomOut
	return txFeeInAtom, nil
}

// CountSigOps returns the number of signature operations for all transaction
// input and output scripts in the provided transaction.  This uses the
// quicker, but imprecise, signature operation counting mechanism from
// txscript.
func CountSigOps(tx *ndrutil.Tx, isCoinBaseTx bool) int {
	msgTx := tx.MsgTx()

	// Accumulate the number of signature operations in all transaction
	// inputs.
	totalSigOps := 0
	for _, txIn := range msgTx.TxIn {
		// Skip coinbase inputs.
		if isCoinBaseTx {
			continue
		}

		numSigOps := txscript.GetSigOpCount(txIn.SignatureScript)
		totalSigOps += numSigOps
	}

	// Accumulate the number of signature operations in all transaction
	// outputs.
	for _, txOut := range msgTx.TxOut {
		numSigOps := txscript.GetSigOpCount(txOut.PkScript)
		totalSigOps += numSigOps
	}

	return totalSigOps
}

// CountP2SHSigOps returns the number of signature operations for all input
// transactions which are of the pay-to-script-hash type.  This uses the
// precise, signature operation counting mechanism from the script engine which
// requires access to the input transaction scripts.
func CountP2SHSigOps(tx *ndrutil.Tx, isCoinBaseTx bool, view *UtxoViewpoint) (int, error) {
	// Coinbase transactions have no interesting inputs.
	if isCoinBaseTx {
		return 0, nil
	}

	// Accumulate the number of signature operations in all transaction
	// inputs.
	msgTx := tx.MsgTx()
	totalSigOps := 0
	for txInIndex, txIn := range msgTx.TxIn {
		// Ensure the referenced input transaction is available.
		originTxHash := &txIn.PreviousOutPoint.Hash
		originTxIndex := txIn.PreviousOutPoint.Index
		utxoEntry := view.LookupEntry(originTxHash)
		if utxoEntry == nil || utxoEntry.IsOutputSpent(originTxIndex) {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return 0, ruleError(ErrMissingTxOut, str)
		}

		// We're only interested in pay-to-script-hash types, so skip
		// this input if it's not one.
		pkScript := utxoEntry.PkScriptByIndex(originTxIndex)
		if !txscript.IsPayToScriptHash(pkScript) {
			continue
		}

		// Count the precise number of signature operations in the
		// referenced public key script.
		sigScript := txIn.SignatureScript
		numSigOps := txscript.GetPreciseSigOpCount(sigScript, pkScript,
			true)

		// We could potentially overflow the accumulator so check for
		// overflow.
		lastSigOps := totalSigOps
		totalSigOps += numSigOps
		if totalSigOps < lastSigOps {
			str := fmt.Sprintf("the public key script from output "+
				"%v contains too many signature operations - "+
				"overflow", txIn.PreviousOutPoint)
			return 0, ruleError(ErrTooManySigOps, str)
		}
	}

	return totalSigOps, nil
}

// checkNumSigOps Checks the number of P2SH signature operations to make
// sure they don't overflow the limits.  It takes a cumulative number of sig
// ops as an argument and increments will each call.
// TxTree true == Regular, false == Stake
func checkNumSigOps(tx *ndrutil.Tx, view *UtxoViewpoint, index int, cumulativeSigOps int) (int, error) {
	numsigOps := CountSigOps(tx, index == 0)

	// Since the first (and only the first) transaction has already been
	// verified to be a coinbase transaction, use (i == 0) && TxTree as an
	// optimization for the flag to countP2SHSigOps for whether or not the
	// transaction is a coinbase transaction rather than having to do a
	// full coinbase check again.
	numP2SHSigOps, err := CountP2SHSigOps(tx, index == 0, view)
	if err != nil {
		log.Tracef("CountP2SHSigOps failed; error returned %v", err)
		return 0, err
	}

	startCumSigOps := cumulativeSigOps
	cumulativeSigOps += numsigOps
	cumulativeSigOps += numP2SHSigOps

	// Check for overflow or going over the limits.  We have to do
	// this on every loop iteration to avoid overflow.
	if cumulativeSigOps < startCumSigOps ||
		cumulativeSigOps > MaxSigOpsPerBlock {
		str := fmt.Sprintf("block contains too many signature "+
			"operations - got %v, max %v", cumulativeSigOps,
			MaxSigOpsPerBlock)
		return 0, ruleError(ErrTooManySigOps, str)
	}

	return cumulativeSigOps, nil
}

// checkTransactionsAndConnect is the local function used to check the
// transaction inputs for a transaction list given a predetermined TxStore.
// After ensuring the transaction is valid, the transaction is connected to the
// UTXO viewpoint.  TxTree true == Regular, false == Stake
func (b *BlockChain) checkTransactionsAndConnect(subsidyCache *SubsidyCache, inputFees ndrutil.Amount, node *blockNode, txs []*ndrutil.Tx, view *UtxoViewpoint, stxos *[]spentTxOut, txTree bool) error {
	// Perform several checks on the inputs for each transaction.  Also
	// accumulate the total fees.  This could technically be combined with
	// the loop above instead of running another loop over the
	// transactions, but by separating it we can avoid running the more
	// expensive (though still relatively cheap as compared to running the
	// scripts) checks against all the inputs when the signature operations
	// are out of bounds.
	totalFees := int64(inputFees) // Stake tx tree carry forward
	var cumulativeSigOps int
	for idx, tx := range txs {
		// Ensure that the number of signature operations is not beyond
		// the consensus limit.
		var err error
		cumulativeSigOps, err = checkNumSigOps(tx, view, idx, cumulativeSigOps)
		if err != nil {
			return err
		}

		// This step modifies the txStore and marks the tx outs used
		// spent, so be aware of this.
		txFee, err := CheckTransactionInputs(b.subsidyCache, tx,
			node.height, view, true, /* check fraud proofs */
			b.chainParams)
		if err != nil {
			log.Tracef("CheckTransactionInputs failed; error "+
				"returned: %v", err)
			return err
		}

		// Sum the total fees and ensure we don't overflow the
		// accumulator.
		lastTotalFees := totalFees
		totalFees += txFee
		if totalFees < lastTotalFees {
			return ruleError(ErrBadFees, "total fees for block "+
				"overflows accumulator")
		}

		// Connect the transaction to the UTXO viewpoint, so that in
		// flight transactions may correctly validate.
		err = view.connectTransaction(tx, node.height, uint32(idx),
			stxos)
		if err != nil {
			return err
		}
	}

	// The total output values of the coinbase transaction must not exceed
	// the expected subsidy value plus total transaction fees gained from
	// mining the block.  It is safe to ignore overflow and out of range
	// errors here because those error conditions would have already been
	// caught by checkTransactionSanity.
	// Apply penalty to fees if we're at stake validation height.

	var totalAtomOutRegular int64

	for _, txOut := range txs[0].MsgTx().TxOut {
		totalAtomOutRegular += txOut.Value
	}

	var expAtomOut int64
	if node.height == 1 {
		expAtomOut = subsidyCache.CalcBlockSubsidy(node.height)
	} else {
		subsidyWork := CalcBlockWorkSubsidy(subsidyCache,
			node.height, b.chainParams)
		subsidyTax := CalcBlockTaxSubsidy(subsidyCache,
			node.height, b.chainParams)
		expAtomOut = subsidyWork + subsidyTax + totalFees
	}

	// AmountIn for the input should be equal to the subsidy.
	coinbaseIn := txs[0].MsgTx().TxIn[0]
	subsidyWithoutFees := expAtomOut - totalFees
	if (coinbaseIn.ValueIn != subsidyWithoutFees) &&
		(node.height > 0) {
		errStr := fmt.Sprintf("bad coinbase subsidy in input;"+
			" got %v, expected %v", coinbaseIn.ValueIn,
			subsidyWithoutFees)
		return ruleError(ErrBadCoinbaseAmountIn, errStr)
	}

	if totalAtomOutRegular > expAtomOut {
		str := fmt.Sprintf("coinbase transaction for block %v"+
			" pays %v which is more than expected value "+
			"of %v", node.hash, totalAtomOutRegular,
			expAtomOut)
		return ruleError(ErrBadCoinbaseValue, str)
	}

	return nil
}

// consensusScriptVerifyFlags returns the script flags that must be used when
// executing transaction scripts to enforce the consensus rules. This includes
// any flags required as the result of any agendas that have passed and become
// active.
func (b *BlockChain) consensusScriptVerifyFlags(node *blockNode) (txscript.ScriptFlags, error) {
	scriptFlags := txscript.ScriptVerifyCleanStack |
		txscript.ScriptVerifyCheckLockTimeVerify |
		txscript.ScriptVerifyCheckSequenceVerify |
		txscript.ScriptVerifySHA256
	return scriptFlags, nil
}

// checkConnectBlock performs several checks to confirm connecting the passed
// block to the chain represented by the passed view does not violate any
// rules.  In addition, the passed view is updated to spend all of the
// referenced outputs and add all of the new utxos created by block.  Thus, the
// view will represent the state of the chain as if the block were actually
// connected and consequently the best hash for the view is also updated to
// passed block.
//
// An example of some of the checks performed are ensuring connecting the block
// would not cause any duplicate transaction hashes for old transactions that
// aren't already fully spent, double spends, exceeding the maximum allowed
// signature operations per block, invalid values in relation to the expected
// block subsidy, or fail transaction script validation.
//
// The CheckConnectBlockTemplate function makes use of this function to perform
// the bulk of its work.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) checkConnectBlock(node *blockNode, block, parent *ndrutil.Block, view *UtxoViewpoint, stxos *[]spentTxOut) error {
	// If the side chain blocks end up in the database, a call to
	// CheckBlockSanity should be done here in case a previous version
	// allowed a block that is no longer valid.  However, since the
	// implementation only currently uses memory for the side chain blocks,
	// it isn't currently necessary.

	// Ensure the view is for the node being checked.
	parentHash := &block.MsgBlock().Header.PrevBlock
	if !view.BestHash().IsEqual(parentHash) {
		return AssertError(fmt.Sprintf("inconsistent view when "+
			"checking block connection: best hash is %v instead "+
			"of expected %v", view.BestHash(), parentHash))
	}

	// Check that the coinbase pays the tax, if applicable.
	err := CoinbasePaysTax(b.subsidyCache, block.Transactions()[0], node.height, b.chainParams)
	if err != nil {
		return err
	}

	// Don't run scripts if this node is before the latest known good
	// checkpoint since the validity is verified via the checkpoints (all
	// transactions are included in the merkle root hash and any changes
	// will therefore be detected by the next checkpoint).  This is a huge
	// optimization because running the scripts is the most time consuming
	// portion of block handling.
	checkpoint := b.latestCheckpoint()
	runScripts := !b.noVerify
	if checkpoint != nil && node.height <= checkpoint.Height {
		runScripts = false
	}
	var scriptFlags txscript.ScriptFlags
	if runScripts {
		var err error
		scriptFlags, err = b.consensusScriptVerifyFlags(node)
		if err != nil {
			return err
		}
	}

	// Load all of the utxos referenced by the inputs for all transactions
	// in the block don't already exist in the utxo view from the database.
	//
	// These utxo entries are needed for verification of things such as
	// transaction inputs, counting pay-to-script-hashes, and scripts.
	err = view.fetchInputUtxos(b.db, block)
	if err != nil {
		return err
	}

	prevMedianTime := node.parent.CalcPastMedianTime()

	// Ensure the regular transaction tree does not contain any transactions
	// that 'overwrite' older transactions which are not fully spent.
	err = b.checkDupTxs(block.Transactions(), view)
	if err != nil {
		log.Tracef("checkDupTxs failed for cur TxTreeRegular: %v", err)
		return err
	}

	err = b.checkTransactionsAndConnect(b.subsidyCache, 0, node,
		block.Transactions(), view, stxos, true)
	if err != nil {
		log.Tracef("checkTransactionsAndConnect failed for cur "+
			"TxTreeRegular: %v", err)
		return err
	}

	// Skip the coinbase since it does not have any inputs and thus
	// lock times do not apply.
	for _, tx := range block.Transactions()[1:] {
		sequenceLock, err := b.calcSequenceLock(node, tx,
			view, true)
		if err != nil {
			return err
		}
		if !SequenceLockActive(sequenceLock, node.height,
			prevMedianTime) {

			str := fmt.Sprintf("block contains " +
				"transaction whose input sequence " +
				"locks are not met")
			return ruleError(ErrUnfinalizedTx, str)
		}
	}

	if runScripts {
		err = checkBlockScripts(block, view, scriptFlags, b.sigCache)
		if err != nil {
			log.Tracef("checkBlockScripts failed; error returned "+
				"on txtreeregular of cur block: %v", err)
			return err
		}
	}

	// First block has special rules concerning the ledger.
	if node.height == 1 {
		err := BlockOneCoinbasePaysTokens(block.Transactions()[0],
			b.chainParams)
		if err != nil {
			return err
		}
	}

	// Update the best hash for view to include this block since all of its
	// transactions have been connected.
	view.SetBestHash(&node.hash)

	return nil
}

// CheckConnectBlockTemplate fully validates that connecting the passed block to
// either the tip of the main chain or its parent does not violate any consensus
// rules, aside from the proof of work requirement.  The block must connect to
// the current tip of the main chain or its parent.
//
// This function is safe for concurrent access.
func (b *BlockChain) CheckConnectBlockTemplate(block *ndrutil.Block) error {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	// Skip the proof of work check as this is just a block template.
	flags := BFNoPoWCheck

	// The block template must build off the current tip of the main chain
	// or its parent.
	tip := b.bestChain.Tip()
	var prevNode *blockNode
	parentHash := block.MsgBlock().Header.PrevBlock
	if parentHash == tip.hash {
		prevNode = tip
	} else if tip.parent != nil && parentHash == tip.parent.hash {
		prevNode = tip.parent
	}
	if prevNode == nil {
		var str string
		if tip.parent != nil {
			str = fmt.Sprintf("previous block must be the current chain tip "+
				"%s or its parent %s, but got %s", tip.hash, tip.parent.hash,
				parentHash)
		} else {
			str = fmt.Sprintf("previous block must be the current chain tip "+
				"%s, but got %s", tip.hash, parentHash)
		}
		return ruleError(ErrInvalidTemplateParent, str)
	}

	// Perform context-free sanity checks on the block and its transactions.
	err := checkBlockSanity(block, b.timeSource, flags, b.chainParams)
	if err != nil {
		return err
	}

	// The block must pass all of the validation rules which depend on having
	// the headers of all ancestors available, but do not rely on having the
	// full block data of all ancestors available.
	err = b.checkBlockPositional(block, prevNode, flags)
	if err != nil {
		return err
	}

	// The block must pass all of the validation rules which depend on having
	// the full block data for all of its ancestors available.
	err = b.checkBlockContext(block, prevNode, flags)
	if err != nil {
		return err
	}

	newNode := newBlockNode(&block.MsgBlock().Header, prevNode)

	// Use the chain state as is when extending the main (best) chain.
	if prevNode.hash == tip.hash {
		// Grab the parent block since it is required throughout the block
		// connection process.
		parent, err := b.fetchMainChainBlockByNode(prevNode)
		if err != nil {
			return ruleError(ErrMissingParent, err.Error())
		}

		view := NewUtxoViewpoint()
		view.SetBestHash(&tip.hash)

		return b.checkConnectBlock(newNode, block, parent, view, nil)
	}

	// At this point, the block template must be building on the parent of the
	// current tip due to the previous checks, so undo the transactions and
	// spend information for the tip block to reach the point of view of the
	// block template.
	view := NewUtxoViewpoint()
	view.SetBestHash(&tip.hash)
	tipBlock, err := b.fetchMainChainBlockByNode(tip)
	if err != nil {
		return err
	}
	parent, err := b.fetchMainChainBlockByNode(tip.parent)
	if err != nil {
		return err
	}

	// Load all of the spent txos for the tip block from the spend journal.
	var stxos []spentTxOut
	err = b.db.View(func(dbTx database.Tx) error {
		stxos, err = dbFetchSpendJournalEntry(dbTx, tipBlock)
		return err
	})
	if err != nil {
		return err
	}

	// Update the view to unspend all of the spent txos and remove the utxos
	// created by the tip block.  Also, if the block votes against its parent,
	// reconnect all of the regular transactions.
	err = view.disconnectBlock(b.db, tipBlock, parent, stxos)
	if err != nil {
		return err
	}

	// The view is now from the point of view of the parent of the current tip
	// block.  Ensure the block template can be connected without violating any
	// rules.
	return b.checkConnectBlock(newNode, block, parent, view, nil)
}
