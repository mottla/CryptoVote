package main

import (
	"errors"
	"flag"
	"fmt"
)

var (
	apiAddr   = flag.String("api", ":3001", "HTTP server address for API")
	p2pAddr   = flag.String("p2p", ":6001", "WebSocket server address for P2P")
	p2pOrigin = flag.String("origin", "http://127.0.0.1", "P2P origin")

	ErrInvalidChain       = errors.New("invalid chain")
	ErrInvalidBlock       = errors.New("invalid block")
	ErrUnknownMessageType = errors.New("unknown message type")
)

func main() {
	//interrupt := interruptListener()

	defer fmt.Printf("Shutdown complete")

	flag.Parse()
	a, b, c, d := ":6000", ":3000", ":6001", ":3001"

	//a=new(string)

	fmt.Printf(" %s ### %s\n", *p2pAddr, *apiAddr)
	go newNode(a, b).run()
	newNode(c, d).run()
	//newNode().run(&a, &b)
	//newNode().run(&c, &d)
	//fmt.Printf(*p2pAddr)
	//newNode().run(p2pAddr, apiAddr)

	//<-interrupt

}
