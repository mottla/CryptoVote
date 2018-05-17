// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/net/websocket"
	"fmt"
	"github.com/naivechain-master/CryptoNote1/edwards"
	"hash"
	"golang.org/x/crypto/sha3"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type Node struct {
	pool             *TxPool
	*http.ServeMux
	blockchain       *Blockchain
	conns            []*Conn
	mu               sync.RWMutex
	logger           *log.Logger
	miner            *CPUMiner
	p2pAddr, apiAddr string
	edcurve          *edwards.TwistedEdwardsCurve
	hasher           hash.Hash
}
type notifier interface {
	submitBlock(block *Block)
}

func (node *Node) submitBlock(block *Block) {
	msg, _, err2 := node.validateChain(Blocks{block})
	if err2 != nil {
		return
	}
	if msg != nil {
		node.broadcast(msg)
	}
}

func newNode(p2pAddr, apiAddr string) *Node {
	bc := newBlockchainGenesis()


	pool := &TxPool{
		mtx:     sync.RWMutex{},
		poolMap: make(map[[64]byte]*transaction),
		logger:  log.New(os.Stdout, fmt.Sprintf("[%v] Pool: ", apiAddr), log.Ldate|log.Ltime, )}

	miner := &CPUMiner{
		pool:              pool,
		Mutex:             sync.Mutex{},
		bc:                bc,
		numWorkers:        0,
		started:           false,
		submitBlockLock:   sync.Mutex{},
		wg:                sync.WaitGroup{},
		workerWg:          sync.WaitGroup{},
		updateNumWorkers:  nil,
		queryHashesPerSec: nil,
		updateHashes:      make(chan uint64),
		speedMonitorQuit:  nil,
		quit:              nil,
		logger: log.New(os.Stdout, fmt.Sprintf("[%v] Miner: ", apiAddr), log.Ldate|log.Ltime,
		),
	}
	n := &Node{
		pool:       pool,
		blockchain: bc,
		miner:      miner,
		conns:      []*Conn{},
		mu:         sync.RWMutex{},
		logger: log.New(
			os.Stdout,
			fmt.Sprintf("[%v] Node: ", apiAddr),
			log.Ldate|log.Ltime,
		),
		p2pAddr: p2pAddr,
		apiAddr: apiAddr,
		edcurve: edwards.Edwards(),
		hasher:  sha3.New256(),
	}
	n.miner.not = notifier(n)
	return n
}

func (node *Node) newApiServer(apiAddr *string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/blocks", node.blocksHandler)
	mux.HandleFunc("/blocksH", node.blocksHandlerH)
	mux.HandleFunc("/addTransaction", node.addTransaction)
	mux.HandleFunc("/peers", node.peersHandler)
	mux.HandleFunc("/addPeer", node.addPeerHandler)
	mux.HandleFunc("/startMining", node.startMining)
	mux.HandleFunc("/stopMining", node.stopMining)
	mux.HandleFunc("/stop", node.stop)
	mux.HandleFunc("/pool", node.showpool)
	mux.HandleFunc("/transactionByID", node.transactionByID)
	mux.HandleFunc("/voteResults", node.voteResults)
	mux.HandleFunc("/validate", node.validate)

	return &http.Server{
		Handler: mux,
		Addr:    *apiAddr,
	}
}

func (node *Node) newP2PServer(p2pAddr *string) *http.Server {
	return &http.Server{
		Handler: websocket.Handler(func(ws *websocket.Conn) {
			conn := newConn(ws)
			node.log("connect to peer:", conn.remoteHost())
			node.addConn(conn)
			node.p2pHandler(conn)
		}),
		Addr: *p2pAddr,
	}
}

func (node *Node) run()  {
	apiSrv := node.newApiServer(&node.apiAddr)

	go func() {
		node.log("start HTTP server for API")
		if err := apiSrv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	p2pSrv := node.newP2PServer(&node.p2pAddr)
	go func() {
		node.log("start WebSocket server for P2P")
		if err := p2pSrv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	signalCh := make(chan os.Signal)
	signal.Notify(signalCh, syscall.SIGTERM)

	for {
		select {
		case s := <-signalCh:
			if s == syscall.SIGTERM {
				node.log("stop servers")
				apiSrv.Shutdown(context.Background())
				p2pSrv.Shutdown(context.Background())
			}
		}

	}
}

func (node *Node) log(v ...interface{}) {
	
	node.logger.Println(v)
}

func (node *Node) logError(err error) {
	node.log("[ERROR]", err)
}

func (node *Node) writeResponse(w http.ResponseWriter, b []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

func (node *Node) error(w http.ResponseWriter, err error, message string) {
	node.logError(err)

	b, err := json.Marshal(&ErrorResponse{
		Error: message,
	})
	if err != nil {
		node.logError(err)
	}

	node.writeResponse(w, b)
}
