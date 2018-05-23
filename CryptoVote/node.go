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
	"hash"
	"golang.org/x/crypto/sha3"
	"github.com/CryptoVote/CryptoVote/database"
	"github.com/CryptoVote/CryptoVote/CryptoNote1/edwards"

	"path/filepath"
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

	cfg = &config{
		TestNet3:             true,
		//ConfigFile:           defaultConfigFile,
		//DebugLevel:           defaultLogLevel,
		//MaxPeers:             defaultMaxPeers,
		//BanDuration:          defaultBanDuration,
		//BanThreshold:         defaultBanThreshold,
		//RPCMaxClients:        defaultMaxRPCClients,
		//RPCMaxWebsockets:     defaultMaxRPCWebsockets,
		//RPCMaxConcurrentReqs: defaultMaxRPCConcurrentReqs,
		//DataDir:              defaultDataDir,
		//LogDir:               defaultLogDir,
		DbType:               "ffldb",
		//RPCKey:               defaultRPCKeyFile,
		//RPCCert:              defaultRPCCertFile,
		//MinRelayTxFee:        mempool.DefaultMinRelayTxFee.ToBTC(),
		//FreeTxRelayLimit:     defaultFreeTxRelayLimit,
		//BlockMinSize:         defaultBlockMinSize,
		//BlockMaxSize:         defaultBlockMaxSize,
		//BlockMinWeight:       defaultBlockMinWeight,
		//BlockMaxWeight:       defaultBlockMaxWeight,
		//BlockPrioritySize:    mempool.DefaultBlockPrioritySize,
		//MaxOrphanTxs:         defaultMaxOrphanTransactions,
		//SigCacheMaxSize:      defaultSigCacheMaxSize,
		//Generate:             defaultGenerate,
		//TxIndex:              defaultTxIndex,
		//AddrIndex:            defaultAddrIndex,
	}



	Nodelog := log.New(
		os.Stdout,
		fmt.Sprintf("[%v] Node: ", apiAddr),
		log.Ldate|log.Ltime,
	)

	bc := newBlockchainGenesis()

	//// Load the block database.
	//db, err := loadBlockDB()
	//if err != nil {
	//	Nodelog.Fatalf("%v", err)
	//	return nil
	//}
	//defer func() {
	//	// Ensure the database is sync'd and closed on shutdown.
	//	Nodelog.Println("Gracefully shutting down the database...")
	//	db.Close()
	//}()

	pool := newTxPool(apiAddr)

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
		logger:     Nodelog,
		p2pAddr:    p2pAddr,
		apiAddr:    apiAddr,
		edcurve:    edwards.Edwards(),
		hasher:     sha3.New256(),
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

func (node *Node) run() {
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

// loadBlockDB loads (or creates when needed) the block database taking into
// account the selected database backend and returns a handle to it.  It also
// contains additional logic such warning the user if there are multiple
// databases which consume space on the file system and ensuring the regression
// test database is clean when in regression test mode.
func loadBlockDB() (database.DB, error) {
	// The memdb backend does not have a file path associated with it, so
	// handle it uniquely.  We also don't want to worry about the multiple
	// database type warnings when running with the memory database.
	if cfg.DbType == "memdb" {
		fmt.Println("Creating block database in memory.")
		db, err := database.Create("ffldb")
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	warnMultipleDBs()

	// The database name is based on the database type.
	dbPath := blockDbPath(cfg.DbType)

	// The regression test is special in that it needs a clean database for
	// each run, so remove it now if it already exists.
	removeRegressionDB(dbPath)

	fmt.Printf("\nLoading block database from '%s'", dbPath)
	db, err := database.Open(cfg.DbType, dbPath, 0xff)
	if err != nil {
		// Return the error if it's not because the database doesn't
		// exist.
		if dbErr, ok := err.(database.Error); !ok || dbErr.ErrorCode !=
			database.ErrDbDoesNotExist {

			return nil, err
		}

		// Create the db if it does not exist.
		err = os.MkdirAll(cfg.DataDir, 0700)
		if err != nil {
			return nil, err
		}
		db, err = database.Create(cfg.DbType, dbPath, 0xff)
		if err != nil {
			return nil, err
		}
	}

	fmt.Println("Block database loaded")
	return db, nil
}

// warnMultipleDBs shows a warning if multiple block database types are detected.
// This is not a situation most users want.  It is handy for development however
// to support multiple side-by-side databases.
func warnMultipleDBs() {
	// This is intentionally not using the known db types which depend
	// on the database types compiled into the binary since we want to
	// detect legacy db types as well.
	dbTypes := []string{"ffldb", "leveldb", "sqlite"}
	duplicateDbPaths := make([]string, 0, len(dbTypes)-1)
	for _, dbType := range dbTypes {
		if dbType == cfg.DbType {
			continue
		}

		// Store db path as a duplicate db if it exists.
		dbPath := blockDbPath(dbType)
		if fileExists(dbPath) {
			duplicateDbPaths = append(duplicateDbPaths, dbPath)
		}
	}

	// Warn if there are extra databases.
	if len(duplicateDbPaths) > 0 {
		selectedDbPath := blockDbPath(cfg.DbType)
		fmt.Printf("WARNING: There are multiple block chain databases "+
			"using different database types.\nYou probably don't "+
			"want to waste disk space by having more than one.\n"+
			"Your current database is located at [%v].\nThe "+
			"additional database is located at %v", selectedDbPath,
			duplicateDbPaths)
	}
}

// dbPath returns the path to the block database given a database type.
func blockDbPath(dbType string) string {
	// The database name is based on the database type.
	dbName := "blocks" + "_" + dbType
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(cfg.DataDir, dbName)
	return dbPath
}

// removeRegressionDB removes the existing regression test database if running
// in regression test mode and it already exists.
func removeRegressionDB(dbPath string) error {
	// Don't do anything if not in regression test mode.
	if !cfg.RegressionTest {
		return nil
	}

	// Remove the old regression test database if it already exists.
	fi, err := os.Stat(dbPath)
	if err == nil {
		fmt.Printf("Removing regression test database from '%s'", dbPath)
		if fi.IsDir() {
			err := os.RemoveAll(dbPath)
			if err != nil {
				return err
			}
		} else {
			err := os.Remove(dbPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
