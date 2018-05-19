// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"golang.org/x/net/websocket"
)

// errorString is a trivial implementation of error.
type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

func (node *Node) addConn(conn *Conn) {
	node.mu.Lock()
	defer node.mu.Unlock()

	node.conns = append(node.conns, conn)

}

func (node *Node) deleteConn(id int64) {
	node.mu.Lock()
	defer node.mu.Unlock()

	conns := []*Conn{}
	for _, conn := range node.conns {
		if conn.id != id {
			conns = append(conns, conn)
		}
	}

	node.conns = conns
}

func (node *Node) disconnectPeer(conn *Conn) {
	defer conn.Close()
	node.log("disconnect peer:", conn.remoteHost())
	node.deleteConn(conn.id)
}

func (node *Node) newLatestBlockMessage() (*Message, error) {
	return newBlocksMessage(Blocks{node.blockchain.getLatestBlock()})
}

func (node *Node) newTxMessage(transaction ...transaction) (*Message, error) {

	return newTxMessage(transaction)
}

func (node *Node) newAllBlocksMessage(startPoint uint32) (*Message, error) {
	if startPoint==0{
		startPoint=1
	}
	bl, err := node.blockchain.getPartialChain(startPoint)
	if err != nil {
		return nil, err
		//newBlocksMessage()
		//return newQueryAllMessage(node.blockchain.chainHeight()+1, 0), err
	}
	return newBlocksMessage(bl)
}

func (node *Node) broadcast(msg *Message) {
	for _, conn := range node.conns {
		if err := node.send(conn, msg); err != nil {
			node.logError(err)
		}
	}
}

func (node *Node) sendRand(msg *Message) {
	sender := rand.Intn(len(node.conns))

	if err := node.send(node.conns[sender], msg); err != nil {
		node.logError(err)
	}

}

func (node *Node) send(conn *Conn, msg *Message) error {
	node.log(fmt.Sprintf(
		"send %s message to %s",
		msg.Type.name(), conn.RemoteAddr(),
	))

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return websocket.Message.Send(conn.Conn, b)
}

func (node *Node) handleTransactionResponse(conn *Conn, msg *Message) error {
	var transactions []transaction
	if err := json.Unmarshal([]byte(msg.Data), &transactions); err != nil {
		return err
	}
	var invalids = make([]transaction, 0)
	var valids = make([]transaction, 0)
	var err error;
	for i, _ := range transactions {

		if node.pool.contains(&transactions[i]) {
			return errorCall("Transaction with this ID is already in pool")
		}
		err = node.validateTransaction(&transactions[i],nil)
		if err != nil {
			invalids = append(invalids, transactions[i])
		} else {
			node.pool.addTransaction(&transactions[i])
			valids = append(valids, transactions[i])
		}
	}
	if len(valids) > 0 {
		msg, err2 := node.newTxMessage(valids...)
		if err2 != nil {
			return err
		}
		node.broadcast(msg)
	}

	return nil
}

func (node *Node) handleBlocksResponse(conn *Conn, msg *Message) error {
	var blocks Blocks
	if err := json.Unmarshal([]byte(msg.Data), &blocks); err != nil {
		return err
	}
	if blocks.Len() == 0 {
		return errorCall("recived empty blocks")
	}

	node.log(fmt.Sprintf("handle %v blocks, starting at index %v : ", blocks.Len(), blocks[0].Index))
	msg, broadcast, err2 := node.validateChain(blocks)

	if err2 != nil {
		return err2
	}

	if msg != nil {
		if broadcast {
			node.broadcast(msg)
		} else {
			node.send(conn, msg)
		}
	}

	return nil
}

func (node *Node) p2pHandler(conn *Conn) {
	for {
		var b []byte
		if err := websocket.Message.Receive(conn.Conn, &b); err != nil {
			if err == io.EOF {
				node.disconnectPeer(conn)
				break
			}
			node.logError(err)
			continue
		}

		var msg Message
		if err := json.Unmarshal(b, &msg); err != nil {
			node.logError(err)
			continue
		}
		node.log(fmt.Sprintf(
			"received %s message from %s",
			msg.Type.name(), conn.RemoteAddr(),
		))

		switch msg.Type {
		case MessageTypeQueryLatest:
			msg, err := node.newLatestBlockMessage()
			if err != nil {
				node.logError(err)
				continue
			}
			if err := node.send(conn, msg); err != nil {
				node.logError(err)
			}
		case MessageTypeQueryAll:
			fmt.Println("MessageTypeQueryAll starting from ", msg.StartPoint)
			response, err := node.newAllBlocksMessage(msg.StartPoint)
			//we get this error, if the request for a blockhight exceeds our latest stale block
			if err != nil {
				//we then ask the sender, to tell send us all block he has, starting from our chain height
				//if err := node.send(conn, response); err != nil {
				//	node.logError(err)
				//}
				continue
			}
			//we can serve the request and give him all blocks starting at msg.StartPoint
			if err := node.send(conn, response); err != nil {
				node.logError(err)
			}
		case MessageTypeBlocks:
			if err := node.handleBlocksResponse(conn, &msg); err != nil {
				node.logError(err)
			}
		case MessageTypeTransaction:
			if err := node.handleTransactionResponse(conn, &msg); err != nil {
				node.logError(err)
			}
		default:
			node.logError(ErrUnknownMessageType)
		}
	}
}
