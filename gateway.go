package bedprox

type Gateway interface {
	ListenAndServe(cpnChan chan<- Conn) error
}
