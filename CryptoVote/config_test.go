// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"
	"bytes"
	"fmt"
	"github.com/naivechain-master/CryptoNote1/edwards"
)

func TestBlockchain_Block_hash(t *testing.T) {
	voteCandidates := [][32]byte{}
	publish := [][32]byte{}
	for i := 0; i < 2; i++ {
		pubkey, priv, _ := edwards.GenerateKeyBytes()

		voteCandidates = append(voteCandidates, pubkey)

		byteres := [32]byte{}
		copy(byteres[:], priv[:32])
		publish = append(publish, byteres)
		fmt.Println("raw ", pubkey)
		fmt.Println("from secret ", byteres)
		//fmt.Println("from parse pub ",P.Serialize())
	}

	for i, _ := range voteCandidates {
		//take the transaction VoteSet entry and check if the secret parses to the public key that
		//were reveald in the CreateVote
		_, PUBfromTx := edwards.PrivKeyFromSecret(edwards.Edwards(), publish[i][:])
		if bytes.Compare(PUBfromTx.Serialize(), voteCandidates[i][:]) != 0 {
			t.Error(&errorString{fmt.Sprintf("Key Missmatch at position %v", i)})

		}
	}
}
func TestBinaryFreeList_Borrow(t *testing.T) {
	va := []int{1,2,3,4}
	fmt.Println(va[1:])
}
