package player

import (
	"testing"
	"../network"
	"../network/localnetwork"
	"math/big"
	"time"
)

func yield(milliseonds time.Duration) {
	time.Sleep(time.Millisecond * milliseonds)
}

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
	testAdd := func(a, b, prime int64) {
		parties := setting(prime, 1, 3)
		parties[0].Share(big.NewInt(a), "a")
		parties[1].Share(big.NewInt(b), "b")
		for _, party := range parties {
			go party.Add("a", "b", "aPlusB")
			go party.Open("aPlusB")
		}
		yield(10)
		res := big.NewInt((a + b) % prime)
		for _, party := range parties {
			reconstructed := party.Reconstruct("aPlusB")
			agreement := res.Cmp(reconstructed) == 0
			if !agreement {
				t.Errorf("Addition failed: expected %d + %d mod %d = %d, but got %d", a, b, prime, res, reconstructed)
			}
		}
	}
	testAdd(3, 9, 11)
}

func TestMultiply(t *testing.T) {
	testMult := func(a, b, prime int64) {
		parties := setting(prime, 1, 3)
		parties[0].Share(big.NewInt(a), "a")
		parties[1].Share(big.NewInt(b), "b")
		for _, party := range parties {
			go party.Multiply("a", "b", "aTimesB")
			go party.Open("aTimesB")
		}
		yield(10)
		res := big.NewInt((a * b) % prime)
		for _, party := range parties {
			reconstructed := party.Reconstruct("aTimesB")
			agreement := res.Cmp(reconstructed) == 0
			if !agreement {
				t.Errorf("Multiplication failed: expected %d * %d mod %d = %d, but got %d", a, b, prime, res, reconstructed)
			}
		}
	}
	testMult(3, 9, 11)
	testMult(3, 3, 11)
	testMult(0, 9, 11)
}