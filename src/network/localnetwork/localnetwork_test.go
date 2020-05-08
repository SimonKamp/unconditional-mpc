package localnetwork

import (
	"math/big"
	".."
	"../../player"
	"testing"
)

func TestSetConnections(t *testing.T) {
	prime := big.NewInt(11)
	threshold := 2
	n := 5	
	parties := make([]network.Handler, n)
	for i := range(parties) {
		parties[i] = player.NewPlayer(prime, threshold, n, i+1)
	}

	for _, party := range(parties) {
		ln := new(localnetwork)
		ln.RegisterHandler(party)
		ln.SetConnections(parties...)
		party.RegisterNetwork(ln)
	}

	parties[0].(*player.Player).Send("data", 3)

}