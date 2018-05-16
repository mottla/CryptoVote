package CrptoNote1

import (
	"testing"
	"fmt"

	"golang.org/x/crypto/sha3"
	"time"
	//"math/big"
	"github.com/naivechain-master/CryptoNote1/edwards"
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


