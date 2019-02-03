// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ndrutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/endurio/ndrd/chaincfg/chainhash"
	"github.com/endurio/ndrd/wire"
)

// OutOfRangeError describes an error due to accessing an element that is out
// of range.
type OutOfRangeError string

// assertBlockImmutability throws a panic when a block has been
// mutated.
var assertBlockImmutability = false

// Error satisfies the error interface and prints human-readable errors.
func (e OutOfRangeError) Error() string {
	return string(e)
}

// Block defines a cryptocurrency block that provides easier and more efficient
// manipulation of raw blocks.  It also memoizes hashes for the block and its
// transactions on their first access so subsequent accesses don't have to
// repeat the relatively expensive hashing operations.
type Block struct {
	msgBlock        *wire.MsgBlock // Underlying MsgBlock
	serializedBlock []byte         // Serialized bytes for the block
	hash            chainhash.Hash // Cached block hash
	transactions    []*Tx          // Transactions
	txnsGenerated   bool           // ALL wrapped transactions generated
}

// MsgBlock returns the underlying wire.MsgBlock for the Block.
func (b *Block) MsgBlock() *wire.MsgBlock {
	// Return the cached block.
	return b.msgBlock
}

// Bytes returns the serialized bytes for the Block.  This is equivalent to
// calling Serialize on the underlying wire.MsgBlock, however it caches the
// result so subsequent calls are more efficient.
func (b *Block) Bytes() ([]byte, error) {
	// Return the cached serialized bytes if it has already been generated.
	if len(b.serializedBlock) != 0 {
		return b.serializedBlock, nil
	}

	// Serialize the MsgBlock.
	var w bytes.Buffer
	w.Grow(b.msgBlock.SerializeSize())
	err := b.msgBlock.Serialize(&w)
	if err != nil {
		return nil, err
	}
	serializedBlock := w.Bytes()

	// Cache the serialized bytes and return them.
	b.serializedBlock = serializedBlock
	return serializedBlock, nil
}

// BlockHeaderBytes returns the serialized bytes for the Block's header.  This is
// equivalent to calling Serialize on the underlying wire.MsgBlock, but it
// returns a byte slice.
func (b *Block) BlockHeaderBytes() ([]byte, error) {
	// Return the cached serialized bytes if it has already been generated.
	if len(b.serializedBlock) != 0 {
		return b.serializedBlock, nil
	}

	// Serialize the MsgBlock.
	var w bytes.Buffer
	w.Grow(wire.MaxBlockHeaderPayload)
	err := b.msgBlock.Header.Serialize(&w)
	if err != nil {
		return nil, err
	}
	serializedBlockHeader := w.Bytes()

	// Cache the serialized bytes and return them.
	return serializedBlockHeader, nil
}

// Hash returns the block identifier hash for the Block.  This is equivalent to
// calling BlockHash on the underlying wire.MsgBlock, however it caches the
// result so subsequent calls are more efficient.
func (b *Block) Hash() *chainhash.Hash {
	if assertBlockImmutability {
		hash := b.msgBlock.BlockHash()
		if !hash.IsEqual(&b.hash) {
			str := fmt.Sprintf("ASSERT: mutated util.block detected, old hash "+
				"%v, new hash %v", b.hash, hash)
			panic(str)
		}
	}

	return &b.hash
}

// Tx returns a wrapped transaction (ndrutil.Tx) for the transaction at the
// specified index in the Block.  The supplied index is 0 based.  That is to
// say, the first transaction in the block is txNum 0.  This is nearly
// equivalent to accessing the raw transaction (wire.MsgTx) from the
// underlying wire.MsgBlock, however the wrapped transaction has some helpful
// properties such as caching the hash so subsequent calls are more efficient.
func (b *Block) Tx(txNum int) (*Tx, error) {
	// Ensure the requested transaction is in range.
	numTx := uint64(len(b.msgBlock.Transactions))
	if txNum < 0 || uint64(txNum) > numTx {
		str := fmt.Sprintf("transaction index %d is out of range - max %d",
			txNum, numTx-1)
		return nil, OutOfRangeError(str)
	}

	// Generate slice to hold all of the wrapped transactions if needed.
	if len(b.transactions) == 0 {
		b.transactions = make([]*Tx, numTx)
	}

	// Return the wrapped transaction if it has already been generated.
	if b.transactions[txNum] != nil {
		return b.transactions[txNum], nil
	}

	// Generate and cache the wrapped transaction and return it.
	newTx := NewTx(b.msgBlock.Transactions[txNum])
	newTx.SetIndex(txNum)
	b.transactions[txNum] = newTx
	return newTx, nil
}

// Transactions returns a slice of wrapped transactions (ndrutil.Tx) for all
// transactions in the Block.  This is nearly equivalent to accessing the raw
// transactions (wire.MsgTx) in the underlying wire.MsgBlock, however it
// instead provides easy access to wrapped versions (ndrutil.Tx) of them.
func (b *Block) Transactions() []*Tx {
	// Return transactions if they have ALL already been generated.  This
	// flag is necessary because the wrapped transactions are lazily
	// generated in a sparse fashion.
	if b.txnsGenerated {
		return b.transactions
	}

	// Generate slice to hold all of the wrapped transactions if needed.
	if len(b.transactions) == 0 {
		b.transactions = make([]*Tx, len(b.msgBlock.Transactions))
	}

	// Generate and cache the wrapped transactions for all that haven't
	// already been done.
	for i, tx := range b.transactions {
		if tx == nil {
			newTx := NewTx(b.msgBlock.Transactions[i])
			newTx.SetIndex(i)
			b.transactions[i] = newTx
		}
	}

	b.txnsGenerated = true
	return b.transactions
}

// TxHash returns the hash for the requested transaction number in the Block.
// The supplied index is 0 based.  That is to say, the first transaction in the
// block is txNum 0.  This is equivalent to calling TxHash on the underlying
// wire.MsgTx, however it caches the result so subsequent calls are more
// efficient.
func (b *Block) TxHash(txNum int) (*chainhash.Hash, error) {
	// Attempt to get a wrapped transaction for the specified index.  It
	// will be created lazily if needed or simply return the cached version
	// if it has already been generated.
	tx, err := b.Tx(txNum)
	if err != nil {
		return nil, err
	}

	// Defer to the wrapped transaction which will return the cached hash if
	// it has already been generated.
	return tx.Hash(), nil
}

// TxLoc returns the offsets and lengths of each transaction in a raw block.
// It is used to allow fast indexing into transactions within the raw byte
// stream.
func (b *Block) TxLoc() ([]wire.TxLoc, error) {
	rawMsg, err := b.Bytes()
	if err != nil {
		return nil, err
	}
	rbuf := bytes.NewBuffer(rawMsg)

	var mblock wire.MsgBlock
	txLocs, err := mblock.DeserializeTxLoc(rbuf)
	if err != nil {
		return nil, err
	}
	return txLocs, err
}

// Height returns a casted int64 height from the block header.
//
// This function should not be used for new code and will be
// removed in the future.
func (b *Block) Height() int64 {
	return int64(b.msgBlock.Header.Height)
}

// NewBlock returns a new instance of a block given an underlying
// wire.MsgBlock.  See Block.
func NewBlock(msgBlock *wire.MsgBlock) *Block {
	return &Block{
		hash:     msgBlock.BlockHash(),
		msgBlock: msgBlock,
	}
}

// NewBlockDeepCopyCoinbase returns a new instance of a block given an underlying
// wire.MsgBlock, but makes a deep copy of the coinbase transaction since it's
// sometimes mutable.
func NewBlockDeepCopyCoinbase(msgBlock *wire.MsgBlock) *Block {
	// Copy the msgBlock and the pointers to all the transactions.
	msgBlockCopy := new(wire.MsgBlock)

	lenTxs := len(msgBlock.Transactions)
	mtxsCopy := make([]*wire.MsgTx, lenTxs)
	copy(mtxsCopy, msgBlock.Transactions)

	msgBlockCopy.Transactions = mtxsCopy
	msgBlockCopy.Header = msgBlock.Header

	// Deep copy the first transaction. Also change the coinbase pointer.
	msgBlockCopy.Transactions[0] =
		NewTxDeep(msgBlockCopy.Transactions[0]).MsgTx()

	bl := &Block{
		msgBlock: msgBlockCopy,
	}
	bl.hash = msgBlock.BlockHash()

	return bl
}

// NewBlockDeepCopy deep copies an entire block down to the wire components and
// returns the new block based off of this copy.
func NewBlockDeepCopy(msgBlock *wire.MsgBlock) *Block {
	// Deep copy the header and all the transactions.
	msgBlockCopy := new(wire.MsgBlock)
	lenTxs := len(msgBlock.Transactions)
	mtxsCopy := make([]*wire.MsgTx, lenTxs)
	for i, mtx := range msgBlock.Transactions {
		txd := NewTxDeep(mtx)
		mtxsCopy[i] = txd.MsgTx()
	}
	msgBlockCopy.Transactions = mtxsCopy
	msgBlockCopy.Header = msgBlock.Header

	bl := &Block{
		msgBlock: msgBlockCopy,
	}
	bl.hash = msgBlock.BlockHash()

	return bl
}

// NewBlockFromBytes returns a new instance of a block given the
// serialized bytes.  See Block.
func NewBlockFromBytes(serializedBlock []byte) (*Block, error) {
	br := bytes.NewReader(serializedBlock)
	b, err := NewBlockFromReader(br)
	if err != nil {
		return nil, err
	}
	b.serializedBlock = serializedBlock
	return b, nil
}

// NewBlockFromReader returns a new instance of a block given a
// Reader to deserialize the block.  See Block.
func NewBlockFromReader(r io.Reader) (*Block, error) {
	// Deserialize the bytes into a MsgBlock.
	var msgBlock wire.MsgBlock
	err := msgBlock.Deserialize(r)
	if err != nil {
		return nil, err
	}

	b := Block{
		hash:     msgBlock.BlockHash(),
		msgBlock: &msgBlock,
	}
	return &b, nil
}

// NewBlockFromBlockAndBytes returns a new instance of a block given
// an underlying wire.MsgBlock and the serialized bytes for it.  See Block.
func NewBlockFromBlockAndBytes(msgBlock *wire.MsgBlock, serializedBlock []byte) *Block {
	return &Block{
		hash:            msgBlock.BlockHash(),
		msgBlock:        msgBlock,
		serializedBlock: serializedBlock,
	}
}
