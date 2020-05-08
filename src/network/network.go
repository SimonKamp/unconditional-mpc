package network

type (
	//Network is an abstraction of the method used to connect parties
	Network interface {
		Send(data interface{}, receiver int)
		RegisterHandler(handler Handler)
	}
	//Handler handles data received from the network
	Handler interface {
		Handle(data interface{}, sender int)
		RegisterNetwork(network Network)
		Index() int
	}
)