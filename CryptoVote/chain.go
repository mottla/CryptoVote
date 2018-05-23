// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"sync"
	"log"
	"os"
	"fmt"
	"bytes"
)

type Blockchain struct {
	blocks      Blocks
	mu          sync.RWMutex
	accessMap   map[BlockHash]*Block           //key: blockhash
	lookupMap   map[BlockHash]bool             //key : blockhash
	votingMap   map[TransactionID]*voting      //key: ed sig of vontings ->voting contract transaction
	voteSetMap  map[TransactionID]*transaction //key: ed sig. value voteSet transaction only
	txLookupMap map[TransactionID]*Block       //key: edsignature [32:]bytes
	logger      *log.Logger
}

func (bc *Blockchain) log(v ...interface{}) {
	bc.logger.Println(v)
}

func (bc *Blockchain) logError(err error) {
	bc.log("[ERROR]", err)
}

func newBlockchain() *Blockchain {

	return &Blockchain{
		blocks:      Blocks{},
		mu:          sync.RWMutex{},
		accessMap:   map[BlockHash]*Block{},
		lookupMap:   map[BlockHash]bool{},
		txLookupMap: make(map[TransactionID]*Block),
		votingMap:   make(map[TransactionID]*voting),
		voteSetMap:  make(map[TransactionID]*transaction),
		logger:      log.New(os.Stdout, fmt.Sprintf("[%v] Blockchain: ", apiAddr), log.Ldate|log.Ltime, ),
	}
}
func newBlockchainGenesis() (re *Blockchain) {

	return &Blockchain{
		blocks:      Blocks{genesisBlock},
		mu:          sync.RWMutex{},
		accessMap:   map[BlockHash]*Block{genesisBlock.Hash: genesisBlock},
		lookupMap:   map[BlockHash]bool{genesisBlock.Hash: true},
		txLookupMap: make(map[TransactionID]*Block),
		votingMap:   make(map[TransactionID]*voting),
		voteSetMap:  make(map[TransactionID]*transaction),
	}
}

//returns the latest stales blocks index
func (bc *Blockchain) chainHeight() uint32 {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return uint32(len(bc.blocks) - 1)
}

func (bc *Blockchain) getGenesisBlock() *Block {
	return bc.Block_ind(0)
}

func (bc *Blockchain) getPartialChain(startPoint uint32) (bl Blocks, err error) {

	if startPoint > bc.chainHeight() {
		return nil, errorCall("requested chain startpoint is higher then latest stale block ");
	}
	return bc.blocks[startPoint:], nil
}

func (bc *Blockchain) getLatestBlock() *Block {
	//is lock necessary if we return in one line?
	bc.mu.Lock()
	defer bc.mu.Unlock()
	return bc.blocks[len(bc.blocks)-1]
}

func (bc *Blockchain) Block_ind(index uint32) (b *Block) {

	bc.mu.Lock()
	defer bc.mu.Unlock()
	if index >= 0 && int(index) < len(bc.blocks) {
		return bc.blocks[index]
	}
	bc.log("##################### index ", index)
	return nil
}

func (bc *Blockchain) hasBlock(hash BlockHash) bool {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	//bc.log("looking for %v in accesmap", hash)
	return bc.lookupMap[hash]
}
func (bc *Blockchain) hasTransaction(tx *transaction) (ok bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	//bc.log("looking for %v in accesmap", hash)
	_, ok = bc.txLookupMap[copyBytesTo32(tx.EdSignature)]
	return
}

//returns the block, holding a transaction.
func (bc *Blockchain) isRevealed(id TransactionID) (ok bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	//bc.log("looking for %v in accesmap", hash)
	if res, ok := bc.txLookupMap[id]; ok {
		return res.Data.Typ == REVEAL_VOTING
	}
	return false
}

func (bc *Blockchain) Block_hash(hash BlockHash) *Block {

	bc.mu.Lock()
	defer bc.mu.Unlock()
	//bc.log("looking for %v in accesmap", hash)
	return bc.accessMap[hash]
}

func (bc *Blockchain) BlockHoldingTx(edSig TransactionID) (res *Block, ok bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	res, ok = bc.txLookupMap[edSig]
	return
}

func (bc *Blockchain) Voting(edSig TransactionID) (res *voting, ok bool) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	res, ok = bc.votingMap[edSig]
	return
}

//this function should only be called by temp chains, needed for validation
//it adds a block, without any validity checks
func (bc *Blockchain) addBlockCareless(block *Block) {

	bc.mu.Lock()
	defer bc.mu.Unlock()

	bc.blocks = append(bc.blocks, block)
	//fmt.Printf("adding to acces map key %v", block.Hash)
	bc.accessMap[block.Hash] = block
	bc.lookupMap[block.Hash] = true

	trans := block.Data
	switch trans.Typ {
	case ADD_VOTERS:
		bc.voteSetMap[copyBytesTo32(trans.EdSignature)] = &trans
	case CREATE_VOTING:
		voting := newVoting(false)
		voting.add(&block.Data)
		for _, id := range trans.VoteSet {
			voting.add(bc.voteSetMap[id])
		}
		bc.votingMap[copyBytesTo32(trans.EdSignature)] = voting
	case VOTE:
		voting, ok := bc.votingMap[copyBytesTo32(trans.EdSignature)]
		if !ok {
			voting = newVoting(false)
			bc.votingMap[copyBytesTo32(trans.EdSignature)] = voting
		}
		voting.add(&trans)
	case REVEAL_VOTING:
		voting, ok := bc.votingMap[copyBytesTo32(trans.EdSignature)]
		if !ok {
			voting = newVoting(false)
			bc.votingMap[copyBytesTo32(trans.EdSignature)] = voting
		}
		voting.add(&trans)
	default:
		return
	}
	//fmt.Println("ADDING ", copyBytesTo32(block.Data.EdSignature))
	bc.txLookupMap[ copyBytesTo32(block.Data.EdSignature) ] = block
}

//adds a block to the chain.
//updates the votingMap if the block contained voting transactions
//asserts that block has been fully validated.
func (bc *Blockchain) addBlock(block *Block, pool *TxPool) bool {

	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.lookupMap[block.Hash] {
		bc.logError(errorCall("blockhash already in chain "))
		panic("shuld not happen1")
		return false
	}

	if block.Index-1 < 0 || int(block.Index-1) >= len(bc.blocks) {
		bc.logError(errorCall(fmt.Sprintf("cannot add to parent with index ", block.Index-1)))
		panic("shuld not happen2")
		return false
	}

	if err := block.isValidAncestor(bc.blocks[block.Index-1]); err != nil {
		panic("shuld not happen3")
		bc.logError(err)
		return false
	}

	//if the new block induces a fork, all blocks and and all votings listet on those blocks starting from the block.Index will be removed
	if len(bc.blocks) > int(block.Index) {
		for i, _ := range bc.blocks[block.Index:] {
			//all transactions that were included before the fork go back into the pool

			delete(bc.accessMap, bc.blocks[i].Hash)
			delete(bc.lookupMap, bc.blocks[i].Hash)

			if !bc.blocks[i].Data.isEmpty() {
				delete(bc.txLookupMap, copyBytesTo32(bc.blocks[i].Data.EdSignature))

				switch block.Data.Typ {
				case CREATE_VOTING:
					delete(bc.votingMap, copyBytesTo32(bc.blocks[i].Data.EdSignature))
				case ADD_VOTERS:
					delete(bc.voteSetMap, copyBytesTo32(bc.blocks[i].Data.EdSignature))
				default:

				}
				pool.addTransaction(&bc.blocks[i].Data)
			}
		}
		bc.blocks = bc.blocks[:block.Index]
	}

	if len(bc.blocks) == int(block.Index) {
		pool.remove(&block.Data)
		bc.blocks = append(bc.blocks, block)
		//fmt.Printf("adding to acces map key %v", block.Hash)
		bc.accessMap[block.Hash] = block
		bc.lookupMap[block.Hash] = true

		trans := block.Data
		switch trans.Typ {
		case ADD_VOTERS:
			bc.voteSetMap[copyBytesTo32(trans.EdSignature)] = &trans
		case CREATE_VOTING:
			voting := newVoting(true)
			voting.add(&trans)
			for _, id := range trans.VoteSet {
				voting.add(bc.voteSetMap[id])
			}
			bc.votingMap[copyBytesTo32(trans.EdSignature)] = voting
		case VOTE:
			voting := bc.votingMap[trans.VoteID]
			voting.add(&trans)
		case REVEAL_VOTING:
			voting := bc.votingMap[trans.VoteID]
			voting.add(&trans)
		default:
			return true
		}
		//fmt.Println("ADDING ", copyBytesTo32(block.Data.EdSignature))
		bc.txLookupMap[ copyBytesTo32(trans.EdSignature) ] = block
		return true
	}

	return false
}

func (bc *Blockchain) validDifficultyChange(indexParent uint32, difficultyOfChild uint8) bool {
	//TODO difficulty adaptation
	return bc.Block_ind(indexParent).Difficulty == difficultyOfChild
}

func (bc *Blockchain) String() string {
	var buffer = new(bytes.Buffer)
	for _, b := range bc.blocks {
		buffer.WriteString(fmt.Sprintf("%v :%x ->", b.Index, b.Hash[:10]))
	}
	return buffer.String()
}
