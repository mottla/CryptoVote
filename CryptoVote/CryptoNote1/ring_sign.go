// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package CryptoNote1

import (
	"fmt"
	"math/big"
	"io"
	"crypto/elliptic"
	"crypto/ecdsa"
	"crypto/rand"
	mr "math/rand"
	"encoding/binary"
	"bytes"
	"hash"

	"golang.org/x/crypto/sha3"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
	"sync"
)

var defaultCurve = edwards.Edwards()
var defaultHasher = sha3.New256()

//returns a new Linkable Spontaneous Anonymous Group Signature Struct.
//if key is set, ls can be used for signing and verification. If not, ls can be used for verification only
//default curve and hash can be replaced. Depricated to do so in the main protocol.
func NewLSAG(key *ecdsa.PrivateKey, _curve elliptic.Curve, _hash hash.Hash) (ls *LSAGsignature) {
	ls = new(LSAGsignature)

	if _curve == nil {
		_curve = defaultCurve
	}
	ls.Curve = _curve

	if _hash == nil {
		_hash = defaultHasher
	} else if _hash.Size() != 32 {
		panic("only hashes with outputsize of 32 bytes allowed!", )
	}

	ls.hasher = _hash
	if key != nil {
		ls.AddKeyImage(key)
	}
	return
}
func NewTRS(_curve elliptic.Curve, _hash hash.Hash) (ls *TRSsignature) {
	ls = new(TRSsignature)

	if _curve == nil {
		_curve = edwards.Edwards()
	}
	ls.Curve = _curve

	if _hash == nil {
		_hash = sha3.New256()
	} else if _hash.Size() != 32 {
		panic("only hashes with outputsize of 32 bytes allowed!", )
	}

	ls.hasher = _hash
	return
}

type RingSignature interface {
	Sign(message []byte, pubKeys []*ecdsa.PublicKey, signerKey *ecdsa.PrivateKey, signerPosition int)
	Verify(message []byte, pubKeys []*ecdsa.PublicKey) bool
	BytesToBuffer(buf *bytes.Buffer)
	GetCurve() elliptic.Curve
	getHasher() hash.Hash
	hashToEcc(x, y *big.Int) (hx, hy *big.Int)
	AddKeyImage(priv *ecdsa.PrivateKey)
}

func (s *LSAGsignature) GetCurve() elliptic.Curve {
	return s.Curve
}
func (s *LSAGsignature) getHasher() hash.Hash {
	return s.hasher
}
func (s *TRSsignature) GetCurve() elliptic.Curve {
	return s.Curve
}
func (s *TRSsignature) getHasher() hash.Hash {
	return s.hasher
}

type LSAGsignature struct {
	Sigma
	elliptic.Curve
	hasher hash.Hash
}
type TRSsignature struct {
	Sigma
	elliptic.Curve
	hasher hash.Hash
}

//create a Linkable Spontaneous Anonymous Groups Signature
//asserts that the calling Sigma was prepared properly by calling PrepareRingSig_random(..) before
//produces a ring signature on a given Curve for a given message as input using the scheme in
//https://bitcointalk.org/index.php?topic=972541.msg10619684#msg10619684
//note that len(Sigma.Ci) =1 != len(Sigma.Ri)=len(pubKeys), and therefore the signature requires approximately
//half the storage as a Traceable Ring Signature produced by SignTRS(...)
func (s *LSAGsignature) Sign(message []byte, pubKeys []*ecdsa.PublicKey, signerKey *ecdsa.PrivateKey, signerPosition int) {

	s.Ri = make([]*big.Int, len(pubKeys))
	var Lix, Liy *big.Int
	var Rix, Riy *big.Int
	var hash []byte
	var x1, x2, x4, x3 *big.Int
	var t1, t2 *big.Int
	Curve := s.Curve
	hasher := s.hasher
	j := int(signerPosition+1) % len(pubKeys)
	alpha := RandFieldElement(Curve)
	//fmt.Printf("alpha %x", alpha)
	//fmt.Printf("\npubkey %x", pubKeys[signerPosition].X)
	Lix, Liy = Curve.ScalarBaseMult(alpha.Bytes())
	t1, t2 = s.hashToEcc(pubKeys[signerPosition].X, pubKeys[signerPosition].Y)
	//fmt.Printf("\n### Hpx %v ", t1)
	Rix, Riy = Curve.ScalarMult(t1, t2, alpha.Bytes())
	//fmt.Printf("\nLi %x %v", Lix, Curve.IsOnCurve(Lix, Liy))
	//fmt.Printf("\nRi %x %v", Rix, Curve.IsOnCurve(Rix, Riy))
	hasher.Reset()
	hasher.Write(message)
	hasher.Write(Lix.Bytes())
	hasher.Write(Liy.Bytes())
	hasher.Write(Rix.Bytes())
	hasher.Write(Riy.Bytes())
	hash = hasher.Sum(nil)

	if j == 0 {
		s.Ci = []*big.Int{new(big.Int).SetBytes(hash)}
	}

	for counter := 0; counter < len(pubKeys)-1; counter++ {
		//fmt.Println("asdgasljga#################")
		s.Ri[j] = RandFieldElement(Curve)
		x1, x2 = Curve.ScalarBaseMult(s.Ri[j].Bytes())
		x3, x4 = Curve.ScalarMult(pubKeys[j].X, pubKeys[j].Y, hash)
		Lix, Liy = Curve.Add(x1, x2, x3, x4)

		t1, t2 = s.hashToEcc(pubKeys[j].X, pubKeys[j].Y)
		x1, x2 = Curve.ScalarMult(t1, t2, s.Ri[j].Bytes())
		x3, x4 = Curve.ScalarMult(s.Ix, s.Iy, hash)
		Rix, Riy = Curve.Add(x1, x2, x3, x4)
		hasher.Reset()
		hasher.Write(message)
		hasher.Write(Lix.Bytes())
		hasher.Write(Liy.Bytes())
		hasher.Write(Rix.Bytes())
		fmt.Println(Rix.Bytes())
		hasher.Write(Riy.Bytes())
		hash = hasher.Sum(nil)
		j++
		j %= len(pubKeys)
		if j == 0 {
			s.Ci = []*big.Int{new(big.Int).SetBytes(hash)}
		}
	}

	s.Ri[j] = new(big.Int)

	//b:=new(big.Int).Mul(new(big.Int).SetBytes(hash), signerKey.GetD())
	//edwards.ScalarAdd()
	//s.Ri[j] = edwards.ScalarSub(alpha,b)
	s.Ri[j].Sub(alpha, new(big.Int).Mul(new(big.Int).SetBytes(hash), signerKey.D))
	s.Ri[j].Mod(s.Ri[j], Curve.Params().N)

	//tt3, tt4 := Curve.ScalarMult(s.Ix, s.Iy, s.Ci[j].Bytes())
	//fmt.Printf("\nRi step 1 IX %x and ci %x  on curve %v", s.Ix, s.Ci, Curve.IsOnCurve(tt3, tt4))
	//tt1, tt2 := Curve.ScalarMult(t1, t2, s.Ri[j].Bytes())
	//fmt.Printf("\n H(P).x %v on curve %v sj bytes %v", tt1, Curve.IsOnCurve(tt1, tt2),s.Ri[j].Bytes())
	////edwards.FieldElementToBigInt()
	//Rixtest, Riytest := Curve.Add(tt1, tt2, tt3, tt4)
	//
	//Rresx, _ := Curve.ScalarMult(t1, t2, alpha.Bytes())
	//fmt.Printf("\nRi restored %x --- original %x %v", Rresx, Rixtest, Curve.IsOnCurve(Rixtest, Riytest))

}

//checks weather the given LSAG signature is valid according to verification rules in MRL-0005
//returns true if valid, false otherwise
func (s *LSAGsignature) Verify(message []byte, pubKeys []*ecdsa.PublicKey) bool {
	if len(s.Ci) != 1 || len(s.Ri) != len(pubKeys) {
		fmt.Println("improper prepared signature")
		return false
	}
	Curve := s.Curve

	hasher := s.hasher
	ci := make([][]byte, len(pubKeys))
	ci[0] = s.Ci[0].Bytes()
	var Lix, Liy *big.Int
	var Rix, Riy *big.Int
	var x1, x2, x4, x3 *big.Int
	var t1, t2 *big.Int

	for i, sj := range s.Ri {
		x1, x2 = Curve.ScalarBaseMult(sj.Bytes())
		x3, x4 = Curve.ScalarMult(pubKeys[i].X, pubKeys[i].Y, ci[i])
		Lix, Liy = Curve.Add(x1, x2, x3, x4)
		t1, t2 = s.hashToEcc(pubKeys[i].X, pubKeys[i].Y)
		x3, x4 = Curve.ScalarMult(s.Ix, s.Iy, ci[i])
		x1, x2 = Curve.ScalarMult(t1, t2, sj.Bytes())
		Rix, Riy = Curve.Add(x1, x2, x3, x4)
		hasher.Reset()
		hasher.Write(message)
		hasher.Write(Lix.Bytes())
		hasher.Write(Liy.Bytes())
		hasher.Write(Rix.Bytes())
		hasher.Write(Riy.Bytes())
		ci[(i+1)%len(pubKeys)] = hasher.Sum(nil)
	}
	if bytes.Equal(ci[0], s.Ci[0].Bytes()) {
		return true
	}
	return false

}

//create a Traceable Ring Signature as proposed in the CryptoNote Whitepaper
//asserts that the calling Sigma was prepared properly by calling PrepareRingSig_random(..)
//produces a ring signature on a given Curve for a given message as input using the scheme in
//note that len(Sigma.Ci) = len(Sigma.Ri)=len(pubKeys), and therefore the signature requires approximately
//twice the storage as a Linkable Spontaneous Anonymous Group Signature  produced by SignLSAG(...)
func (s *TRSsignature) Sign(message []byte, pubKeys []*ecdsa.PublicKey, signerKey *ecdsa.PrivateKey, signerPosition int) {
	//n:=len(pkps)
	keySet := pubKeys
	Li := make([]bigPair, len(keySet))
	Ri := make([]bigPair, len(keySet))
	qi := make([]*big.Int, len(keySet))
	wi := make([]*big.Int, len(keySet))
	Curve := s.Curve
	hasher := s.hasher
	var wSum = new(big.Int)
	hasher.Reset()
	hasher.Write(message)
	//wg := new(sync.WaitGroup)
	for i := 0; i < len(keySet); i++ {
		//wg.Add(1)
		//go func(i int) {
			qi[i] = RandFieldElement(s.Curve)

			var x1, x2, x4, x3 *big.Int
			var t1, t2 *big.Int
			if i != int(signerPosition) {
				wi[i] = RandFieldElement(s.Curve)
				x1, x2 = Curve.ScalarBaseMult(qi[i].Bytes())
				x3, x4 = Curve.ScalarMult(keySet[i].X, keySet[i].Y, wi[i].Bytes())
				Li[i].a, Li[i].b = Curve.Add(x1, x2, x3, x4)
				t1, t2 = s.hashToEcc(keySet[i].X, keySet[i].Y)
				x1, x2 = Curve.ScalarMult(t1, t2, qi[i].Bytes())
				x3, x4 = Curve.ScalarMult(s.Ix, s.Iy, wi[i].Bytes())
				Ri[i].a, Ri[i].b = Curve.Add(x1, x2, x3, x4)
			} else { //at i==signerPosition
				Li[i].a, Li[i].b = Curve.ScalarBaseMult(qi[i].Bytes())
				t1, t2 = s.hashToEcc(keySet[i].X, keySet[i].Y)
				Ri[i].a, Ri[i].b = Curve.ScalarMult(t1, t2, qi[i].Bytes())
			}
		//	wg.Done()
		//}(i)

	}
	wi[signerPosition] = new(big.Int)
	//wg.Wait()
	for i, l := range Li {
		wSum.Add(wSum, wi[i])
		wSum.Mod(wSum, Curve.Params().N)
		hasher.Write(l.a.Bytes())
		hasher.Write(l.b.Bytes())
	}
	for _, r := range Ri {
		hasher.Write(r.a.Bytes())
		hasher.Write(r.b.Bytes())
	}
	hashsum := hasher.Sum(nil)

	wi[signerPosition].Sub(new(big.Int).SetBytes(hashsum[:]), wSum)
	wi[signerPosition].Mod(wi[signerPosition], Curve.Params().N)

	s.Ci = wi

	qi[signerPosition].Sub(qi[signerPosition], new(big.Int).Mul(wi[signerPosition], signerKey.D))
	qi[signerPosition].Mod(qi[signerPosition], Curve.Params().N)
	s.Ri = qi
}

//checks weather the given signature is valid according to verification rules in CryptoNote 2.0 VER
//returns true if valid, false otherwise
func (s *TRSsignature) Verify(message []byte, pubKeys []*ecdsa.PublicKey) bool {
	if len(s.Ci) == 0 || !(len(s.Ci) == len(s.Ri) && len(s.Ci) == len(pubKeys)) {
		return false
	}
	Curve := s.Curve
	hasher := s.hasher
	Li := make([]bigPair, len(s.Ci))
	Ri := make([]bigPair, len(s.Ci))
	var x1, x2, x4, x3 *big.Int
	var t1, t2 *big.Int
	hasher.Reset()
	hasher.Write(message)
	var cSum = new(big.Int)
	for i, r := range s.Ri {
		cSum.Add(cSum, s.Ci[i])
		cSum.Mod(cSum, Curve.Params().N)
		x1, x2 = Curve.ScalarBaseMult(r.Bytes())
		x3, x4 = Curve.ScalarMult(pubKeys[i].X, pubKeys[i].Y, s.Ci[i].Bytes())
		Li[i].a, Li[i].b = Curve.Add(x1, x2, x3, x4)
		t1, t2 = s.hashToEcc(pubKeys[i].X, pubKeys[i].Y)
		x1, x2 = Curve.ScalarMult(t1, t2, r.Bytes())
		x3, x4 = Curve.ScalarMult(s.Ix, s.Iy, s.Ci[i].Bytes())
		Ri[i].a, Ri[i].b = Curve.Add(x1, x2, x3, x4)
	}
	for _, l := range Li {
		hasher.Write(l.a.Bytes())
		hasher.Write(l.b.Bytes())
	}
	for _, r := range Ri {
		hasher.Write(r.a.Bytes())
		hasher.Write(r.b.Bytes())
	}

	hashsum := hasher.Sum(nil)
	if cSum.Mod(cSum, Curve.Params().N).Cmp(new(big.Int).Mod(new(big.Int).SetBytes(hashsum), Curve.Params().N)) == 0 {
		return true
	}
	return false

}

type bigPair struct {
	a, b *big.Int
}

type publicKeyPair struct {
	Ax, Ay, Bx, By *big.Int
}
type Sigma struct {
	Ix, Iy *big.Int //keyimage I=xH(P)
	//position of signer Public Key in the pubKeys Array. Should be assigned randomly.
	Ci []*big.Int //commitment for TRS. In LSAG only len(Ci)=1
	Ri []*big.Int //commitment for LSAG and TRS
}

func (s TRSsignature) String() string {
	return fmt.Sprintf("\nKeyImage[%x,%x]\nCi %v\nRi%v \n", s.Ix, s.Iy, s.Ci, s.Ri)
}
func (s LSAGsignature) String() string {
	return fmt.Sprintf("\nKeyImage[%x,%x]\nCi %v\nRi%v \n", s.Ix, s.Iy, s.Ci, s.Ri)
}

func (s Sigma) String() string {
	return fmt.Sprintf("\nKeyImage[%x,%x]\nCi %v\nRi%v \n", s.Ix, s.Iy, s.Ci, s.Ri)
}

func (s *TRSsignature) BytesToBuffer(buf *bytes.Buffer) {
	//TODO instead of Ix and Iy i could use a 32byte representation
	buf.Write(s.Ix.Bytes())
	buf.Write(s.Iy.Bytes())
	for i, _ := range s.Ci {
		buf.Write(s.Ci[i].Bytes())
	}
	for i, _ := range s.Ri {
		buf.Write(s.Ri[i].Bytes())
	}
}

func (s *LSAGsignature) BytesToBuffer(buf *bytes.Buffer) {
	//TODO instead of Ix and Iy i could use a 32byte representation
	buf.Write(s.Ix.Bytes())
	buf.Write(s.Iy.Bytes())
	for i, _ := range s.Ci {
		buf.Write(s.Ci[i].Bytes())
	}
	for i, _ := range s.Ri {
		buf.Write(s.Ri[i].Bytes())
	}
}

func (s *Sigma) BytesToBuffer(buf *bytes.Buffer) {
	//TODO instead of Ix and Iy i could use a 32byte representation
	buf.Write(s.Ix.Bytes())
	buf.Write(s.Iy.Bytes())
	for i, _ := range s.Ci {
		buf.Write(s.Ci[i].Bytes())
	}
	for i, _ := range s.Ri {
		buf.Write(s.Ri[i].Bytes())
	}
}

type user struct {
	A, B *ecdsa.PrivateKey
	pkP  publicKeyPair
}

func (u *user) randomInit(c elliptic.Curve) (i *user) {
	pA, err := ecdsa.GenerateKey(c, rand.Reader)

	if err != nil {
		panic("rand user generation failed")
	}
	pB, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		panic("rand user generation failed")
	}

	u.A = pA
	u.B = pB
	u.pkP.Ax = pA.X
	u.pkP.Ay = pA.Y
	u.pkP.Bx = pB.X
	u.pkP.By = pB.Y
	return u
}

var one = new(big.Int).SetInt64(1)
// RandFieldElement returns a random element of the field underlying the given
// Curve using the procedure given in [NSA] A.2.1.
func RandFieldElement(c elliptic.Curve) (k *big.Int) {
	params := c.Params()
	b := make([]byte, params.BitSize/8+8)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		panic("random failed..")
	}
	b[0] &= 248
	b[31] &= 127
	b[31] |= 64

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(params.N, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

//Used as Hp in Whitepaper.
//Maps Field element to a Field element of the corresponding Curve.
//returns a deterministic random point on the Curve
//not that its unnecessary to use a cryptographic hash function here.. though why not^^
func (s *TRSsignature) hashToEcc(x, y *big.Int) (hx, hy *big.Int) {
	s.hasher.Reset()

	s.hasher.Write(x.Bytes())
	s.hasher.Write(y.Bytes())
	hx, hy = s.Curve.ScalarBaseMult(s.hasher.Sum(nil)[:])
	return

}
func (s *LSAGsignature) hashToEcc(x, y *big.Int) (hx, hy *big.Int) {
	s.hasher.Reset()
	s.hasher.Write(x.Bytes())
	s.hasher.Write(y.Bytes())
	hx, hy = s.Curve.ScalarBaseMult(s.hasher.Sum(nil)[:])
	return

}

//Initializes the Key Image which is necessary for signing, validating and the prevention of doublespending after all
func (s *TRSsignature) AddKeyImage(priv *ecdsa.PrivateKey) {

	hx, hp := s.hashToEcc(priv.X, priv.Y)

	//TODO check if priv.getD == priv.serialize
	s.Ix, s.Iy = s.Curve.ScalarMult(hx, hp, priv.D.Bytes())

	return

}
func (s *LSAGsignature) AddKeyImage(priv *ecdsa.PrivateKey) {
	hx, hp := s.hashToEcc(priv.X, priv.Y)

	//TODO check if priv.getD == priv.serialize
	s.Ix, s.Iy = s.Curve.ScalarMult(hx, hp, priv.D.Bytes())

	return
}

// creates a signature template with random cosigners and random signer position
// usefull for testing
func PrepareRingSig_random(in RingSignature, randSeed int64, cosigners int) (signerPosition int, signerPrivatekey *ecdsa.PrivateKey, pkps []*ecdsa.PublicKey) {
	b := make([]byte, 4)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		panic("rand failed")
	}
	signerPosition = int(binary.BigEndian.Uint32(b))
	if cosigners == 0 {
		signerPosition = 0
	} else {
		signerPosition %= (cosigners + 1)
	}
	privs, pkps := randKeySet(in.GetCurve(), 1, cosigners+1)
	//privs, pkps := randPrivScalarKeyList2(edwards.Edwards(), cosigners+1)
	signerPrivatekey = privs[signerPosition]
	in.AddKeyImage(signerPrivatekey)
	return signerPosition, signerPrivatekey, pkps
}

//returns a set of deterministic-randomely generated public and corresponding privatekeys
func randKeySet(curve elliptic.Curve, randSeed int64, i int) (privates []*ecdsa.PrivateKey, publics []*ecdsa.PublicKey) {
	//r := mr.New(mr.NewSource(randSeed))

	privates = make([]*ecdsa.PrivateKey, i)
	publics = make([]*ecdsa.PublicKey, i)
	for j := 0; j < i; j++ {
		rand := RandFieldElement(curve)
		x, y := curve.ScalarBaseMult(rand.Bytes())
		publics[j] = &ecdsa.PublicKey{Y: y, X: x}
		privates[j] = &ecdsa.PrivateKey{D: rand, PublicKey: *publics[j]}
	}
	return
}

func randPrivScalarKeyList2(curve *edwards.TwistedEdwardsCurve, i int) (privs []*ecdsa.PrivateKey, publs []*ecdsa.PublicKey) {
	r := mr.New(mr.NewSource(754321))

	privKeyList := make([]*ecdsa.PrivateKey, i)
	pubKeyList := make([]*ecdsa.PublicKey, i)
	for j := 0; j < i; j++ {
		for {
			bIn := new([32]byte)
			for k := 0; k < edwards.PrivScalarSize; k++ {
				randByte := r.Intn(255)
				bIn[k] = uint8(randByte)
			}

			bInBig := new(big.Int).SetBytes(bIn[:])
			bInBig.Mod(bInBig, curve.N)
			bIn = copyBytes(bInBig.Bytes())
			bIn[31] &= 248

			pks, pubs, err := edwards.PrivKeyFromScalar(curve, bIn[:])
			if err != nil {
				r.Seed(int64(j) + r.Int63n(12345))
				continue
			}

			//// No duplicates allowed.
			//if j > 0 &&
			//	(bytes.Equal(pks.Serialize(), privKeyList[j-1].Serialize())) {
			//	continue
			//}

			privKeyList[j] = pks.ToECDSA()
			pubKeyList[j] = pubs.ToECDSA()
			fmt.Printf("\nrand generate on curve %v \n", curve.IsOnCurve(pubs.X, pubs.Y))
			r.Seed(int64(j) + 54321)
			break
		}
	}

	return privKeyList, pubKeyList
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

//// GenerateKey generates a public/private key pair using entropy from rand.
//// If rand is nil, crypto/rand.Reader will be used.
//func GenerateKey(rand io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
//	if rand == nil {
//		rand = cryptorand.Reader
//	}
//
//	privateKey = make([]byte, PrivateKeySize)
//	publicKey = make([]byte, PublicKeySize)
//	_, err = io.ReadFull(rand, privateKey[:32])
//	if err != nil {
//		return nil, nil, err
//	}
//
//	digest := sha512.Sum512(privateKey[:32])
//	digest[0] &= 248
//	digest[31] &= 127
//	digest[31] |= 64
//
//	var A edwards25519.ExtendedGroupElement
//	var hBytes [32]byte
//	copy(hBytes[:], digest[:])
//	edwards25519.GeScalarMultBase(&A, &hBytes)
//	var publicKeyBytes [32]byte
//	A.ToBytes(&publicKeyBytes)
//
//	copy(privateKey[32:], publicKeyBytes[:])
//	copy(publicKey, publicKeyBytes[:])
//
//	return publicKey, privateKey, nil
//}
