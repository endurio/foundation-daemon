// Copyright (c) 2014-2016 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2018-2019 The Endurio developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
)

// HashError identifies an error that indicates a hash was specified that does
// not exist.
type HashError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e HashError) Error() string {
	return fmt.Sprintf("hash %v does not exist", string(e))
}

// DeploymentError identifies an error that indicates a deployment ID was
// specified that does not exist.
type DeploymentError string

// Error returns the assertion error as a human-readable string and satisfies
// the error interface.
func (e DeploymentError) Error() string {
	return fmt.Sprintf("deployment ID %v does not exist", string(e))
}

// AssertError identifies an error that indicates an internal code consistency
// issue and should be treated as a critical and unrecoverable error.
type AssertError string

// Error returns the assertion error as a huma-readable string and satisfies
// the error interface.
func (e AssertError) Error() string {
	return "assertion failed: " + string(e)
}

// ErrorCode identifies a kind of error.
type ErrorCode int

// These constants are used to identify a specific RuleError.
const (
	// ErrDuplicateBlock indicates a block with the same hash already
	// exists.
	ErrDuplicateBlock ErrorCode = iota

	// ErrMissingParent indicates that the block was an orphan.
	ErrMissingParent

	// ErrBlockTooBig indicates the serialized block size exceeds the
	// maximum allowed size.
	ErrBlockTooBig

	// ErrWrongBlockSize indicates that the block size in the header is not
	// the actual serialized size of the block.
	ErrWrongBlockSize

	// ErrBlockVersionTooOld indicates the block version is too old and is
	// no longer accepted since the majority of the network has upgraded
	// to a newer version.
	ErrBlockVersionTooOld

	// ErrInvalidTime indicates the time in the passed block has a precision
	// that is more than one second.  The chain consensus rules require
	// timestamps to have a maximum precision of one second.
	ErrInvalidTime

	// ErrTimeTooOld indicates the time is either before the median time of
	// the last several blocks per the chain consensus rules or prior to the
	// most recent checkpoint.
	ErrTimeTooOld

	// ErrTimeTooNew indicates the time is too far in the future as compared
	// the current time.
	ErrTimeTooNew

	// ErrDifficultyTooLow indicates the difficulty for the block is lower
	// than the difficulty required by the most recent checkpoint.
	ErrDifficultyTooLow

	// ErrUnexpectedDifficulty indicates specified bits do not align with
	// the expected value either because it doesn't match the calculated
	// valued based on difficulty regarted rules or it is out of the valid
	// range.
	ErrUnexpectedDifficulty

	// ErrHighHash indicates the block does not hash to a value which is
	// lower than the required target difficultly.
	ErrHighHash

	// ErrBadMerkleRoot indicates the calculated merkle root does not match
	// the expected value.
	ErrBadMerkleRoot

	// ErrBadCheckpoint indicates a block that is expected to be at a
	// checkpoint height does not match the expected one.
	ErrBadCheckpoint

	// ErrForkTooOld indicates a block is attempting to fork the block chain
	// before the most recent checkpoint.
	ErrForkTooOld

	// ErrCheckpointTimeTooOld indicates a block has a timestamp before the
	// most recent checkpoint.
	ErrCheckpointTimeTooOld

	// ErrNoTransactions indicates the block does not have a least one
	// transaction.  A valid block must have at least the coinbase
	// transaction.
	ErrNoTransactions

	// ErrTooManyTransactions indicates the block has more transactions than
	// are allowed.
	ErrTooManyTransactions

	// ErrNoTxInputs indicates a transaction does not have any inputs.  A
	// valid transaction must have at least one input.
	ErrNoTxInputs

	// ErrNoTxOutputs indicates a transaction does not have any outputs.  A
	// valid transaction must have at least one output.
	ErrNoTxOutputs

	// ErrTxTooBig indicates a transaction exceeds the maximum allowed size
	// when serialized.
	ErrTxTooBig

	// ErrBadTxOutValue indicates an output value for a transaction is
	// invalid in some way such as being out of range.
	ErrBadTxOutValue

	// ErrDuplicateTxInputs indicates a transaction references the same
	// input more than once.
	ErrDuplicateTxInputs

	// ErrBadTxInput indicates a transaction input is invalid in some way
	// such as referencing a previous transaction outpoint which is out of
	// range or not referencing one at all.
	ErrBadTxInput

	// ErrMissingTxOut indicates a transaction output referenced by an input
	// either does not exist or has already been spent.
	ErrMissingTxOut

	// ErrUnfinalizedTx indicates a transaction has not been finalized.
	// A valid block may only contain finalized transactions.
	ErrUnfinalizedTx

	// ErrDuplicateTx indicates a block contains an identical transaction
	// (or at least two transactions which hash to the same value).  A
	// valid block may only contain unique transactions.
	ErrDuplicateTx

	// ErrOverwriteTx indicates a block contains a transaction that has
	// the same hash as a previous transaction which has not been fully
	// spent.
	ErrOverwriteTx

	// ErrImmatureSpend indicates a transaction is attempting to spend a
	// coinbase that has not yet reached the required maturity.
	ErrImmatureSpend

	// ErrSpendTooHigh indicates a transaction is attempting to spend more
	// value than the sum of all of its inputs.
	ErrSpendTooHigh

	// ErrBadFees indicates the total fees for a block are invalid due to
	// exceeding the maximum possible value.
	ErrBadFees

	// ErrTooManySigOps indicates the total number of signature operations
	// for a transaction or block exceed the maximum allowed limits.
	ErrTooManySigOps

	// ErrFirstTxNotCoinbase indicates the first transaction in a block
	// is not a coinbase transaction.
	ErrFirstTxNotCoinbase

	// ErrCoinbaseHeight indicates that the encoded height in the coinbase
	// is incorrect.
	ErrCoinbaseHeight

	// ErrMultipleCoinbases indicates a block contains more than one
	// coinbase transaction.
	ErrMultipleCoinbases

	// ErrBadCoinbaseScriptLen indicates the length of the signature script
	// for a coinbase transaction is not within the valid range.
	ErrBadCoinbaseScriptLen

	// ErrBadCoinbaseValue indicates the amount of a coinbase value does
	// not match the expected value of the subsidy plus the sum of all fees.
	ErrBadCoinbaseValue

	// ErrBadCoinbaseOutpoint indicates that the outpoint used by a coinbase
	// as input was non-null.
	ErrBadCoinbaseOutpoint

	// ErrBadCoinbaseFraudProof indicates that the fraud proof for a coinbase
	// input was non-null.
	ErrBadCoinbaseFraudProof

	// ErrBadCoinbaseAmountIn indicates that the AmountIn (=subsidy) for a
	// coinbase input was incorrect.
	ErrBadCoinbaseAmountIn

	// ErrScriptMalformed indicates a transaction script is malformed in
	// some way.  For example, it might be longer than the maximum allowed
	// length or fail to parse.
	ErrScriptMalformed

	// ErrScriptValidation indicates the result of executing transaction
	// script failed.  The error covers any failure when executing scripts
	// such signature verification failures and execution past the end of
	// the stack.
	ErrScriptValidation

	// ErrForceReorgWrongChain indicates that a reroganization was attempted
	// to be forced, but the chain indicated was not mirrored by b.bestChain.
	ErrForceReorgWrongChain

	// ErrForceReorgMissingChild indicates that a reroganization was attempted
	// to be forced, but the child node to reorganize to could not be found.
	ErrForceReorgMissingChild

	// ErrBadBlockHeight indicates that a block header's embedded block height
	// was different from where it was actually embedded in the block chain.
	ErrBadBlockHeight

	// ErrBlockOneTx indicates that block height 1 failed to correct generate
	// the block one premine transaction.
	ErrBlockOneTx

	// ErrBlockOneTx indicates that block height 1 coinbase transaction in
	// zero was incorrect in some way.
	ErrBlockOneInputs

	// ErrBlockOneOutputs indicates that block height 1 failed to incorporate
	// the ledger addresses correctly into the transaction's outputs.
	ErrBlockOneOutputs

	// ErrNoTax indicates that there was no tax present in the coinbase of a
	// block after height 1.
	ErrNoTax

	// ErrExpiredTx indicates that the transaction is currently expired.
	ErrExpiredTx

	// ErrExpiryTxSpentEarly indicates that an output from a transaction
	// that included an expiry field was spent before coinbase maturity
	// many blocks had passed in the blockchain.
	ErrExpiryTxSpentEarly

	// ErrFraudAmountIn indicates the witness amount given was fraudulent.
	ErrFraudAmountIn

	// ErrFraudBlockHeight indicates the witness block height given was
	// fraudulent.
	ErrFraudBlockHeight

	// ErrFraudBlockIndex indicates the witness block index given was
	// fraudulent.
	ErrFraudBlockIndex

	// ErrZeroValueOutputSpend indicates that a transaction attempted to spend a
	// zero value output.
	ErrZeroValueOutputSpend

	// ErrKnownInvalidBlock indicates that this block has previously failed
	// validation.
	ErrKnownInvalidBlock

	// ErrInvalidAncestorBlock indicates that an ancestor of this block has
	// failed validation.
	ErrInvalidAncestorBlock

	// ErrInvalidTemplateParent indicates that a block template builds on a
	// block that is either not the current best chain tip or its parent.
	ErrInvalidTemplateParent

	// numErrorCodes is the maximum error code number used in tests.
	numErrorCodes
)

// Map of ErrorCode values back to their constant names for pretty printing.
var errorCodeStrings = map[ErrorCode]string{
	ErrDuplicateBlock:         "ErrDuplicateBlock",
	ErrMissingParent:          "ErrMissingParent",
	ErrBlockTooBig:            "ErrBlockTooBig",
	ErrWrongBlockSize:         "ErrWrongBlockSize",
	ErrBlockVersionTooOld:     "ErrBlockVersionTooOld",
	ErrInvalidTime:            "ErrInvalidTime",
	ErrTimeTooOld:             "ErrTimeTooOld",
	ErrTimeTooNew:             "ErrTimeTooNew",
	ErrDifficultyTooLow:       "ErrDifficultyTooLow",
	ErrUnexpectedDifficulty:   "ErrUnexpectedDifficulty",
	ErrHighHash:               "ErrHighHash",
	ErrBadMerkleRoot:          "ErrBadMerkleRoot",
	ErrBadCheckpoint:          "ErrBadCheckpoint",
	ErrForkTooOld:             "ErrForkTooOld",
	ErrCheckpointTimeTooOld:   "ErrCheckpointTimeTooOld",
	ErrNoTransactions:         "ErrNoTransactions",
	ErrTooManyTransactions:    "ErrTooManyTransactions",
	ErrNoTxInputs:             "ErrNoTxInputs",
	ErrNoTxOutputs:            "ErrNoTxOutputs",
	ErrTxTooBig:               "ErrTxTooBig",
	ErrBadTxOutValue:          "ErrBadTxOutValue",
	ErrDuplicateTxInputs:      "ErrDuplicateTxInputs",
	ErrBadTxInput:             "ErrBadTxInput",
	ErrMissingTxOut:           "ErrMissingTxOut",
	ErrUnfinalizedTx:          "ErrUnfinalizedTx",
	ErrDuplicateTx:            "ErrDuplicateTx",
	ErrOverwriteTx:            "ErrOverwriteTx",
	ErrImmatureSpend:          "ErrImmatureSpend",
	ErrSpendTooHigh:           "ErrSpendTooHigh",
	ErrBadFees:                "ErrBadFees",
	ErrTooManySigOps:          "ErrTooManySigOps",
	ErrFirstTxNotCoinbase:     "ErrFirstTxNotCoinbase",
	ErrCoinbaseHeight:         "ErrCoinbaseHeight",
	ErrMultipleCoinbases:      "ErrMultipleCoinbases",
	ErrBadCoinbaseScriptLen:   "ErrBadCoinbaseScriptLen",
	ErrBadCoinbaseValue:       "ErrBadCoinbaseValue",
	ErrBadCoinbaseOutpoint:    "ErrBadCoinbaseOutpoint",
	ErrBadCoinbaseFraudProof:  "ErrBadCoinbaseFraudProof",
	ErrBadCoinbaseAmountIn:    "ErrBadCoinbaseAmountIn",
	ErrScriptMalformed:        "ErrScriptMalformed",
	ErrScriptValidation:       "ErrScriptValidation",
	ErrForceReorgWrongChain:   "ErrForceReorgWrongChain",
	ErrForceReorgMissingChild: "ErrForceReorgMissingChild",
	ErrBadBlockHeight:         "ErrBadBlockHeight",
	ErrBlockOneTx:             "ErrBlockOneTx",
	ErrBlockOneInputs:         "ErrBlockOneInputs",
	ErrBlockOneOutputs:        "ErrBlockOneOutputs",
	ErrNoTax:                  "ErrNoTax",
	ErrExpiredTx:              "ErrExpiredTx",
	ErrExpiryTxSpentEarly:     "ErrExpiryTxSpentEarly",
	ErrFraudAmountIn:          "ErrFraudAmountIn",
	ErrFraudBlockHeight:       "ErrFraudBlockHeight",
	ErrFraudBlockIndex:        "ErrFraudBlockIndex",
	ErrZeroValueOutputSpend:   "ErrZeroValueOutputSpend",
	ErrKnownInvalidBlock:      "ErrKnownInvalidBlock",
	ErrInvalidAncestorBlock:   "ErrInvalidAncestorBlock",
	ErrInvalidTemplateParent:  "ErrInvalidTemplateParent",
}

// String returns the ErrorCode as a human-readable name.
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

// RuleError identifies a rule violation.  It is used to indicate that
// processing of a block or transaction failed due to one of the many validation
// rules.  The caller can use type assertions to determine if a failure was
// specifically due to a rule violation and access the ErrorCode field to
// ascertain the specific reason for the rule violation.
type RuleError struct {
	ErrorCode   ErrorCode // Describes the kind of error
	Description string    // Human readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e RuleError) Error() string {
	return e.Description
}

// ruleError creates an RuleError given a set of arguments.
func ruleError(c ErrorCode, desc string) RuleError {
	return RuleError{ErrorCode: c, Description: desc}
}
