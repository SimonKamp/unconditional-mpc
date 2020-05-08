package localnetwork

import ".."

type localnetwork struct {
	connections map[int]network.Handler
	handler network.Handler
}

func (ln *localnetwork)Send(data interface{}, receiver int) {
	ln.connections[receiver].Handle(data, ln.handler.Index())
}

func (ln *localnetwork)RegisterHandler(handler network.Handler) {
	ln.handler = handler
}

func (ln *localnetwork)SetConnections(handlers ...network.Handler) {
	ln.connections = make(map[int]network.Handler)
	for _, handler := range handlers {
		ln.connections[handler.Index()] = handler
	}
}