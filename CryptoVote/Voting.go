// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"sync"
	"bytes"
	"fmt"
	"github.com/CryptoVote/CryptoVote/CryptoNote1"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
)

type voting struct {
	id              [32]byte
	lock            sync.Mutex
	voteSetMap      map[[32]byte]*transaction //key is the voteSet id
	contract        *transaction
	votesMap        map[[32]byte]*transaction //key is the keyimage to avoid multiple voting with the same key
	reveal          *transaction
	selfMaintaining bool
	voteResults     map[string]uint
}

//returns a empty voting container
//selfMaintaining = true means, that the order of inserting matters, also it automatically computes and stores the votingresult once the reveal has been added
func newVoting(selfMaintaining bool) (v *voting) {
	return &voting{
		votesMap:        make(map[[32]byte]*transaction),
		voteSetMap:      make(map[[32]byte]*transaction),
		voteResults:     make(map[string]uint),
		lock:            sync.Mutex{},
		selfMaintaining: selfMaintaining,
	}
}

//adds transaction to voting.
//asserts that maps have been initialized.
//overrides and does no double spend checking
//safe for concurrent access
func (v *voting) add(t *transaction) {

	//todo consider self maintaining ordering
	if t == nil {
		return
	}
	v.lock.Lock()

	if !(v.selfMaintaining && v.IsRevealed()) {
		switch t.Typ {

		case CREATE_VOTING:
			v.id = copyBytesTo32(t.EdSignature)
			v.contract = t

		case VOTE:

			key, err := t.keyImage()
			if err != nil {
				return
			}
			v.votesMap[*key] = t
		case REVEAL_VOTING:
			v.reveal = t
			if v.selfMaintaining {
				v.lock.Unlock()
				v.Voteresults()
				return
			}
		case ADD_VOTERS:
			v.voteSetMap[copyBytesTo32(t.EdSignature)] = t
		default:

		}
	}
	v.lock.Unlock()
}

//safe for concurrent access
func (v *voting) contains(t *transaction) (ok bool) {
	v.lock.Lock()
	defer v.lock.Unlock()

	switch t.Typ {
	case CREATE_VOTING:
		if v.contract != nil {
			ok = v.contract.EdSignature == t.EdSignature
		}
	case VOTE:
		key, err := t.keyImage()
		if err != nil {
			//should not happen
			panic(err)
			return
		}
		_, ok = v.votesMap[*key]
	case REVEAL_VOTING:
		if v.reveal != nil {
			ok = v.reveal.EdSignature == t.EdSignature
		}
	case ADD_VOTERS:
		_, ok = v.voteSetMap[copyBytesTo32(t.EdSignature)]
	default:
		ok = false
	}
	return ok
}

//removes transaction from voting. Returns true if removed, false otherwise
func (v *voting) remove(t *transaction) (ok bool) {
	v.lock.Lock()
	defer v.lock.Unlock()

	switch t.Typ {
	case CREATE_VOTING:
		if v.contract != nil {
			ok = v.contract.EdSignature == t.EdSignature
			if ok {
				v.contract = nil
			}
		}

	case VOTE:
		key, err := t.keyImage()
		if err != nil {
			return
		}
		_, ok = v.votesMap[*key]
		if ok {
			delete(v.votesMap, *key)
		}
	case REVEAL_VOTING:
		if v.reveal != nil {
			ok = v.reveal.EdSignature == t.EdSignature
			if ok {
				v.reveal = nil
			}
		}
	case ADD_VOTERS:

		_, ok = v.voteSetMap[copyBytesTo32(t.EdSignature)]
		if ok {
			delete(v.voteSetMap, copyBytesTo32(t.EdSignature))
		}
	default:
		ok = false
	}
	return

}

func (v *voting) IsDoubleSpend(keyImage TransactionID) (yes bool) {
	v.lock.Lock()
	defer v.lock.Unlock()
	_, yes = v.votesMap[keyImage]

	return
}

func (v *voting) AllAllowedVotersPubkeys() (PubKeys [][32]byte, err error) {
	v.lock.Lock()
	defer v.lock.Unlock()
	keys := make([][32]byte, 0)

	for _, pub := range v.contract.VoteSet {
		vsTx, ok := v.voteSetMap[pub];
		if !ok {
			return nil, errorCall("Votesets in voting missing")
		}
		for _, key := range vsTx.PubKeys {
			keys = append(keys, key)
		}

	}

	return keys, nil
}

func (v *voting) AllowedCandidate(voteTo [32]byte) (yes bool) {

	//todo maybe move candidate pubkeys into map for speed
	for _, v := range v.contract.PubKeys {
		if bytes.Compare(v[:], voteTo[:]) == 0 {
			return true
		}
	}
	return
}

//returns false, if voting contract is set to: no_reveal_needed = true
func (v *voting) IsRevealed() (yes bool) {

	if v.contract == nil {
		return false
	}
	if !v.contract.RevealNeeded {
		return false
	}

	return v.reveal != nil
}

func (v *voting) RevealeNeeded() (yes bool) {
	return v.contract.RevealNeeded
}

//mostValueableToMineNext returns a transaction from the voting container.
// we set the default node to a friendly one
//eg. we first try to mine all available  votesMap, before including the reveal which closes the chance for all the un-mined ones to have their voting confirmed/considered
func (v *voting) mostValueableToMineNext() (t *transaction) {
	v.lock.Lock()
	defer v.lock.Unlock()

	//the following return orders are relevant

	for _, v := range v.voteSetMap {
		return v
	}

	if v.contract != nil {
		return v.contract
	}

	for _, v := range v.votesMap {
		return v
	}

	if v.reveal != nil {
		return v.reveal
	}

	return nil
}

//pop returns a transaction from the voting container by removing it from it. This is useful for
//maintaining the transactionpool. we set the default node to a friendly one
//eg we first try to mine all available  votesMap, before including the reveal which closes the chance for all the un-mined ones to have their voting confirmed/considered
func (v *voting) pop() (t *transaction) {
	v.lock.Lock()
	defer v.lock.Unlock()

	for i, _ := range v.voteSetMap {
		t = v.voteSetMap[i]
		delete(v.voteSetMap, i)
		return t
	}

	if v.contract != nil {
		t = v.contract
		v.contract = nil
		return t
	}

	for i, _ := range v.votesMap {
		t = v.votesMap[i]
		delete(v.votesMap, i)
		return t
	}

	if v.reveal != nil {
		t = v.reveal
		v.reveal = nil
		return t
	}

	return nil
}

func (vote *voting) Voteresults() (counter map[string]uint, err error) {
	vote.lock.Lock()
	defer vote.lock.Unlock()

	if vote.RevealeNeeded() && !vote.IsRevealed() {
		return counter, errorCall("Votingresults were not reveled yet. Try again later!")
	}

	//this is a self maintaining vote container and its results have been computed once alredady. we return the old result.
	if vote.selfMaintaining && len(vote.voteResults) > 0 {
		return vote.voteResults, nil
	}

	counter = make(map[string]uint)

	if !vote.RevealeNeeded() {

		//set each candidate to 0
		for _, v := range vote.contract.PubKeys {
			counter[fmt.Sprintf("%x", v)] = 0
		}
		//sum up, if the candidate exists
		for _, v := range vote.votesMap {
			if vote.AllowedCandidate(v.VoteTo) {
				counter[fmt.Sprintf("%x", v.VoteTo)] += 1
			}

		}
		if vote.selfMaintaining {
			vote.voteResults = counter
		}
		return counter, nil
	}

	if vote.IsRevealed() {

		edcurve := edwards.Edwards()
		for _, v := range vote.contract.PubKeys {
			counter[fmt.Sprintf("%x", v)] = 0
		}

		for _, v := range vote.votesMap {
			for _, secret := range vote.reveal.PrivateKeys {
				var P, R, pub *edwards.PublicKey
				var private *edwards.PrivateKey
				var err1, err2, err3 error
				//parsing serialized ed25519 points into public and privatekeys
				private, pub, err1 = edwards.PrivKeyFromScalar(edcurve, secret[:])
				P, err2 = edwards.ParsePubKey(edcurve, v.VoteTo[:])
				R, err3 = edwards.ParsePubKey(edcurve, v.RevealElement[:])
				if err1 != nil || err2 != nil || err3 != nil {
					return nil, errorCall("Error Parsing scalar to edwards key")
				}
				if CryptoNote1.VerifyOneTime_VOTE(private, *P, *R, nil, edcurve) {
					counter[fmt.Sprintf("%x", pub.Serialize())] += 1
					continue
				}
			}
		}

	}
	if vote.selfMaintaining {
		vote.voteResults = counter
	}
	return
}
