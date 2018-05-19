// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

)

var (
	apiAddr   = flag.String("api", ":3000", "HTTP server address for API")
	p2pAddr   = flag.String("p2p", ":6000", "WebSocket server address for P2P")
	p2pOrigin = flag.String("origin", "http://127.0.0.1", "P2P origin")

	ErrInvalidChain       = errors.New("invalid chain")
	ErrInvalidBlock       = errors.New("invalid block")
	ErrUnknownMessageType = errors.New("unknown message type")
)

func main() {
	interrupt := interruptListener()

	//defers dont work on os.exit().. Ill check that
	defer fmt.Printf("Shutdown complete")

	// Use all processor cores and up some limits.
	runtime.GOMAXPROCS(runtime.NumCPU())
	if err := SetLimits(); err != nil {
		os.Exit(1)
	}

	flag.Parse()

	go newNode(*p2pAddr, *apiAddr).run()
	<-interrupt

}
