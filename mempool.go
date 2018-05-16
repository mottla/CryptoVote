package main

import (
	"sync"
	"io"
	"crypto/rand"

	//"bytes"
	"log"
)

// TxPool is used as a source of transactions that need to be mined into blocks
// and relayed to other peers.  It is safe for concurrent access from multiple
// peers.
type TxPool struct {
	lastUpdated int64 // last time pool was updated
	mtx         sync.RWMutex
	poolMap     map[[64]byte]*transaction
	logger      *log.Logger
}

func (node *TxPool) log(v ...interface{}) {
	node.logger.Println(v)
}

func (node *TxPool) logError(err error) {
	node.log("[ERROR]", err)
}

func (mp *TxPool) addTransaction(trans *transaction) bool {
	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()


	//if bytes.Compare(trans.EdSignature[:], [64]byte{}[:])C

	mp.poolMap[trans.EdSignature] = trans
	mp.log("added new transaction to pool %v", trans.Typ)
	return true

}

func (mp *TxPool) contains(tx *transaction) bool {
	mp.mtx.Lock()
	defer mp.mtx.Unlock()
	if mp.poolMap[tx.EdSignature] != nil {
		return true
	}
	return false

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
	for _, value := range mp.poolMap {
		return value
	}

	return new(transaction)
}

func (mp *TxPool) remove(tx *transaction) bool {
	// Protect concurrent access.
	mp.mtx.Lock()
	defer mp.mtx.Unlock()

	if mp.poolMap[tx.EdSignature] != nil {
		delete(mp.poolMap, tx.EdSignature)
		return true
	}

	return false

}
