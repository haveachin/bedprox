package bedprox

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/sandertv/go-raknet"
)

type Listener struct {
	Bind       string
	PingStatus PingStatus

	*raknet.Listener
}

type Gateway struct {
	ID                   string
	Listeners            []Listener
	ReceiveProxyProtocol bool
	ClientTimeout        time.Duration
	ServerIDs            []string
	Log                  logr.Logger
}

type PingStatus struct {
	Edition         string
	ProtocolVersion int
	VersionName     string
	PlayerCount     int
	MaxPlayerCount  int
	GameMode        string
	GameModeNumeric int
	MOTD            string
}

func (p PingStatus) marshal(l *raknet.Listener) []byte {
	motd := strings.Split(p.MOTD, "\n")
	motd1 := motd[0]
	motd2 := ""
	if len(motd) > 1 {
		motd2 = motd[1]
	}

	port := l.Addr().(*net.UDPAddr).Port
	return []byte(fmt.Sprintf("%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;%v;",
		p.Edition, motd1, p.ProtocolVersion, p.VersionName, p.PlayerCount, p.MaxPlayerCount,
		l.ID(), motd2, p.GameMode, p.GameModeNumeric, port, port))
}

func (gw *Gateway) Start(cpnChan chan<- ProcessingConn) error {
	for n, listener := range gw.Listeners {
		gw.Log.Info("Start listener",
			"bind", listener.Bind,
		)

		l, err := raknet.Listen(listener.Bind)
		if err != nil {
			return err
		}
		l.PongData(listener.PingStatus.marshal(l))

		gw.Listeners[n].Listener = l
	}

	gw.listenAndServe(cpnChan)
	return nil
}

func (gw Gateway) wrapConn(c net.Conn) ProcessingConn {
	return ProcessingConn{
		Conn:          newConn(c),
		remoteAddr:    c.RemoteAddr(),
		proxyProtocol: gw.ReceiveProxyProtocol,
		serverIDs:     gw.ServerIDs,
	}
}

func (gw *Gateway) listenAndServe(cpnChan chan<- ProcessingConn) {
	wg := sync.WaitGroup{}
	wg.Add(len(gw.Listeners))

	for _, listener := range gw.Listeners {
		l := listener
		go func() {
			for {
				c, err := l.Accept()
				if err != nil {
					break
				}

				gw.Log.Info("connected",
					"remoteAddress", c.RemoteAddr(),
				)

				cpnChan <- gw.wrapConn(c)
			}
			wg.Done()
		}()
	}

	wg.Wait()
}
