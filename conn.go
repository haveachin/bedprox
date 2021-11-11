package bedprox

import (
	"io"
	"net"
)

type PacketWriter interface {
	WritePacket(pk []byte) error
}

type PacketReader interface {
	ReadPacket() ([]byte, error)
}

// Conn is a minecraft Connection
type Conn interface {
	net.Conn
	PacketReader
	PacketWriter
}

type ProcessedConn interface {
	Conn
	Username() string
	ServerAddr() string
}

type ConnTunnel struct {
	c  net.Conn
	rc net.Conn
}

func (t ConnTunnel) Start() {
	defer t.Close()

	go io.Copy(t.c, t.rc)
	io.Copy(t.rc, t.c)
}

func (t ConnTunnel) Close() {
	_ = t.c.Close()
	_ = t.rc.Close()
}
