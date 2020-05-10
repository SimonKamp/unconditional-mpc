package localnetwork

import (
	".."
	"../../player"
	"testing"
)

func TestSetConnections(t *testing.T) {
	prime := int64(11)
	threshold := 2
	n := 5	
	parties := make([]network.Handler, n)
	for i := range(parties) {
		parties[i] = player.NewPlayer(prime, threshold, n, i+1)
	}

	networks := LocalNetworks(n)
	for i, network := range(networks) {
		network.RegisterHandler(parties[i])
		network.SetConnections(parties...)
	}

	parties[0].(*player.Player).Send("data", 3)

}