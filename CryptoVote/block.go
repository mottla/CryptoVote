// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"crypto/sha256"
	"fmt"

	"io"
	"encoding/binary"
	"crypto/rand"
	"bytes"
	"time"
)

var genesisBlock = &Block{
	Index:        0,
	PreviousHash: [32]byte{},
	Timestamp:    1525996819,
	//Data:         []byte("my genesis block!!"),
	Difficulty: 5, //trailing zero bits for a valid blockhash
	Hash:       [32]byte{1},
}

type Blocks []*Block

func (blocks Blocks) Len() int {
	return len(blocks)
}

func (blocks Blocks) Swap(i, j int) {
	blocks[i], blocks[j] = blocks[j], blocks[i]
}

func (blocks Blocks) Less(i, j int) bool {
	return blocks[i].Index < blocks[j].Index
}

type Block struct {
	Index        uint32      `json:"index"`
	PreviousHash [32]byte    `json:"previousHash"`
	Timestamp    int64       `json:"timestamp"`
	Difficulty   uint8       `json:"difficulty"`
	Nonce        uint32      `json:"nonce"`
	ExtraNonce   uint64      `json: "extrnonce"`
	Data         transaction `json:"Data"`
	Hash         [32]byte    `json:"hash"`
}

func (child *Block) isValidAncestor(parent *Block) error {
	if parent == nil || child == nil {
		return errorCall("is nil at ")
	}
	if bytes.Compare(parent.Hash[:], emptyHash[:]) == 0 || bytes.Compare(child.Hash[:], emptyHash[:]) == 0 {
		return errorCall("hash is nil")
	}
	if parent.Timestamp > child.Timestamp {
		return errorCall("parent older then child")
	}

	if parent.Index+1 != child.Index {
		return errorCall("index not ascending")
	}
	//TODO difficulty adaptation
	if parent.Difficulty != child.Difficulty {
		return errorCall("difficulty change invalid")
	}

	if bytes.Compare(parent.Hash[:], child.PreviousHash[:]) != 0 {
		return errorCall("child is not referencing parent")
	}

	if !child.isIntrinsicallyValid() {
		return errorCall("child is intrinsically invalid")
	}

	return nil
}

func (block *Block) hash() [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf(
		"%d%d%d%s%d%d%s",
		block.Nonce, block.ExtraNonce, block.Index, block.PreviousHash, block.Timestamp, block.Difficulty, block.Data,
	)))
}

// MaxBlockHeaderPayload is the maximum number of bytes a block header can be.
// Version 4 bytes + Timestamp 4 bytes + Bits 4 bytes + Nonce 4 bytes +
// PrevBlock and MerkleRoot hashes.
const MaxBlockHeaderPayload = 16 + (64 * 2)

//// BlockHash computes the block identifier hash for the given block header.
//func (block *Block) BlockHash() string {
//	// Encode the header and double sha256 everything prior to the number of
//	// transactions.  Ignore the error returns since there is no way the
//	// encode could fail except being out of memory which would cause a
//	// run-time panic.
//	buf := bytes.NewBuffer(make([]byte, 0, MaxBlockHeaderPayload))
//	_ = writeBlockHeader(buf, 0, h)
//
//	return chainhash.DoubleHashH(buf.Bytes())
//}

func (block *Block) updateExtraNonce() {
	b := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		panic("rand failed")
	}
	block.ExtraNonce = binary.BigEndian.Uint64(b)
}

func (block *Block) isIntrinsicallyValid() bool {
	hash := block.hash()
	if !block.isValidHash() {
		return false
	}
	//todo maybe add some buffer here (1-30seconds ??).. duno how to determine properly though
	if block.Timestamp > time.Now().Unix() {
		return false
	}

	targetDifficulty := make([]byte, 32)
	targetDifficulty[block.Difficulty/8] = (byte(128) >> (block.Difficulty % 8))

	if bytes.Compare(hash[:], targetDifficulty) == +1 {
		return false
	}
	return true
}

func (block *Block) isValidHash() bool {
	return block.Hash == block.hash()
}
