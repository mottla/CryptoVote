package main

import (
	"testing"
	"bytes"
	"io"
	"os"
	"net/http"
	"encoding/json"
	"fmt"
)

func TestAddConn(t *testing.T) {
	node := newNode(*p2pAddr,*apiAddr)
	conn1 := newConn(nil)
	conn2 := newConn(nil)

	node.addConn(conn1)
	if len(node.conns) != 1 {
		t.Errorf("want 1 but %d", len(node.conns))
	}

	node.addConn(conn2)
	if len(node.conns) != 2 {
		t.Errorf("want 2 but %d", len(node.conns))
	}
}

func TestAddConn_HTTP(t *testing.T) {
	u := make(map[string]string)
	u["peer"]="ws://127.0.0.1:6001"
	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(u)
	fmt.Printf(b.String())
	res, _ := http.Post("http://127.0.0.1:3000/addPeer", "application/json; charset=utf-8", b)
	io.Copy(os.Stdout, res.Body)
}


func TestDeleteConn(t *testing.T) {
	node := newNode(*p2pAddr,*apiAddr)
	conn1 := newConn(nil)
	conn2 := newConn(nil)
	node.conns = []*Conn{conn1, conn2}

	node.deleteConn(conn1.id)
	if len(node.conns) != 1 {
		t.Fatalf("want 1 but %d", len(node.conns))
	}
	if node.conns[0].id != conn2.id {
		t.Errorf("want %d but %d", conn2.id, node.conns[0].id)
	}

	node.deleteConn(conn2.id)
	if len(node.conns) != 0 {
		t.Fatalf("want 0 but %d", len(node.conns))
	}
}
