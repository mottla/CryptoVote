// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"fmt"

	"bytes"
	"github.com/agl/ed25519"
	"github.com/CryptoVote/CryptoVote/CryptoNote1"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
	"golang.org/x/crypto/sha3"
)

type TransactionType byte
type TransactionID [32]byte

const (
	CREATE_VOTING TransactionType = 12
	VOTE          TransactionType = 13
	ADD_VOTERS    TransactionType = 11
	REVEAL_VOTING TransactionType = 14
)

var emptySig = [64]byte{}


//this struct is used for all kinds of possible moves than can be made on the naive chain
//e.g. Create a Vote, Vote on something, Create a Set of Voters (list of public keys)
//to 'create a vote', on needs to place at least on 'set of voteSetMap' first on the chain. After confirmation
// a 'create a vote' is possible.
//To 'vote on something' one needs to proof the ownership of a corresponding privatekey contained in a set of allowed voteSetMap
//fot the selected voting contract. We do so by using LSAG signatures
type transaction struct {
	EdSignature  [64]byte `json:"ID"`   //the signature is used to verify the transaction, by
	SignerPublic [32]byte `json:"Data"` //needed to verify the signature.

	Typ           TransactionType   `json:"Typ"`
	VoteSet       []TransactionID   `json:"voteset"`       //set of blockhashes holding the type specific information
	PubKeys       [][32]byte        `json:"pubkeys"`       //set containing public ed25519 points
	Signature     CryptoNote1.Sigma `json:"sig"`           //Ring signature
	VoteID        TransactionID     `json:"ID_of_Voting"`  //for Vote and reveal. Points on the vote contract edwars signature
	VoteTo        [32]byte          `json:"voteOnAddress"` //for Vote transactionByID only. VoteTo is one of the addresses listet by the contract a user can vote on
	RevealElement [32]byte          `json:"reveal_element"`
	PrivateKeys   [][32]byte        `json:"private_keys"` //only needed to reaveal the votingresults for a COMPLETE_VOTE transaction
	RevealNeeded  bool              `json:"reveal_needed"`
}

func (t *transaction) isEmpty() bool {

	//non vote transaction must have a edsignature
	if t.Typ != VOTE && bytes.Compare(t.EdSignature[:], emptySig[:]) == 0 {
		return true
	}
	if t.Typ == VOTE {
		return t.VoteID == emptyHash
	}
	return false
}

func (t *transaction) keyImage() (img *[32]byte, err error) {
	if t.Typ == VOTE {
		if t.Signature.Ix == nil || t.Signature.Iy == nil {
			return nil, errorCall("transaction should have a signature !=nil ")
		}
		return edwards.BigIntPointToEncodedBytes(t.Signature.Ix, t.Signature.Iy), nil
	}
	return nil, errorCall("cannot get keyimage from non-vote transaction")
}

//TODO if one sends huge transactions this check can be costly..
func (t *transaction) toMsg() *bytes.Buffer {

	buffer := new(bytes.Buffer)
	buffer.Write(t.SignerPublic[:])
	buffer.Write([]byte{byte(t.Typ)})
	buffer.Write([]byte{byte(t.Typ)})
	for i, _ := range t.VoteSet {
		buffer.Write(t.VoteSet[i][:])
	}
	for i, _ := range t.PubKeys {
		buffer.Write(t.PubKeys[i][:])
	}
	if t.Signature.Ix != nil {
		t.Signature.BytesToBuffer(buffer)
	}

	buffer.Write(t.VoteID[:])
	buffer.Write(t.VoteTo[:])
	buffer.Write(t.RevealElement[:])
	for i, _ := range t.PrivateKeys {
		buffer.Write(t.PrivateKeys[i][:])
	}
	if t.RevealNeeded {
		buffer.Write([]byte{1})
	} else {
		buffer.Write([]byte{0})
	}
	return buffer

}

//check if the signature matches the transaction data
//Their ringsignature will be checked after some preprocessing
func (t *transaction) verifySignature(curve *edwards.TwistedEdwardsCurve) bool {
	if t.isEmpty() {
		return false
	}
	msg := t.toMsg().Bytes()
	return ed25519.Verify(&t.SignerPublic, msg, &t.EdSignature)
}

func NewTransaction(t *transaction, curve *edwards.TwistedEdwardsCurve, scalar *[32]byte) *transaction {
	if t.Typ == VOTE {
		//a voting transaction must not be signed with edwards! would make ringsignature useless :)
		//instead we use its hash for identification
		t.EdSignature=sha3.Sum512(t.toMsg().Bytes())
		return t
	}
	t.hashAndSign(curve, scalar)
	return t
}

//take a secret and use it to create a ed25519 signature (R,S)
func (t *transaction) hashAndSign(curve *edwards.TwistedEdwardsCurve, scalar *[32]byte) *transaction {
	priv, pub, _ := edwards.PrivKeyFromScalar(curve, scalar[:])
	t.SignerPublic = *copyBytes(pub.Serialize())
	msg := t.toMsg().Bytes()
	r, s, err := edwards.Sign(curve, priv, msg)
	if err != nil {
		fmt.Printf("unexpected error %s", err)
		return t
	}

	temp2 := [64]byte{}
	copy(temp2[:32], edwards.BigIntToEncodedBytes(r)[:])
	copy(temp2[32:], edwards.BigIntToEncodedBytes(s)[:])
	t.EdSignature = temp2
	return t
}

// copyBytes copies a byte slice to a 32 byte array.
func copyBytes(aB []byte) *[32]byte {
	if aB == nil {
		return nil
	}
	s := new([32]byte)

	// If we have a short byte string, expand
	// it so that it's long enough.
	aBLen := len(aB)
	if aBLen < 32 {
		diff := 32 - aBLen
		for i := 0; i < diff; i++ {
			aB = append([]byte{0x00}, aB...)
		}
	}

	for i := 0; i < 32; i++ {
		s[i] = aB[i]
	}

	return s
}

func (s transaction) String() string {

	switch s.Typ {
	case CREATE_VOTING:
		return fmt.Sprintf(
			"Typ: %v | Votesets: %v | Candidates: %v ", s.Typ.name(), s.VoteSet, s.PubKeys)
	case REVEAL_VOTING:
		return fmt.Sprintf(
			"Typ: %v | Revealing : %v | Candidates privateKeys: %v ", s.Typ.name(), s.VoteID, s.PrivateKeys)
	case VOTE:
		return fmt.Sprintf(
			"Typ: %v | on Contract: %v | vote goes to: %v | the signature:", s.Typ.name(), s.VoteID, s.VoteTo, s.Signature)
	case ADD_VOTERS:
		return fmt.Sprintf(
			"Typ: %v | Allowed Voters are: %v", s.Typ.name(), s.PubKeys)
	default:
		return "UNKNOWN"
	}
}

func (ms TransactionType) name() string {
	switch ms {
	case CREATE_VOTING:
		return "Create voting"
	case VOTE:
		return "Vote"
	case ADD_VOTERS:
		return "Vote Set"
	case REVEAL_VOTING:
		return "Complete voting"
	default:
		return "UNKNOWN Transaction Type"
	}
}
func (ms TransactionType) exists() bool {
	switch ms {
	case CREATE_VOTING:
		return true
	case VOTE:
		return true
	case ADD_VOTERS:
		return true
	case REVEAL_VOTING:
		return true;
	default:
		return false
	}
}
