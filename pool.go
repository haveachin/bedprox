package bedprox

import (
	"github.com/go-logr/logr"
)

type ConnPool struct {
	Log logr.Logger
}

func (cp *ConnPool) Start(poolChan <-chan ConnTunnel) {
	for {
		ct, ok := <-poolChan
		if !ok {
			break
		}

		go ct.Start()
	}
}
