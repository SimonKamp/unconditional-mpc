package localnetwork

import (
	".."
)

type Localnetwork struct {
	connections map[int]network.Handler
	handler network.Handler
}

func (ln *Localnetwork)Send(data interface{}, receiver int) {
	go ln.connections[receiver].Handle(data, ln.handler.Index())
}

func (ln *Localnetwork)RegisterHandler(handler network.Handler) {
	ln.handler = handler
	ln.handler.RegisterNetwork(ln)
}

func (ln *Localnetwork)SetConnections(handlers ...network.Handler) {
	ln.connections = make(map[int]network.Handler)
	for _, handler := range handlers {
		ln.connections[handler.Index()] = handler
	}
}

func LocalNetworks(numberOfNetworks int) (networks []*Localnetwork) {
	networks = make([]*Localnetwork, numberOfNetworks)
	for i := range networks {
		networks[i] = new(Localnetwork)
	}
	return
}