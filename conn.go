package bedprox

import (
	"bufio"
	"io"
	"net"

	"github.com/sandertv/go-raknet"
)

type PacketWriter interface {
	WritePacket(pk []byte) error
}

type PacketReader interface {
	ReadPacket() ([]byte, error)
}

type conn struct {
	*raknet.Conn
}

func newConn(c net.Conn) Conn {
	return &conn{
		Conn: c.(*raknet.Conn),
	}
}

// Conn is a minecraft Connection
type Conn interface {
	net.Conn
	PacketReader

	Reader() *bufio.Reader
}

func (c *conn) Reader() *bufio.Reader {
	return bufio.NewReader(c)
}

type ProcessingConn struct {
	Conn
	readBytes     []byte
	remoteAddr    net.Addr
	srvHost       string
	username      string
	proxyProtocol bool
	serverIDs     []string
}

func (c ProcessingConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

type ProcessedConn struct {
	ProcessingConn
	ServerConn *raknet.Conn
	ServerID   string
}

func (c ProcessedConn) StartPipe() {
	defer c.Close()

	go io.Copy(c.ServerConn, c)
	io.Copy(c, c.ServerConn)
}

func (c ProcessedConn) Close() {
	c.ServerConn.Close()
	c.ProcessingConn.Close()
}
