package bedrock

import (
	"net"

	"github.com/sandertv/go-raknet"
)

// Conn is a minecraft Connection
type Conn struct {
	*raknet.Conn

	proxyProtocol bool
	serverIDs     []string
}

type ProcessedConn struct {
	*Conn
	readBytes     []byte
	remoteAddr    net.Addr
	srvHost       string
	username      string
	proxyProtocol bool
	serverIDs     []string
}

func (c ProcessedConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c ProcessedConn) Username() string {
	return c.username
}

func (c ProcessedConn) ServerAddr() string {
	return c.srvHost
}
