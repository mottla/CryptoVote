// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package CryptoNote1

import (
	"testing"
	"fmt"
	"math/rand"
	"crypto/elliptic"
	"math/big"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
	"github.com/go-ethereum/crypto/sha3"
)

func TestNewTRS(t *testing.T) {
	//curve := edwards.Edwards()
	curve := elliptic.P256()
	alpha := RandFieldElement(curve)
	alpha2 := RandFieldElement(curve)
	sum := new(big.Int).Sub(alpha, alpha2)
	sum.Mod(sum, curve.Params().N)
	//sum := edwards.ScalarAdd(alpha,alpha2)
	//fmt.Printf("%x\n",alpha2)
	//fmt.Printf("%x\n",new(big.Int).Neg(alpha2))
	//sub := edwards.ScalarAdd(alpha,new(big.Int).Neg(alpha2))
	////sum := new(big.Int).Sub(alpha,alpha2)
	//sum.Mod(sum,curve.Params().N)
	rsum, _ := curve.ScalarBaseMult(sum.Bytes())
	fmt.Println(rsum)
	//r1,_:=curve.ScalarBaseMult(sub.Bytes())
	//fmt.Println(r1)

	aa1, aa2 := curve.ScalarBaseMult(alpha.Bytes())
	aa3, aa4 := curve.ScalarBaseMult(new(big.Int).Neg(alpha2).Bytes())
	a1, _ := curve.Add(aa1, aa2, aa3, aa4)
	fmt.Println(a1)
}

func TestSub2(t *testing.T) {
	curve := edwards.Edwards()
	//curve:= elliptic.P256()
	alpha := RandFieldElement(curve)
	alpha2 := RandFieldElement(curve)
	alpha3 := RandFieldElement(curve)
	//sum := new(big.Int).Sub(alpha,alpha2)
	//sum.Mod(sum,curve.Params().N)
	mul := edwards.ScalarMu(alpha2, alpha3)
	sum := edwards.ScalarSub(alpha, mul)
	r1, r2 := curve.ScalarBaseMult(sum.Bytes())
	fmt.Println("%v 5v", r1, curve.IsOnCurve(r1, r2))
	aa1, aa2 := curve.ScalarBaseMult(alpha.Bytes())
	aa3, aa4 := curve.ScalarBaseMult(mul.Bytes())
	a1, a2 := curve.Add(aa1, aa2, aa3, aa4.Neg(aa4))
	fmt.Println(a1)
	fmt.Println("%v 5v", a1, curve.IsOnCurve(a1, a2))
}
func TestSub3(t *testing.T) {
	curve := edwards.Edwards()
	//curve:= elliptic.P256()
	alpha := RandFieldElement(curve)
	alpha2 := RandFieldElement(curve)
	alpha3 := RandFieldElement(curve)
	//sum := new(big.Int).Sub(alpha,alpha2)
	//sum.Mod(sum,curve.Params().N)
	mul := edwards.ScalarMu(alpha2, alpha3)
	sum := edwards.ScalarAdd(alpha, mul.Neg(mul))
	r1, r2 := curve.ScalarBaseMult(sum.Bytes())
	fmt.Println("%v 5v", r1, curve.IsOnCurve(r1, r2))
	aa1, aa2 := curve.ScalarBaseMult(alpha.Bytes())
	aa3, aa4 := curve.ScalarBaseMult(mul.Bytes())
	a1, a2 := curve.Add(aa1, aa2, aa3, aa4)
	fmt.Println(a1)
	fmt.Println("%v 5v", a1, curve.IsOnCurve(a1, a2))
}

func TestSignVerify_LSAG(t *testing.T) {
	randSeed := int64(54321)
	tRand := rand.New(rand.NewSource(randSeed))

	var sig = NewLSAG(nil, elliptic.P256(), sha3.New256())
	message := []byte("hallo world")
	sigpos, privatekey, pkps := PrepareRingSig_random(sig, randSeed, 45)
	sig.Sign(message, pkps, privatekey, sigpos)
	res := sig.Verify(message, pkps)
	if !res {
		t.Error("Verification failed")
		return
	}

	//lets mess with the verification process
	//alter random bit at message
	sigpos, privatekey, pkps = PrepareRingSig_random(sig, randSeed, 45)
	sig.Sign(message, pkps, privatekey, sigpos)

	pos := tRand.Intn(len(message))
	bitPos := tRand.Intn(7)
	message[pos] ^= 1 << uint8(bitPos)
	if sig.Verify(message, pkps) {
		t.Error("Verification expected to fail")
		return
	}

	//alter random bit at public key
	sigpos, privatekey, pkps = PrepareRingSig_random(sig, randSeed, 5)
	sig.Sign(message, pkps, privatekey, sigpos)

	pos = tRand.Intn(len(pkps))
	bitPos = tRand.Intn(7)
	alteredbytes := pkps[pos].Y.Bytes()
	alteredbytes[tRand.Intn(len(alteredbytes))] ^= 1 << uint8(bitPos)
	pkps[pos].Y.SetBytes(alteredbytes)
	if sig.Verify(message, pkps) {
		t.Error("Verification expected to fail")
		return
	}

}

//func TestSetParameters(t *testing.T) {
//
//	publ, priv, _ := edwards.GenerateKeyBytes()
//
//	PRIV, _ := edwards.PrivKeyFromSecret(edwards.Edwards(), priv[:32])
//	p, Pu := edwards.PrivKeyFromBytes(edwards.Edwards(), priv[:])
//	P, _ := edwards.ParsePubKey(edwards.Edwards(), publ[:])
//	x, y := p.Public()
//	fmt.Printf("%v \n %v \n %v %v\n", Pu, P, x, y)
//	fmt.Printf("%v ", PRIV.ToECDSA())
//
//	//X, Y := Curve.ScalarBaseMult(edwards.ComputeScalar(&priv)[:])
//	//fmt.Printf("\nsdf %v %v", X, Y)
//	//
//	X, Y := edwards.Edwards().ScalarBaseMult(p.Serialize())
//	fmt.Printf("\n %v %v", X, Y)
//	//X, Y = Curve.ScalarBaseMult(p.ToECDSA().D.Bytes())
//	//fmt.Printf("\n %v %v", X, Y)
//	//X, Y = Curve.ScalarBaseMult(p.ToECDSA().D.Bytes())
//	//fmt.Printf("\n %v %v", X, Y)
//
//}

func BenchmarkSignVerify_TRS(b *testing.B) {
	//randSeed := int64(54321)
	////tRand := rand.New(rand.NewSource(randSeed))
	//
	//for i := 0; i < 100; i++ {
	//	var sig = RingSignature(NewLSAG(nil, nil))
	//
	//	message := []byte("hallo world")
	//	sigpos, privatekey, pkps := PrepareRingSig_random(sig, randSeed, 7)
	//
	//	sig.Sign(message, pkps, privatekey, sigpos)
	//
	//	res := sig.Verify(message, pkps)
	//	if !res {
	//		b.Error("Verification P256 failed")
	//		return
	//	}
	//	randSeed += 3
	//}

}
