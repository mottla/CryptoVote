// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"testing"
	"fmt"
	"flag"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"net/http"
	"time"
)

func Test1Node(t *testing.T) {
	defer fmt.Printf("Shutdown complete")
	interrupt := interruptListener()
	flag.Parse()
	a, b := ":6000", ":3000"
	go newNode(a,b).run()
	<-interrupt

}
func Test2Node(t *testing.T) {
	defer fmt.Printf("Shutdown complete")

	flag.Parse()
	a, b := ":6001", ":3001"
	go newNode(a,b).run()
	time.Sleep(3 * time.Second)
	connectPeers(":6000", ":3001")

	var input string
	fmt.Scanln(&input)
}

func Test3Node(t *testing.T) {
	defer fmt.Printf("Shutdown complete")

	flag.Parse()
	a, b := ":6003", ":3003"
	go newNode(a,b).run()
	time.Sleep(200 * time.Second)
	connectPeers(":6001", ":3003")

	var input string
	fmt.Scanln(&input)
}
func Test4Node(t *testing.T) {
	defer fmt.Printf("Shutdown complete")

	flag.Parse()
	a, b := ":6004", ":3004"
	go newNode(a,b).run()
	time.Sleep(50 * time.Second)
	connectPeers(":6003", ":3004")

	var input string
	fmt.Scanln(&input)
}
func Test5Node(t *testing.T) {


	connectPeers(":6004", ":3000")

}


func Test4Nodes(t *testing.T) {
	defer fmt.Printf("Shutdown complete")

	flag.Parse()
	a, b, c, d := ":6000", ":3000", ":6001", ":3001"
	go newNode(a,b).run()
	go newNode(c,d).run()
	e, f, g, h := ":6002", ":3002", ":6003", ":3003"
	go newNode(e,f).run()
	go newNode(g,h).run()

	time.Sleep(2 * time.Second)
	connectPeers(a, d)
	connectPeers(c, f)
	connectPeers(g, b)
	var input string
	fmt.Scanln(&input)
}

func connectPeers(from_wsPort, to_apiPort string) {
	u := make(map[string]string)
	u["peer"] = fmt.Sprintf("ws://127.0.0.1%s", from_wsPort)
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(u)
	res, _ := http.Post(fmt.Sprintf("http://127.0.0.1%s/addPeer", to_apiPort), "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)
}
