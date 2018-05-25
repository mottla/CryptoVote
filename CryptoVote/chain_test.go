// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"
	//"bytes"
	//"io"
	//"os"
	//"net/http"

	//"flag"
	//"fmt"
	"bytes"
	"io"
	"os"
	"net/http"
	"encoding/json"
	//"fmt"
	"fmt"
)

func TestValidate(t *testing.T) {
	b := new(bytes.Buffer)
	res, _ := http.Post("http://127.0.0.1:3000/validate", "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)
	fmt.Println()
	b = new(bytes.Buffer)
	res, _ = http.Post("http://127.0.0.1:3001/validate", "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)
	//
	fmt.Println(2)
	b = new(bytes.Buffer)
	res, _ = http.Post("http://127.0.0.1:3003/validate", "application/json; charset=utf-8", b)

	io.Copy(os.Stdout, res.Body)
	fmt.Println()

}

func TestAPI_http(t *testing.T) {

	b := new(bytes.Buffer)
	res, _ := http.Post("http://127.0.0.1:3000/blocksH", "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)
	fmt.Println()
	b = new(bytes.Buffer)
	res, _ = http.Post("http://127.0.0.1:3001/blocksH", "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)

	//fmt.Println(2)
	//b = new(bytes.Buffer)
	//res, _ = http.Post("http://127.0.0.1:3003/blocksH", "application/json; charset=utf-8", b)
	//
	//io.Copy(os.Stdout, res.Body)
	fmt.Println()
	//b = new(bytes.Buffer)
	//res, _ = http.Post("http://127.0.0.1:3004/blocksH", "application/json; charset=utf-8", b)
	//
	//io.Copy(os.Stdout, res.Body)
	//fmt.Println()
	//b = new(bytes.Buffer)
	//res, _ = http.Post("http://127.0.0.1:3003/blocksH", "application/json; charset=utf-8", b)
	//io.Copy(os.Stdout, res.Body)

}

func TestGETID(t *testing.T) {
	params := map[string][32]byte{"id": {4}}
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(&params)
	res, err := http.Post("http://127.0.0.1:3000/transactionByID", "application/json; charset=utf-8", b)
	if err != nil {
		t.Error(err)
		return
	}

	var bl = Block{}
	if err := json.NewDecoder(res.Body).Decode(&bl); err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("\n recived: %v", bl)

}

//func newTestBlockchain(blocks Blocks) *Blockchain {
//	return &Blockchain{
//		blocks: blocks,
//		mu:     sync.RWMutex{},
//	}
//}
//
//func TestGenerateBlock(t *testing.T) {
//	bc := newTestBlockchain(Blocks{testGenesisBlock})
//
//	block := bc.generateBlock("white noise")
//	if block.Index != bc.getLatestBlock().Index+1 {
//		t.Errorf("want %d but %d", bc.getLatestBlock().Index+1, block.Index)
//	}
//	if block.Data != "white noise" {
//		t.Errorf("want %q but %q", "white noise", block.Data)
//	}
//	if block.PreviousHash != bc.getLatestBlock().Hash {
//		t.Errorf("want %q but %q", bc.getLatestBlock().Hash, block.PreviousHash)
//	}
//}
//
//func TestAddBlock(t *testing.T) {
//	bc := newTestBlockchain(Blocks{testGenesisBlock})
//	block := &Block{
//		Index:        1,
//		PreviousHash: testGenesisBlock.Hash,
//		Timestamp:    1494177351,
//		Data:         "white noise",
//		Hash:         "1cee23ac6ce3589aedbd92213e0dbf8ab41f8f8e6181a92c1a8243df4b32078b",
//	}
//
//	bc.addBlock(block)
//	if bc.chainHeight() != 2 {
//		t.Fatalf("want %d but %d", 2, bc.chainHeight())
//	}
//	if bc.getLatestBlock().Hash != block.Hash {
//		t.Errorf("want %q but %q", block.Hash, bc.getLatestBlock().Hash)
//	}
//}
//
//func TestReplaceBlocks(t *testing.T) {
//	bc := newTestBlockchain(Blocks{testGenesisBlock})
//	blocks := Blocks{
//		testGenesisBlock,
//		&Block{
//			Index:        1,
//			PreviousHash: testGenesisBlock.Hash,
//			Timestamp:    1494093545,
//			Data:         "white noise",
//			Hash:         "1cee23ac6ce3589aedbd92213e0dbf8ab41f8f8e6181a92c1a8243df4b32078b",
//		},
//	}
//
//	bc.replaceBlocks(blocks)
//	if bc.chainHeight() != 2 {
//		t.Fatalf("want %d but %d", 2, bc.chainHeight())
//	}
//	if bc.getLatestBlock().Hash != blocks[chainHeight(blocks)-1].Hash {
//		t.Errorf("want %q but %q", blocks[chainHeight(blocks)-1].Hash, bc.getLatestBlock().Hash)
//	}
//}
//
//type isValidChainTestCase struct {
//	name       string
//	blockchain *Blockchain
//	ok         bool
//}
//
//var isValidChainTestCases = []isValidChainTestCase{
//	isValidChainTestCase{
//		"empty",
//		newTestBlockchain(Blocks{}),
//		false,
//	},
//	isValidChainTestCase{
//		"invalid genesis block",
//		newTestBlockchain(Blocks{
//			&Block{
//				Index:        0,
//				PreviousHash: "0",
//				Timestamp:    1465154705,
//				Data:         "bad genesis block!!",
//				Hash:         "627ab16dbcede0cfa91c85a88c30c4eaae41b8500a961d0d09451323c6e25bf8",
//			},
//		}),
//		false,
//	},
//	isValidChainTestCase{
//		"invalid block",
//		newTestBlockchain(Blocks{
//			testGenesisBlock,
//			&Block{
//				Index:        2,
//				PreviousHash: testGenesisBlock.Hash,
//				Timestamp:    1494177351,
//				Data:         "white noise",
//				Hash:         "6e27d73b81b2abf47e6766b8aad12a114614fccac669d0d2162cb842f0484420",
//			},
//		}),
//		false,
//	},
//	isValidChainTestCase{
//		"valid",
//		newTestBlockchain(Blocks{
//			testGenesisBlock,
//			&Block{
//				Index:        1,
//				PreviousHash: testGenesisBlock.Hash,
//				Timestamp:    1494177351,
//				Data:         "white noise",
//				Hash:         "1cee23ac6ce3589aedbd92213e0dbf8ab41f8f8e6181a92c1a8243df4b32078b",
//			},
//		}),
//		true,
//	},
//}
//
//func TestIsValidChain(t *testing.T) {
//	for _, testCase := range isValidChainTestCases {
//		if ok := testCase.blockchain.isValid(); ok != testCase.ok {
//			t.Errorf("[%s] want %t but %t", testCase.name, testCase.ok, ok)
//		}
//	}
//}
