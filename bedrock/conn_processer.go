package bedprox

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"strings"

	"github.com/go-logr/logr"
	"github.com/haveachin/bedprox/protocol"
	"github.com/haveachin/bedprox/protocol/login"
	"github.com/pires/go-proxyproto"
)

// Processing Node
type ConnProcessor struct {
	Log logr.Logger
}

func (cp *ConnProcessor) Start(cpnChan <-chan ProcessingConn, srvChan chan<- ProcessingConn) {
	for {
		c, ok := <-cpnChan
		if !ok {
			break
		}
		cp.Log.Info("processing",
			"remoteAddress", c.RemoteAddr(),
		)

		if err := cp.ProcessConn(&c); err != nil {
			cp.Log.Error(err, "processing",
				"remoteAddress", c.RemoteAddr(),
			)
			c.Close()
			continue
		}
		srvChan <- c
	}
}

func (cp ConnProcessor) ProcessConn(c *ProcessingConn) error {
	if c.proxyProtocol {
		header, err := proxyproto.Read(bufio.NewReader(c))
		if err != nil {
			return err
		}
		c.remoteAddr = header.SourceAddr
	}

	b, err := c.ReadPacket()
	if err != nil {
		return err
	}
	c.readBytes = b

	decoder := protocol.NewDecoder(bytes.NewReader(b))
	pks, err := decoder.Decode()
	if err != nil {
		return err
	}

	if len(pks) < 1 {
		return errors.New("no valid packets received")
	}

	var loginPk protocol.Login
	if err := protocol.Unmarshal(pks[0], &loginPk); err != nil {
		return err
	}

	iData, cData, err := login.Parse(loginPk.ConnectionRequest)
	if err != nil {
		return err
	}
	c.username = iData.DisplayName
	c.srvHost = cData.ServerAddress

	if strings.Contains(c.srvHost, ":") {
		c.srvHost, _, err = net.SplitHostPort(c.srvHost)
		if err != nil {
			return err
		}
	}

	return nil
}
