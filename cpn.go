package bedprox

import (
	"github.com/go-logr/logr"
)

// Processing Node
type CPN struct {
	ConnProcessor
	Log logr.Logger
}

type ConnProcessor interface {
	ProcessConn(c Conn) (ProcessedConn, error)
}

func (cpn *CPN) Start(cpnChan <-chan Conn, srvChan chan<- ProcessedConn) {
	for {
		c, ok := <-cpnChan
		if !ok {
			break
		}
		cpn.Log.Info("processing",
			"remoteAddress", c.RemoteAddr(),
		)

		pc, err := cpn.ProcessConn(c)
		if err != nil {
			cpn.Log.Error(err, "processing",
				"remoteAddress", c.RemoteAddr(),
			)
			pc.Close()
			continue
		}
		srvChan <- pc
	}
}
