package player

import (
	"testing"
	"../network"
	"../network/localnetwork"
	"math/big"
)

func setting(prime int64, threshold, n int) []*Player {
	parties := make([]*Player, n)
	handlers := make([]network.Handler, n)
	for i := range(handlers) {
		parties[i] = NewPlayer(prime, threshold, n, i+1)
		handlers[i] = parties[i]
	}

	for _, party := range(parties) {
		ln := new(localnetwork.Localnetwork)
		ln.RegisterHandler(party)
		ln.SetConnections(handlers...)
	}

	return parties
}

func TestShare(t *testing.T) {
	parties := setting(11, 1, 3)
	parties[0].Share(big.NewInt(3), "id3")
	for _, party := range parties {
		party.Open("id3")
	}
	parties[0].Reconstruct("id3")
}

func TestAdd(t *testing.T) {
	parties := setting(11, 1, 3)
	parties[0].Share(big.NewInt(3), "a")
	parties[1].Share(big.NewInt(9), "b")
	for _, party := range parties {
		party.Add("a", "b", "aPlusB")
		party.Open("aPlusB")
	}
	parties[0].Reconstruct("aPlusB")
}