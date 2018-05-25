// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"

	"fmt"
	"runtime"
	"bytes"

	"io"
	"os"
	"net/http"
	//"math/big"
	//"github.com/naivechain-master/CryptoNote1/edwards"
	//"math/bits"
	"encoding/json"
	"time"
)

func TestCPUMiner_Start(t *testing.T) {
	//node := newNode()

	// Get a channel that will be closed when a shutdown signal has been
	// triggered either from an OS signal such as SIGINT (Ctrl+C) or from
	// another subsystem such as the RPC server.
	interrupt := interruptListener()
	defer fmt.Printf("Shutdown complete")
	PowProcs := runtime.NumCPU()
	if PowProcs != 7 {
		PowProcs--
	}
	//PowProcs=1
	newNode(*p2pAddr,*apiAddr).miner.Start(0, uint8(PowProcs))

	<-interrupt

}
func TestDifficulty_algo(t *testing.T) {
	//b:=[]byte{0,0,1,2,3}
	//c:=new(big.Int).SetBytes([]byte{1})
	//for i:=0;i<32;i++{
	//	fmt.Println(c.Bit(i))
	//}
	//fmt.Println(uint64(b[1])<<32)
	diff := uint32(24)
	bn := diff / 8
	targetDifficulty := make([]byte, 32)
	targetDifficulty[bn] = (byte(128) >> (diff % 8))
	fmt.Printf("target diff %b", targetDifficulty)

	diff = uint32(3)
	bn = diff / 8
	targetDifficulty2 := make([]byte, 32)
	targetDifficulty2[bn] = (byte(128) >> (diff % 8))
	fmt.Printf("\ntarget diff %b\n", targetDifficulty2)
	fmt.Println(bytes.Compare(targetDifficulty, targetDifficulty2))

}

func TestCPUMiner_Start_HTTP(t *testing.T) {
	params := map[string]uint8{"procs":1, "amount": 0}
	fmt.Println(time.Now().Unix())
	b := new(bytes.Buffer)
	fmt.Println(1.1)
	json.NewEncoder(b).Encode(params)
	fmt.Println(1.3)
	res, _ := http.Post("http://127.0.0.1:3001/startMining", "application/json; charset=utf-8", b)
	fmt.Println(2)
	io.Copy(os.Stdout, res.Body)
	fmt.Println(3)
	//params = map[string]uint8{"procs": 3, "amount": 0}
	//b = new(bytes.Buffer)
	//fmt.Println(4)
	//json.NewEncoder(b).Encode(params)
	//res, _ = http.Post("http://127.0.0.1:3001/startMining", "application/json; charset=utf-8", b)
	//fmt.Println(5)
	//io.Copy(os.Stdout, res.Body)
	//
	//params = map[string]uint8{"procs": 2, "amount": 0}
	//b = new(bytes.Buffer)
	//fmt.Println(4)
	//json.NewEncoder(b).Encode(params)
	//res, _ = http.Post("http://127.0.0.1:3003/startMining", "application/json; charset=utf-8", b)
	//fmt.Println(5)
	//io.Copy(os.Stdout, res.Body)
}

func TestCPUMiner_Stop_HTTP(t *testing.T) {

	b := new(bytes.Buffer)
	res, _ := http.Post("http://127.0.0.1:3001/stopMining", "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)

}
