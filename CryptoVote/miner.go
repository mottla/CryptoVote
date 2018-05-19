// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"sync"
	"time"
	"bytes"
	"log"
	"fmt"

)

const (
	// maxNonce is the maximum value a nonce can be in a block header.
	maxNonce = ^uint32(0) // 2^32 - 1

	// maxExtraNonce is the maximum value an extra nonce used in a coinbase
	// transactionByID can be.
	maxExtraNonce = ^uint64(0) // 2^64 - 1

	// hpsUpdateSecs is the number of seconds to wait in between each
	// update to the hashes per second monitor.
	hpsUpdateSecs = 10

	// hashUpdateSec is the number of seconds each worker waits in between
	// notifying the speed monitor with how many hashes have been completed
	// while they are actively searching for a solution.  This is done to
	// reduce the amount of syncs between the workers that must be done to
	// keep track of the hashes per second.
	hashUpdateSecs = 15
)

// CPUMiner provides facilities for solving blocks (mining) using the CPU in
// a concurrency-safe manner.  It consists of two main goroutines -- a speed
// monitor and a controller for worker goroutines which generate and solve
// blocks.  The number of goroutines can be set via the SetMaxGoRoutines
// function, but the default is based on the number of processor cores in the
// system which is typically sufficient.
type CPUMiner struct {
	sync.Mutex
	//g                 *mining.BlkTmplGenerator
	//cfg               Config
	not               notifier //used to broadcast a solution over the network
	bc                *Blockchain
	pool              *TxPool
	numWorkers        uint8
	started           bool
	submitBlockLock   sync.Mutex
	wg                sync.WaitGroup
	workerWg          sync.WaitGroup
	updateNumWorkers  chan struct{}
	updateHight       chan uint32
	queryHashesPerSec chan float64
	updateHashes      chan uint64
	speedMonitorQuit  chan struct{}
	quit              chan struct{}
	logger            *log.Logger
}

func (m *CPUMiner) log(v ...interface{}) {

	m.logger.Println(v)

}

func (m *CPUMiner) logError(err error) {
	m.log("[ERROR]", err)
}

// Start begins the CPU mining process as well as the speed monitor used to
// track hashing metrics.  Calling this function when the CPU miner has
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (m *CPUMiner) Start(blocksToMine, procs uint8) {

	// Respond with an error if server is already mining.
	if m.started {
		//m.Unlock()
		m.log("miner already running")
		return
	}
	m.Lock()
	m.log("miner started with ", procs, "processors and ",blocksToMine," blocks to mine")
	m.started = true
	defer func() {
		m.Unlock()
		//m.started = false
	}()

	m.numWorkers = procs
	m.quit = make(chan struct{})
	m.speedMonitorQuit = make(chan struct{})
	m.updateHight = make(chan uint32)
	m.wg.Add(2)

	go m.miningWorkerController(int(blocksToMine))
	go m.speedMonitor()

}

// speedMonitor handles tracking the number of hashes per second the mining
// process is performing.  It must be run as a goroutine.
func (m *CPUMiner) speedMonitor() {
	m.log("CPU miner speed monitor started")

	var hashesPerSec float64
	var totalHashes uint64
	ticker := time.NewTicker(time.Second * hpsUpdateSecs)
	bcticker := time.NewTicker(time.Second * 15)
	defer ticker.Stop()

out:
	for {
		select {
		// Periodic updates from the workers with how many hashes they
		// have performed.
		case numHashes := <-m.updateHashes:
			totalHashes += numHashes
		case <-bcticker.C:
			m.log("current block height ", m.bc.getLatestBlock().Index)

			// Time to update the hashes per second.
		case <-ticker.C:
			curHashesPerSec := float64(totalHashes) / hpsUpdateSecs
			if hashesPerSec == 0 {
				hashesPerSec = curHashesPerSec
			}
			hashesPerSec = (hashesPerSec + curHashesPerSec) / 2
			totalHashes = 0
			if hashesPerSec != 0 {
				m.log(fmt.Sprintf("Hash speed: %6.0f kilohashes/s\n",
					hashesPerSec/1000))
			}

			// Request for the number of hashes per second.
		case m.queryHashesPerSec <- hashesPerSec:
			// Nothing to do.

		case <-m.speedMonitorQuit:
			close(m.speedMonitorQuit)
			break out
		}
	}

	m.wg.Done()
	m.log("CPU miner speed monitor done")
}

type workersPhone struct {
	quit   chan struct{}
	update chan bool
}

// miningWorkerController launches the worker goroutines that are used to
// generate block templates and solve them.  It also provides the ability to
// dynamically adjust the number of running worker goroutines.
//
// It must be run as a goroutine.
func (m *CPUMiner) miningWorkerController(blocksToMine int) {
	// launchWorkers groups common code to launch a specified number of
	// workers for generating blocks.
	var runningWorkers []workersPhone

	launchWorkers := func(numWorkers uint8) {
		for i := uint8(0); i < numWorkers; i++ {
			quit := make(chan struct{})
			update := make(chan bool)
			runningWorkers = append(runningWorkers, workersPhone{quit, update})

			m.workerWg.Add(1)
			go m.generateBlocks(quit, update)
		}
	}

	// Launch the current number of workers by default.
	runningWorkers = make([]workersPhone, 0, m.numWorkers)
	launchWorkers(m.numWorkers)
	start := m.bc.chainHeight();
	for {
		select {
		// Update the number of running workers.
		case <-m.updateNumWorkers:
			// No change.
			numRunning := uint8(len(runningWorkers))
			if m.numWorkers == numRunning {
				continue
			}

			// Add new workers.
			if m.numWorkers > numRunning {
				launchWorkers(m.numWorkers - numRunning)
				continue
			}

			// Signal the most recently created goroutines to exit.
			for i := numRunning - 1; i >= m.numWorkers; i-- {
				close(runningWorkers[i].quit)
				runningWorkers[i].quit = nil
				runningWorkers = runningWorkers[:i]
			}
		case height := <-m.updateHight:

			//we send a update signal to all workers
			//this forces them to get a new template block to work on
			for i, _ := range runningWorkers {
				go func() { runningWorkers[i].update <- true }()
			}

			if int(height-start) ==  blocksToMine {
				for _, ch := range runningWorkers {
					close(ch.quit)
				}
				goto out
			}
			//runningWorkers = make([]chan struct{}, 0, m.numWorkers)
			//launchWorkers(m.numWorkers)

		case <-m.quit:
			for _, ch := range runningWorkers {
				close(ch.quit)
			}
			goto out
		}
	}
out:
// Wait until all workers shut down to stop the speed monitor since
// they rely on being able to send updates to it.
	m.speedMonitorQuit <- struct{}{}
	m.workerWg.Wait()
	m.wg.Done()
	m.started = false
	m.log("miner controller ended")
}

// generateBlocks is a worker that is controlled by the miningWorkerController.
// It is self contained in that it creates block templates and attempts to solve
// them while detecting when it is performing stale work and reacting
// accordingly by generating a new block template.  When a block is solved, it
// is submitted.
//
// It must be run as a goroutine.
func (m *CPUMiner) generateBlocks(quit chan struct{}, update chan bool) {
	//m.log("Starting generate blocks worker")

	// Start a ticker which is used to signal checks for stale work and
	// updates to the speed monitor.
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
out:
	for {
		// Quit when the miner is stopped.
		select {
		case <-quit:
			break out
		default:
			// Non-blocking select to fall through
		}
		//create a block template for each go routine
		//so far the block content is randomly chosen from the txPool
		temp := m.templateBlock()
		// Attempt to solve the block.  The function will exit early
		// with false when conditions that trigger a stale block, so
		// a new block template can be generated.  When the return is
		// true a solution was found, so submit the solved block.
		if m.solveBlock(temp, ticker, quit, update) {
			m.log("Found block on height ", temp.Index, " with ", temp.Hash[:11],"...")
			//if(m.bc.addBlock(temp, m.pool)){
			//	m.not.submitBlock()
			//}
			//m.updateHight <- m.bc.chainHeight()
			m.not.submitBlock(temp)

			//m.log("Found2", temp.Hash)
			//m.submitBlock(block)
		}
	}
	m.workerWg.Done()
}

// solveBlock attempts to find some combination of a nonce, extra nonce, and
// current timestamp which makes the passed block hash to a value less than the
// target difficulty.  The timestamp is updated periodically and the passed
// block is modified with all tweaks during this process.  This means that
// when the function returns true, the block is ready for submission.
//
// This function will return early with false when conditions that trigger a
// stale block such as a new block showing up or periodically when there are
// new transactions and enough time has elapsed without finding a solution.
func (m *CPUMiner) solveBlock(msgBlock *Block,
	ticker *time.Ticker, quit chan struct{}, update chan bool) bool {

	//// Create some convenience variables.
	//header := &msgBlock.Header
	//targetDifficulty := blockchain.CompactToBig(header.Bits)
	//
	//// Initial state.
	//lastGenerated := time.Now()
	//lastTxUpdate := m.g.TxSource().LastUpdated()
	hashesCompleted := uint64(0)
	var hash [32]byte

	targetDifficulty := make([]byte, 32)
	targetDifficulty[msgBlock.Difficulty/8] = (byte(128) >> (msgBlock.Difficulty % 8))

	for {
		msgBlock.updateExtraNonce()

		// Search through the entire nonce range for a solution while
		// periodically checking for early quit and stale block
		// conditions along with updates to the speed monitor.
		for i := uint32(0); i < maxNonce; i++ {

			//todo test only
			time.Sleep(250 * time.Millisecond)

			select {
			case <-quit:
				return false
			case <-update:
				return false

			case <-ticker.C:
				m.updateHashes <- hashesCompleted
				hashesCompleted = 0

				msgBlock.Timestamp = time.Now().Unix()
				if msgBlock.Index <= m.bc.getLatestBlock().Index {
					//we are working on a stale block.
					return false
				}
			default:
				// Non-blocking select to fall through
			}

			// Update the nonce and hash the block header.
			msgBlock.Nonce = i
			hash = msgBlock.hash()
			hashesCompleted += 1

			// The block is solved when the new block hash is less
			// than the target difficulty.  Yay!
			if bytes.Compare(hash[:], targetDifficulty) == -1 {
				msgBlock.Hash = hash
				m.updateHashes <- hashesCompleted
				return true
			}
		}
	}
	//unreachable
	return false
}

func (m *CPUMiner) templateBlock() *Block {

	data := m.pool.randPoolTransaction()
	//we could extend the protocol to support merkletrees, but lets keep it simple
	bl := m.bc.getLatestBlock()
	block :=
		&Block{
			Index:        bl.Index + 1,
			PreviousHash: bl.Hash,
			Timestamp:    time.Now().Unix(),
			Data:         *data,
			Difficulty:   bl.Difficulty,
		}
	return block
}

// Stop gracefully stops the mining process by signalling all workers, and the
// speed monitor to quit.  Calling this function when the CPU miner has not
// already been started will have no effect.
//
// This function is safe for concurrent access.
func (m *CPUMiner) Stop() {
	m.Lock()
	defer m.Unlock()

	// Nothing to do if the miner is not currently running or if running in
	// discrete mode (using GenerateNBlocks).
	if !m.started {
		return
	}

	close(m.quit)
	m.wg.Wait()
	m.started = false
	m.log("CPU miner stopped")
}

// IsMining returns whether or not the CPU miner has been started and is
// therefore currenting mining.
//
// This function is safe for concurrent access.
func (m *CPUMiner) IsMining() bool {
	return m.started
}
