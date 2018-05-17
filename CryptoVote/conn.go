// Copyright (c) 2018-2019 by mottla
// Use of this source code is governed by an MIT
// license that can be found in the LICENSE file.
package main

import (
	"net/url"
	"time"

	"golang.org/x/net/websocket"
)

type Conn struct {
	*websocket.Conn
	id int64
}

func newConn(ws *websocket.Conn) *Conn {
	return &Conn{
		Conn: ws,
		id:   time.Now().UnixNano(),
	}
}

func (conn *Conn) remoteHost() string {
	u, _ := url.Parse(conn.RemoteAddr().String())

	return u.Host
}
