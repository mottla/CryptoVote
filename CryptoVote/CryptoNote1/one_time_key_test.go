// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package CryptoNote1

import (
	"testing"
	"fmt"
	"time"

	"golang.org/x/crypto/sha3"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"
)

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf("%s took %dms", name, elapsed.Nanoseconds()/1000000)
}

func TestGenerateOneTime(t *testing.T) {


	defer timeTrack(time.Now(), "Verification")

	ha := sha3.New256()
	cu := edwards.Edwards()
	var Bob = new(user).randomInit(cu)
	Px, Py, Rx, Ry := GenerateOneTimePK(Bob.pkP,ha,cu)
	//fmt.Printf("\nstealth address (%v and %v) ", Px, Py)
	res, _ := Bob.VerifyOneTimePK(Px, Py, Rx, Ry,ha,cu)

	fmt.Printf("Verification %v, ", res)

}


