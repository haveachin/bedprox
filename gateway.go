package bedprox

import (
	"net"

	"github.com/go-logr/logr"
)

type Gateway interface {
	GetID() string
	GetServerIDs() []string
	SetLogger(log logr.Logger)
	ListenAndServe(cpnChan chan<- net.Conn) error
}
