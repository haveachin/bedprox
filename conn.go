package bedprox

import (
	"io"
	"net"
)

type ProcessedConn interface {
	net.Conn
	GatewayID() string
	Username() string
	ServerAddr() string
	Disconnect(msg string) error
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
	if t.Conn != nil {
		_ = t.Conn.Close()
	}
	if t.RemoteConn != nil {
		_ = t.RemoteConn.Close()
	}
}
