// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"sync"
	"github.com/naivechain-master/CryptoNote1/edwards"
	"log"
	"fmt"
	"bytes"
	"os"
)

type Blockchain struct {
	blocks      Blocks
	mu          sync.RWMutex
	accessMap   map[[32]byte]*Block  //key: blockhash
	lookupMap   map[[32]byte]bool    //key : blockhash
	votingMap   map[[32]byte]*Voting //key: blockhash
	txLookupMap map[[32]byte]*Block  //key: edsignature [0:32]bytes
	logger      *log.Logger
}

func (bc *Blockchain) log(v ...interface{}) {
	bc.logger.Println(v)
}

func (bc *Blockchain) logError(err error) {
	bc.log("[ERROR]", err)
}

type Voting struct {
	contract *transaction
	votes    map[[32]byte]*transaction //key is the keyimage to avoid multiple voting with the same key
	reveal   *transaction
}

func newBlockchain() *Blockchain {

	return &Blockchain{
		blocks:      Blocks{},
		mu:          sync.RWMutex{},
		accessMap:   map[[32]byte]*Block{},
		lookupMap:   map[[32]byte]bool{},
		txLookupMap: make(map[[32]byte]*Block),
		//key is the corresponding blockhash
		votingMap: make(map[[32]byte]*Voting), //key is the blockhash holding the votingcontract
		logger:    log.New(os.Stdout, fmt.Sprintf("[%v] Blockchain: ", apiAddr), log.Ldate|log.Ltime, ),
	}
}
func newBlockchainGenesis() (re *Blockchain) {

	return &Blockchain{
		blocks:      Blocks{genesisBlock},
		mu:          sync.RWMutex{},
		accessMap:   map[[32]byte]*Block{genesisBlock.Hash: genesisBlock},
		lookupMap:   map[[32]byte]bool{genesisBlock.Hash: true},
		txLookupMap: make(map[[32]byte]*Block),
		//key is the corresponding blockhash
		votingMap: make(map[[32]byte]*Voting), //key is the blockhash holding the votingcontract
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

func (bc *Blockchain) hasBlock(hash [32]byte) bool {
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

func (bc *Blockchain) Block_hash(hash [32]byte) *Block {

	bc.mu.Lock()
	defer bc.mu.Unlock()
	//bc.log("looking for %v in accesmap", hash)
	return bc.accessMap[hash]
}

//TODO
func (bc *Blockchain) BlockHoldingTx(edSig [32]byte) (res *Block, ok bool) {

	bc.mu.Lock()
	defer bc.mu.Unlock()
	res, ok = bc.txLookupMap[edSig]
	return
}

//this function should only be called by temp chains, needed for validation
//it adds a block, without any validity checks
func (bc *Blockchain) addBlockCareless(block *Block) {

	bc.blocks = append(bc.blocks, block)
	//fmt.Printf("adding to acces map key %v", block.Hash)
	bc.accessMap[block.Hash] = block
	bc.lookupMap[block.Hash] = true
	switch block.Data.Typ {
	case ADD_VOTERS:
		//do nothing
	case CREATE_VOTING:
		bc.votingMap[block.Hash] = &Voting{contract: &block.Data, votes: make(map[[32]byte]*transaction)}
	case VOTE:
		vm, ok := bc.votingMap[block.Data.VoteHash]
		if !ok {
			bc.log("the vote you wanted to add, does not have a entry in the votingMap")
			break
		}
		key := edwards.BigIntPointToEncodedBytes(block.Data.Signature.Ix, block.Data.Signature.Iy);
		_, ok = vm.votes[*key]
		if ok {
			bc.log("keyimage already in map.. check that! block should be invalid!!")
			break
		}
		vm.votes[*key] = &block.Data

	case COMPLETE_VOTING:
		vm, ok := bc.votingMap[block.Data.VoteHash]
		if !ok {
			bc.log("the reveal you wanted to add, does not have a entry in the votingMap")
			break
		}
		vm.reveal = &block.Data

	default:
		return
	}

	//fmt.Println("ADDING ", copyBytesTo32(block.Data.EdSignature))
	bc.txLookupMap[ copyBytesTo32(block.Data.EdSignature) ] = block

}

//adds a block to the chain.
//updates the votingMap if the block contained voting transactions
//asserts that block has been fully validated.
func (bc *Blockchain) addBlock(block *Block,
	pool *TxPool) bool {

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
			delete(bc.votingMap, bc.blocks[i].Hash)
			bc.lookupMap[bc.blocks[i].Hash] = false

			if !bc.blocks[i].Data.isNil() {
				delete(bc.txLookupMap, copyBytesTo32(bc.blocks[i].Data.EdSignature))
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
		switch block.Data.Typ {
		case ADD_VOTERS:
			//do nothing
		case CREATE_VOTING:
			bc.votingMap[block.Hash] = &Voting{contract: &block.Data, votes: make(map[[32]byte]*transaction)}
		case VOTE:
			vm, ok := bc.votingMap[block.Data.VoteHash]
			if !ok {
				panic("vote error 1 should not happen")
				bc.log("the vote you wanted to add, does not have a entry in the votingMap")
				break
			}
			key := edwards.BigIntPointToEncodedBytes(block.Data.Signature.Ix, block.Data.Signature.Iy);
			_, ok = vm.votes[*key]
			if ok {
				panic("vote error 2 should not happen")
				bc.log("keyimage already in map.. check that! block should be invalid!!")
				break
			}
			vm.votes[*key] = &block.Data

		case COMPLETE_VOTING:
			vm, ok := bc.votingMap[block.Data.VoteHash]
			if !ok {
				panic("complete vote error should not happen")
				bc.log("the reveal you wanted to add, does not have a entry in the votingMap")
				break
			}
			vm.reveal = &block.Data

		default:
			return true
		}
		//fmt.Println("ADDING ", copyBytesTo32(block.Data.EdSignature))
		bc.txLookupMap[ copyBytesTo32(block.Data.EdSignature) ] = block
		return true
	}

	return false
}

var emptyHash = [32]byte{}

func (bc *Blockchain) validDifficultyChange(indexParent uint32, difficultyOfChild uint8) bool {
	//TODO difficulty adaptation
	return bc.Block_ind(indexParent).Difficulty == difficultyOfChild
}

//
//func (bc *Blockchain) validatePartialChain(blocks Blocks) (invalidStartingFrom int) {
//
//	for i := 0; i < len(blocks)-1; i++ {
//		if err := blocks[i+1].isValidAncestor(blocks[i]); err != nil {
//			return int(blocks[i+1].Index)
//		}
//
//	}
//	return -1
//}

func (bc *Blockchain) String() string {
	var buffer = new(bytes.Buffer)
	for _, b := range bc.blocks {
		buffer.WriteString(fmt.Sprintf("%v :%x ->", b.Index, b.Hash[:10]))
	}
	return buffer.String()
}

//func (bc *Blockchain) isValid() bool {
//	bc.mu.RLock()
//	defer bc.mu.RUnlock()
//
//	if bc.chainHeight() == 0 {
//		return false
//	}
//	if !bc.isValidGenesisBlock() {
//		return false
//	}
//
//	prevBlock := bc.getGenesisBlock()
//	for i := 1; i < bc.chainHeight(); i++ {
//		block := bc.Block_ind(i)
//
//		if ok := isValidAncestor(block, prevBlock); !ok {
//			return false
//		}
//
//		prevBlock = block
//	}
//
//	return true
//}
