// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.

package CrptoNote1

import (
	"math/big"
	"github.com/naivechain-master/CryptoNote1/edwards"
	"hash"
	"crypto/elliptic"
	//"crypto/ecdsa"
)

//takes an elliptic Curve and a public key pair (A,B) as input
//returns a One-Time-Public-Stealth-Key P=Hs(rA)G+BG and R=rA
//where r is a random integer < Curve.N
//only the possessor of the (A;B) corresponding private-keys is able to reconstruct
//the private-key corresponding too P
func GenerateOneTimePK(pkP publicKeyPair, hasher hash.Hash, Curve elliptic.Curve) (Px, Py, Rx, Ry *big.Int) {

	if Curve == nil {
		Curve = defaultCurve
	}

	if hasher == nil {
		hasher = defaultHasher
	} else if hasher.Size() != 32 {
		panic("only hashes with outputsize of 32 bytes allowed!", )
	}

	hasher.Reset()
	r := RandFieldElement(Curve)
	// X1,y1 = Hs(rA)G
	Px, Py = Curve.ScalarMult(pkP.Ax, pkP.Ay, r.Bytes())
	re := hasher.Sum(append(Px.Bytes()[:], Py.Bytes()[:]...))
	ra := new(big.Int).SetBytes(re[:])
	ra.Mod(ra, Curve.Params().N)
	Px, Py = Curve.ScalarBaseMult(ra.Bytes())
	//+BG
	Px, Py = Curve.Add(Px, Py, pkP.Bx, pkP.By)
	Rx, Ry = Curve.ScalarBaseMult(r.Bytes())
	return
}

//takes an elliptic Curve and a public key pair (A,B) as input
//returns a One-Time-Public-Stealth-Key P=Hs(rA)G and R=rA
//where r is a random integer < Curve.N
//only the possessor of the (A) corresponding private-keys is able to reconstruct
//the private-key corresponding too P
func GenerateOneTime_VOTE(key *edwards.PublicKey, hasher hash.Hash, Curve elliptic.Curve) (P, R edwards.PublicKey) {

	if Curve == nil {
		Curve = defaultCurve
	}

	if hasher == nil {
		hasher = defaultHasher
	} else if hasher.Size() != 32 {
		panic("only hashes with outputsize of 32 bytes allowed!", )
	}

	hasher.Reset()
	r := RandFieldElement(Curve)
	// X1,y1 = Hs(rA)G
	Px, Py := Curve.ScalarMult(key.X, key.Y, r.Bytes())
	re := hasher.Sum(append(Px.Bytes()[:], Py.Bytes()[:]...))
	ra := new(big.Int).SetBytes(re[:])
	ra.Mod(ra, Curve.Params().N)
	Px, Py = Curve.ScalarBaseMult(ra.Bytes())
	P = edwards.PublicKey{X: Px, Y: Py}
	Rx, Ry := Curve.ScalarBaseMult(r.Bytes())
	R = edwards.PublicKey{X: Rx, Y: Ry}
	return
}

//let the user u check, if he is the owner of a given Public-Stealth-Key P and shared secret R
//returns the corresponding private key that allows signing the given Stealth-Key if he is the owner
//returns nil if he does not own the Stealth-Key
func VerifyOneTime_VOTE(valKey *edwards.PrivateKey, P, R edwards.PublicKey, hasher hash.Hash, Curve elliptic.Curve) (success bool) {
	//TODO check concurrency here.. this method is not threadsecure

	if Curve == nil {
		Curve = defaultCurve
	}

	if hasher == nil {
		hasher = defaultHasher
	} else if hasher.Size() != 32 {
		panic("only hashes with outputsize of 32 bytes allowed!", )
	}

	hasher.Reset()
	px, py := Curve.ScalarMult(R.X, R.Y, valKey.GetD().Bytes())
	re := hasher.Sum(append(px.Bytes()[:], py.Bytes()[:]...))
	x := new(big.Int).SetBytes(re[:])
	px, py = Curve.ScalarBaseMult(x.Bytes())
	if px.Cmp(P.X) == 0 && py.Cmp(P.Y) == 0 {
		return true
	}
	return false

}

//let the user u check, if he is the owner of a given Public-Stealth-Key P and shared secret R
//returns the corresponding private key that allows signing the given Stealth-Key if he is the owner
//returns nil if he does not own the Stealth-Key
func (u *user) VerifyOneTimePK(Px, Py, Rx, Ry *big.Int, hasher hash.Hash, Curve elliptic.Curve) (success bool, x *big.Int) {
	hasher.Reset()
	px, py := Curve.ScalarMult(Rx, Ry, u.A.D.Bytes())
	re := hasher.Sum(append(px.Bytes()[:], py.Bytes()[:]...))
	x = new(big.Int).Add(u.B.D, new(big.Int).SetBytes(re[:]))
	px, py = Curve.ScalarBaseMult(x.Bytes())
	if px.Cmp(Px) == 0 && py.Cmp(Py) == 0 {
		return true, x
	}
	return false, nil

}
