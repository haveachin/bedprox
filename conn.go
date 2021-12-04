package bedprox

import (
	"io"
	"net"
)

type ProcessedConn interface {
	net.Conn
	Username() string
	ServerAddr() string
}

type ConnTunnel struct {
	Conn       net.Conn
	RemoteConn net.Conn
}

func (t ConnTunnel) Start() {
	defer t.Close()

	go io.Copy(t.Conn, t.RemoteConn)
	io.Copy(t.RemoteConn, t.Conn)
}

func (t ConnTunnel) Close() {
	_ = t.Conn.Close()
	_ = t.RemoteConn.Close()
}
