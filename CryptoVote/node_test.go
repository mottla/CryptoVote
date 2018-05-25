// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"
	"math/big"
	"math/rand"
	"bytes"
	"net/http"
	"encoding/json"
	"io"
	"os"
	"fmt"
	"time"

	"encoding/hex"
	"crypto/ecdsa"
	"golang.org/x/crypto/sha3"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
	"github.com/CryptoVote/CryptoVote/CryptoNote1"
)

var (
	cosigners         = 8
	voteingCandidates = 6 //number of votingcandidates
	//voteOn            = 2 //starting at 0,1,..
	signerPrivatekey = [32]byte{10, 208, 227, 225, 224, 127, 119, 57, 208, 241, 225, 21, 244, 52, 214, 155, 198, 66, 54,
		32, 211, 17, 38, 94, 174, 127, 220, 47, 156, 18, 202, 216}
	id, err = hex.DecodeString("6c8aff29d82ffe3ce36a6bb28a41a3863ab777be96a1c27bc32fa37b4e671465698434f418fa6d389f96018f27f14d2276baccafe42ef6d43aaddeebd5c7cb0d")
)
//6c8aff29d82ffe3ce36a6bb28a41a3863ab777be96a1c27bc32fa37b4e6714657022e49a4cc2851ddfaa91751bd3827478c58d53fa195a79fc74ad1e14596200
//6c8aff29d82ffe3ce36a6bb28a41a3863ab777be96a1c27bc32fa37b4e671465f031133d7b9cb913cec0badc3b36434b86f44239ccab3016c2136676e734a705
func TestRandomUint64(t *testing.T) {
	curve := edwards.Edwards()
	privatekeys := randPrivScalarKeyList(curve, 245624576, 1)
	fmt.Printf("%x", privatekeys[0].Serialize())
	asdf, _, _ := edwards.PrivKeyFromScalar(curve, signerPrivatekey[:])
	fmt.Printf("\n%x", asdf.GetD())
}

//for adding transactions
var node = ":3000"
//for getting transactions
var getnode = ":3000"
//for vote results
var resnode = ":3000"
//steps 1 to 4 allow a full testing for a reveal required voting
//1#
func Test_VOTESET_VOTECONTRACT(t *testing.T) {

	curve := edwards.Edwards()

	fmt.Println("\nCreate serializedPubKeys and post a vote set to node...")

	//using deterministic random, so each time we run the test, we get the same keys..
	privatekeys := randPrivScalarKeyList(curve, 4352, cosigners+1)

	serializedPubKeys := [][32]byte{}

	for _, v := range privatekeys {
		a, b := v.Public()
		pk := edwards.NewPublicKey(curve, a, b)
		temp := [32]byte{}
		copy(temp[:], pk.Serialize())
		serializedPubKeys = append(serializedPubKeys, temp)
	}

	ADD_VOTERS_tx := NewTransaction(&transaction{
		Typ:     ADD_VOTERS,
		PubKeys: serializedPubKeys,
	}, curve, &signerPrivatekey)

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(ADD_VOTERS_tx)
	res, _ := http.Post(fmt.Sprintf("http://127.0.0.1%s/addTransaction", node), "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)

	fmt.Println("\nStart the nodes miner...")
	//mine this block
	mineOne()

	fmt.Println("\nAsk the node for the blockhash holding the voteset...")
	fmt.Printf("%x", ADD_VOTERS_tx.EdSignature)
	_, err := getBlock(ADD_VOTERS_tx.EdSignature)
	if err != nil {
		t.Error("get block by id failed")
		return
	}

	fmt.Println("\nCreate Vote Now...")

	voteCandidates := [][32]byte{}
	privatekeysVoters := randPrivScalarKeyList(curve, 1111, voteingCandidates)
	serializedVoterPrivKeys := [][32]byte{}

	for _, v := range privatekeysVoters {

		a, b := v.Public()
		pk := edwards.NewPublicKey(curve, a, b)
		temp := [32]byte{}
		copy(temp[:], pk.Serialize())
		voteCandidates = append(voteCandidates, temp)
		temp2 := [32]byte{}
		copy(temp2[:], v.Serialize())
		serializedVoterPrivKeys = append(serializedVoterPrivKeys, temp2)
	}

	//we now have our voteset on the chain and got the containing hash returned. lets work with that
	CREATE_VOTING_tx := NewTransaction(&transaction{
		Typ:          CREATE_VOTING,
		VoteSet:      []TransactionID{copyBytesTo32(ADD_VOTERS_tx.EdSignature)},
		PubKeys:      voteCandidates,
		RevealNeeded: true,
	}, curve, &signerPrivatekey)

	fmt.Printf("\nVoting has unique id: %x", CREATE_VOTING_tx.EdSignature)

	b = new(bytes.Buffer)
	json.NewEncoder(b).Encode(CREATE_VOTING_tx)
	res, _ = http.Post(fmt.Sprintf("http://127.0.0.1%s/addTransaction", node), "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)

	fmt.Println("\nStart the miner again...")
	//mine this block
	mineOne()
}

//2#
func Test_VOTE(t *testing.T) {

	curve := edwards.Edwards()
	voter := 1
	voteon := 1

	privatekeys := randPrivScalarKeyList(curve, 4352, cosigners+1)

	serializedPubKeys := [][32]byte{}
	pubkeys := []*ecdsa.PublicKey{}

	for _, v := range privatekeys {
		a, b := v.Public()
		pk := edwards.NewPublicKey(curve, a, b)
		pubkeys = append(pubkeys, pk.ToECDSA())
		temp := [32]byte{}
		copy(temp[:], pk.Serialize())
		serializedPubKeys = append(serializedPubKeys, temp)

	}

	voteCandidates := [][32]byte{}
	privatekeysVoters := randPrivScalarKeyList(curve, 1111, voteingCandidates)
	fmt.Println()
	serializedVoterPrivKeys := [][32]byte{}

	for _, v := range privatekeysVoters {
		a, b := v.Public()
		pk := edwards.NewPublicKey(curve, a, b)
		temp := [32]byte{}
		copy(temp[:], pk.Serialize())
		voteCandidates = append(voteCandidates, temp)
		temp2 := [32]byte{}
		copy(temp2[:], v.Serialize())
		serializedVoterPrivKeys = append(serializedVoterPrivKeys, temp2)

	}

	fmt.Println("\nStarting vote process. First we need to get the blockhash holding the Vote...")

	_, err := getBlock(*copyBytes64(id));
	if err != nil {
		t.Error("get block by id failed")
		return
	}

	fmt.Println("\nWe found the block. Now LSAG")

	VOTEON, err := edwards.ParsePubKey(curve, voteCandidates[voteon][:])
	if err != nil {
		t.Error("parsing failed")
		return
	}

	P, R := CryptoNote1.GenerateOneTime_VOTE(VOTEON, nil, nil)
	Pbyt := [32]byte{}
	Rbyt := [32]byte{}
	copy(Rbyt[:], R.Serialize())
	copy(Pbyt[:], P.Serialize())
	hasher := sha3.New256()
	hasher.Reset()
	tempid := copyBytesTo32(*copyBytes64(id))
	hasher.Write(tempid[:])
	hasher.Write(Pbyt[:])
	message := hasher.Sum(nil)

	sig := CryptoNote1.NewLSAG(privatekeys[voter].ToECDSA(), nil, nil)

	//fmt.Printf("Creating Linkable Spontaneous Anonymous Groups Signature\non %s, using SHA256.  SignerPublic position is %v of %v\n", curve.Params().Name, sig.signerPosition, chainHeight(sig.pubKeys))
	sig.Sign(message, pubkeys, privatekeys[voter].ToECDSA(), voter)

	fmt.Printf("\nSignature on %x \n with %v", message, sig)
	go func() { fmt.Print(sig.Verify(message, pubkeys)) }()

	//fmt.Printf("\n Keys %v", pubkeys)

	VOTE_tx := NewTransaction(&transaction{
		Typ:           VOTE,
		Signature:     sig.Sigma,
		VoteTo:        Pbyt,
		VoteID:        copyBytesTo32(*copyBytes64(id)),
		RevealElement: Rbyt,
	}, curve, &signerPrivatekey)

	//fmt.Printf("%v\n%v\n%v", u.Signature.Ci, u.Signature.Ri)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(VOTE_tx)
	http.Post(fmt.Sprintf("http://127.0.0.1%s/addTransaction", node), "application/json; charset=utf-8", b)

	mineOne()
}

//3#
func Test_REVEAL(t *testing.T) {
	_, err := getBlock(*copyBytes64(id));
	if err != nil {
		t.Error("get block by id failed")
		return
	}
	fmt.Println("\nRevealing now")

	curve := edwards.Edwards()

	voteCandidates := [][32]byte{}
	privatekeysVoters := randPrivScalarKeyList(curve, 1111, voteingCandidates)
	serializedVoterPrivKeys := [][32]byte{}

	for _, v := range privatekeysVoters {

		a, b := v.Public()
		pk := edwards.NewPublicKey(curve, a, b)
		temp := [32]byte{}
		copy(temp[:], pk.Serialize())
		voteCandidates = append(voteCandidates, temp)
		temp2 := [32]byte{}
		copy(temp2[:], v.Serialize())
		serializedVoterPrivKeys = append(serializedVoterPrivKeys, temp2)
	}

	Reveal_VOTE_tx := NewTransaction(&transaction{
		Typ:         REVEAL_VOTING,
		PrivateKeys: serializedVoterPrivKeys,
		VoteID:      copyBytesTo32(*copyBytes64(id)),
	}, curve, &signerPrivatekey)
	//fmt.Printf("%v\n%v\n%v", u.Signature.Ci, u.Signature.Ri)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(Reveal_VOTE_tx)
	http.Post(fmt.Sprintf("http://127.0.0.1%s/addTransaction", node), "application/json; charset=utf-8", b)
	mineOne()
}

//4#
func Test_VoteRes(t *testing.T) {
	m, err := getVoteResults(id)
	if err != nil {
		t.Error(err)
		return
	}
	for key, value := range m {
		fmt.Println("Key:", key, "Value:", value)
	}
}

func getVoteResults(id []byte) (c map[string]uint, err error) {
	var attempts = 5;
	searchid := map[string][32]byte{"id": copyBytesTo32(*copyBytes64(id))}
	resC := make(map[string]uint)

	for ; attempts > 0; attempts-- {
		time.Sleep(3 * time.Second)
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(searchid)
		res, _ := http.Post(fmt.Sprintf("http://127.0.0.1%s/voteResults", resnode), "application/json; charset=utf-8", b)

		json.NewDecoder(res.Body).Decode(&resC)
		if len(resC) != 0 {
			return resC, err
		}
	}

	return c, &errorString{"Get voteresults by ID failed"}

}

func randPrivScalarKeyList(curve *edwards.TwistedEdwardsCurve, randSeed int64, i int) []*edwards.PrivateKey {
	r := rand.New(rand.NewSource(randSeed))

	privKeyList := make([]*edwards.PrivateKey, i)
	for j := 0; j < i; j++ {
		for {
			bIn := new([32]byte)
			for k := 0; k < edwards.PrivScalarSize; k++ {
				randByte := r.Intn(255)
				bIn[k] = uint8(randByte)
			}

			bInBig := new(big.Int).SetBytes(bIn[:])
			bInBig.Mod(bInBig, curve.N)
			copy(bIn[:], bInBig.Bytes())
			bIn[31] &= 248

			pks, _, err := edwards.PrivKeyFromScalar(curve, bIn[:])
			if err != nil {
				r.Seed(int64(j) + r.Int63n(12345))
				continue
			}

			// No duplicates allowed.
			if j > 0 &&
				(bytes.Equal(pks.Serialize(), privKeyList[j-1].Serialize())) {
				continue
			}

			privKeyList[j] = pks
			r.Seed(int64(j) + randSeed)
			break
		}
	}

	return privKeyList
}

func getBlock(transactionID [64]byte) (bl Block, err error) {
	id := copyBytesTo32(transactionID)
	var attempts = 5;
	searchid := map[string][32]byte{"id": id}

	for ; attempts > 0; attempts-- {
		time.Sleep(3 * time.Second)
		b := new(bytes.Buffer)
		json.NewEncoder(b).Encode(searchid)
		res, _ := http.Post(fmt.Sprintf("http://127.0.0.1%s/transactionByID", getnode), "application/json; charset=utf-8", b)

		json.NewDecoder(res.Body).Decode(&bl)
		if bl.Hash != [32]byte{} {
			return bl, err
		}
	}

	return bl, &errorString{"Get Block by ID failed"}

}

func mineOne() {
	params := map[string]uint8{"procs": 7, "amount": 1}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(params)
	res, _ := http.Post(fmt.Sprintf("http://127.0.0.1%s/startMining", node), "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)
}

// copyBytes64 copies a byte slice to a 64 byte array.
func copyBytes64(aB []byte) *[64]byte {
	if aB == nil {
		return nil
	}

	s := new([64]byte)

	// If we have a short byte string, expand
	// it so that it's long enough.
	aBLen := len(aB)
	if aBLen < 64 {
		diff := 64 - aBLen
		for i := 0; i < diff; i++ {
			aB = append([]byte{0x00}, aB...)
		}
	}

	for i := 0; i < 64; i++ {
		s[i] = aB[i]
	}

	return s
}
