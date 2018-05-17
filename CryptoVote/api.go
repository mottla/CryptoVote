// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"net/http"
	"golang.org/x/net/websocket"
	"fmt"
)

//returns the entire ledger
func (n *Node) validate(w http.ResponseWriter, r *http.Request) {

	b, err := json.Marshal("all fine")
	blocks := n.blockchain.blocks
	for i := 0; i < len(blocks)-1; i++ {
		if err := blocks[i+1].isValidAncestor(blocks[i]); err != nil {

			b, err = json.Marshal(fmt.Sprintf("invalid ancestor at height", blocks[i+1].Index, err.Error()))
			break
		}
		if !blocks[0].Data.isNil() {
			if err := n.validateTransaction(&blocks[i].Data, nil); err != nil {

				b, err = json.Marshal(fmt.Sprintf("invalid transactions in chain at ", i, err.Error()))
				break
			}
		}

	}

	if err != nil {
		n.error(w, err, "failed to decode response")
		return
	}

	n.writeResponse(w, b)
}

//returns the entire ledger
func (node *Node) blocksHandler(w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(node.blockchain.blocks)
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}

	node.writeResponse(w, b)
}

//searches the blockchain for after a passed transactionByID ID and returns the block that holds it, if contained
func (node *Node) transactionByID(w http.ResponseWriter, r *http.Request) {

	byteMap := make(map[string][32]byte)

	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&byteMap); err != nil {
		node.error(w, err, "failed to decode params")
		return
	}
	block, ok := node.blockchain.BlockHoldingTx(byteMap["id"])
	fmt.Println("received ", byteMap["id"])
	b, err := json.Marshal("transactionByID not found")
	if ok {
		b, err = json.Marshal(block)
	}

	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}
	node.writeResponse(w, b)
}

func (node *Node) showpool(w http.ResponseWriter, r *http.Request) {
	var re = make(map[int]string)
	c := 0
	for _, v := range node.pool.poolMap {
		re[c] = fmt.Sprintf("%x", v.EdSignature)
		c++
	}

	b, err := json.Marshal(re)
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}

	node.writeResponse(w, b)
}

func (node *Node) voteResults(w http.ResponseWriter, r *http.Request) {
	//allows user to ask for results by ether passing the Voting-Contract ID
	//or the block-hash holding the Contract
	byteMap := map[string][32]byte{}

	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&byteMap); err != nil {
		node.error(w, err, "failed to decode params")
		return
	}

	var vote *Voting
	var resMap = make(map[string]uint)
	var err error

	if byteMap["id"] != [32]byte{} {
		bl, ok := node.blockchain.BlockHoldingTx(byteMap["id"])
		if ok {
			vote = node.blockchain.votingMap[bl.Hash]
		}
	}
	if vote != nil {
		resMap, err = node.voteresults(vote)
		if err != nil {
			node.error(w, err, "failed to count voteresult")
			return
		}
	}

	b, err := json.Marshal(resMap)
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}
	node.writeResponse(w, b)
}

//returns the index with corresponding hash of each block in descending order
func (node *Node) blocksHandlerH(w http.ResponseWriter, r *http.Request) {
	var re = make([]string, 0)

	for _, b := range node.blockchain.blocks {
		//fmt.Printf("\nType %v, Hash %x... ", b.Index, b.Hash[:8])
		re = append(re, fmt.Sprintf("%v Type:%v H:%x... ", b.Index, b.Data.Typ, b.Hash[:5]))

	}

	b, err := json.Marshal(re)
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}

	node.writeResponse(w, b)
}

func (node *Node) addTransaction(w http.ResponseWriter, r *http.Request) {

	var t transaction
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		node.error(w, err, "failed to decode params")
		return
	}
	node.log("received transaction: %v", t)

	if node.pool.contains(&t) {
		err := errorCall("Transaction with this ID is already in pool")
		node.error(w, err, err.Error())
		return
	}
	err := node.validateTransaction(&t,nil)
	if err != nil {
		node.error(w, err, err.Error())
		return
	}
	success := node.pool.addTransaction(&t)

	if !success {
		node.error(w, err, err.Error())
		return
	}
	msg, err := node.newTxMessage(t)
	if err != nil {
		node.error(w, err, "failed to build message")
		return
	}

	node.broadcast(msg)
	b, err := json.Marshal("Transaction approved. Wait for block inclusion now!")
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}

	node.writeResponse(w, b)
}

func (node *Node) startMining(w http.ResponseWriter, r *http.Request) {

	priceMap := make(map[string]uint8)

	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&priceMap); err != nil {
		node.error(w, err, "failed to decode params")
		return
	}
	fmt.Println(priceMap)
	var b []byte
	var err error
	if !node.miner.IsMining() {
		node.miner.Start(priceMap["amount"], priceMap["procs"])
		b, err = json.Marshal(fmt.Sprintf("miner started with %v processors for the next %v blocks", priceMap["procs"], priceMap["amount"]))

	} else {
		b, err = json.Marshal("Miner already running")
	}
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}
	node.writeResponse(w, b)

}

func (node *Node) stop(w http.ResponseWriter, r *http.Request) {
	node.log("stop servers not implementet")
	//node.miner.quit <- struct{}{};
	//node.
	//apiSrv.Shutdown(context.Background())
	//p2pSrv.Shutdown(context.Background())
	//b, err := json.Marshal(map[string]string{
	//	"hash": "asdfasdf",
	//})
	//if err != nil {
	//	node.error(w, err, "failed to decode response")
	//	return
	//}
	//fmt.Print(os.Stdout)
	//
	//node.writeResponse(w, b)

}

func (node *Node) stopMining(w http.ResponseWriter, r *http.Request) {

	if node.miner.IsMining() {
		node.miner.quit <- struct{}{};
	}

	b, err := json.Marshal("miner stoped")
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}
	//fmt.Print(os.Stdout)

	node.writeResponse(w, b)

}

func (node *Node) peersHandler(w http.ResponseWriter, r *http.Request) {
	peerHosts := make([]string, len(node.conns))
	for i, conn := range node.conns {
		peerHosts[i] = conn.remoteHost()
	}

	b, err := json.Marshal(peerHosts)
	if err != nil {
		node.error(w, err, "failed to decode response")
		return
	}

	node.writeResponse(w, b)
}

func (node *Node) addPeerHandler(w http.ResponseWriter, r *http.Request) {

	var params struct {
		Peer string `json:"peer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		node.error(w, err, "failed to decode params")
		return
	}

	ws, err := websocket.Dial(params.Peer, "", *p2pOrigin)
	if err != nil {
		node.error(w, err, "failed to connect to peer")
		return
	}

	conn := newConn(ws)
	node.log("connect to peer:", conn.remoteHost())
	node.addConn(conn)
	go node.p2pHandler(conn)

	msg := newQueryAllMessage(node.blockchain.chainHeight(), 0)

	if err := node.send(conn, msg); err != nil {
		node.logError(err)
	}

	//if err := node.send(conn, newQueryLatestMessage()); err != nil {
	//	node.logError(err)
	//}

	node.peersHandler(w, r)
}
