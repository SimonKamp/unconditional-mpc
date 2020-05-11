package player

import (
	"testing"
	"../network"
	"../network/localnetwork"
	"math/big"
	"time"
	"fmt"
)

func yield(milliseonds time.Duration) {
	time.Sleep(time.Millisecond * milliseonds)
}

func setting(prime int64, threshold, n int) map[int]*Player {
	parties := make(map[int]*Player, n)
	handlers := make([]network.Handler, n)
	for i := range(handlers) {
		parties[i+1] = NewPlayer(prime, threshold, n, i+1)
		handlers[i] = parties[i+1]
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
	parties[1].Share(big.NewInt(3), "id3")
	for _, party := range parties {
		party.Open("id3")
	}
	parties[1].Reconstruct("id3")
}

func TestAdd(t *testing.T) {
	testAdd := func(a, b, prime int64) {
		parties := setting(prime, 1, 3)
		parties[1].Share(big.NewInt(a), "a")
		parties[2].Share(big.NewInt(b), "b")
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
		parties[1].Share(big.NewInt(a), "a")
		parties[2].Share(big.NewInt(b), "b")
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

func TestInterpret(t *testing.T) {
	parties := setting(11, 1, 3)
	party1Input := map[string]*big.Int{
		"A": big.NewInt(1),
	}
	parties[1].setInput(party1Input)
	party2Input := map[string]*big.Int{
		"B": big.NewInt(2),
	}
	parties[2].setInput(party2Input)
	party3Input := map[string]*big.Int{
		"C": big.NewInt(3),
	}
	parties[3].setInput(party3Input)
	instructions := []instruction{
		instruction{"IN", "1", "A"},
		instruction{"IN", "2", "B"},
		instruction{"IN", "3", "C"},
		instruction{"ADD", "A", "B", "APB"},
		instruction{"MUL", "APB", "C", "3X3"},
		instruction{"MUL", "3X3", "3X3", "9X9"},
		instruction{"MUL", "9X9", "9X9", "4X4"},
		instruction{"OPEN", "9X9"},
		instruction{"OUT", "9X9"},
		instruction{"OPEN", "3X3"},
		instruction{"OUT", "3X3"},
		instruction{"OPEN", "4X4"},
		instruction{"OUT", "4X4"},
	}
	go parties[1].Interpret(instructions)
	go parties[2].Interpret(instructions)
	output := parties[3].Interpret(instructions)
	for id, val := range output {
		fmt.Println("Test:", id, val)
	}
	yield(10)
}