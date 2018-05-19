// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"

	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
)

func TestTransaction_String(t *testing.T) {
	curve := edwards.Edwards()
	pos := 1

	privatekeys := randPrivScalarKeyList(curve, 4352, cosigners+1)
	serializedPubKeys := [][32]byte{}
	pubkeys := []edwards.PublicKey{}

	for _, v := range privatekeys {
		a, b := v.Public()
		pk := edwards.NewPublicKey(curve, a, b)
		pubkeys = append(pubkeys, *pk)
		temp := [32]byte{}
		copy(temp[:], pk.Serialize())
		serializedPubKeys = append(serializedPubKeys, temp)
	}

	ADD_VOTERS_tx := NewTransaction(&transaction{
		Typ:     ADD_VOTERS,
		PubKeys: serializedPubKeys,
	}, curve, copyBytes(privatekeys[pos].Serialize()))

	//ADD_VOTERS_tx.VoteTo=[32]byte{1}
	if !ADD_VOTERS_tx.verifySignature(curve) {
		t.Error("failed")
	}
}
