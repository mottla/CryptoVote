// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"sync"
	"io"
	"crypto/rand"

	//"bytes"
	"log"
	"os"
	"fmt"
)

// TxPool is used as a source of transactions that need to be mined into blocks
// and relayed to other peers.  It is safe for concurrent access from multiple
// peers.
type TxPool struct {
	lastUpdated int64 // last time pool was updated
	mtx         sync.RWMutex
	votingMap   map[[32]byte]*voting      //key: ed sig of vontings ->voting contract transaction
	voteSetMap  map[[32]byte]*transaction //key: ed sig. value voteSet transaction only
	logger      *log.Logger
}

func (node *TxPool) log(v ...interface{}) {
	node.logger.Println(v)
}

func (node *TxPool) logError(err error) {
	node.log("[ERROR]", err)
}

//returns the block, holding a transaction.
func (mp *TxPool) votingContractById(id [32]byte) (res *voting, ok bool) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	//bc.log("looking for %v in accesmap", hash)
	res, ok = mp.votingMap[id]

	return
}

func (mp *TxPool) contains(trans *transaction) (ok bool) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	var voting *voting
	switch trans.Typ {
	case ADD_VOTERS:
		_, ok = mp.voteSetMap[copyBytesTo32(trans.EdSignature)]
		return
	case CREATE_VOTING:
		voting, ok = mp.votingMap[copyBytesTo32(trans.EdSignature)]
		if !ok {
			return false
		}
	default:
		voting, ok = mp.votingMap[trans.VoteID]
		if !ok {
			return false
		}
	}
	return voting.contains(trans)
}

//tell the pool that we included a reveal and hence all votesMap that are still pending, can be rejected
func (mp *TxPool) revealUpdate(key [32]byte) (unincludedTransactionCount int) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	if v, ok := mp.votingMap[key]; ok {
		unincludedTransactionCount = len(v.votesMap)
		delete(mp.votingMap, key)
	}

	return 0

}

//picks an arbitrary transactionByID from pool
//this approach is very naive and for this educational purpose only
func (mp *TxPool) randPoolTransaction() (tx *transaction) {

	b := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return
	}
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	//if no voting was there, we include the votesets no one needs so far
	for _, voting := range mp.voteSetMap {
		return voting
	}

	//pick a voting at random
	for key, _ := range mp.votingMap {
		//gives us the most important tx to be mined nex and removes it from the voting
		tx = mp.votingMap[key].mostValueableToMineNext()
		if tx != nil {
			return tx

		}
		//tx is nill->we emptied it, now remove the container
		delete(mp.votingMap, key)
	}


	return new(transaction)
}

func newTxPool(apiAddr string) *TxPool {
	return &TxPool{
		mtx:        sync.RWMutex{},
		votingMap:  make(map[[32]byte]*voting),
		voteSetMap: make(map[[32]byte]*transaction),
		logger:     log.New(os.Stdout, fmt.Sprintf("[%v] Pool: ", apiAddr), log.Ldate|log.Ltime, )}
}


func (mp *TxPool) addTransaction(trans *transaction) {
	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	mp.log("adding new transaction to pool ", trans.Typ.name())

	switch trans.Typ {
	case ADD_VOTERS:

		mp.voteSetMap[copyBytesTo32(trans.EdSignature)] = trans

	case CREATE_VOTING:
		voting, ok := mp.votingMap[copyBytesTo32(trans.EdSignature)]
		if !ok {
			voting = newVoting(false)
			mp.votingMap[copyBytesTo32(trans.EdSignature)] = voting
		}
		voting.add(trans)
		//for _, id := range trans.VoteSet {
		//	if v, ok := mp.voteSetMap[id]; ok {
		//		voting.add(v)
		//		delete(mp.voteSetMap, id)
		//	}
		//}
	default:
		voting, ok := mp.votingMap[trans.VoteID]
		if !ok {
			voting = newVoting(false)
			mp.votingMap[trans.VoteID] = voting
		}
		voting.add(trans)

	}

}


func (mp *TxPool) remove(trans *transaction) (ok bool) {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	var voting *voting
	switch trans.Typ {
	case ADD_VOTERS:

		_, ok = mp.voteSetMap[copyBytesTo32(trans.EdSignature)]
		if ok {
			delete(mp.voteSetMap,copyBytesTo32(trans.EdSignature))
			return
		}

	case CREATE_VOTING:
		voting, ok = mp.votingMap[copyBytesTo32(trans.EdSignature)]
		if ok {
			//i could check if the container is empty and remove it then
			return voting.remove(trans)
		}

	default:
		voting, ok = mp.votingMap[trans.VoteID]
		if ok {
			//i could check if the container is empty and remove it then
			return voting.remove(trans)
		}

	}

	return
}
