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
		yield(1)
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
		yield(1)
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

func TestRun(t *testing.T) {
	parties := setting(11, 1, 3)
	
	parties[1].scanInput("tests/test1/input1")
	parties[2].scanInput("tests/test1/input2")
	parties[3].scanInput("tests/test1/input3")

	parties[1].scanInstructions("tests/test1/sourcefile")
	parties[2].scanInstructions("tests/test1/sourcefile")
	parties[3].scanInstructions("tests/test1/sourcefile")

	go parties[1].Run()
	go parties[2].Run()
	output := parties[3].Run()
	if output["3*3"].Cmp(big.NewInt(9)) != 0 {
		t.Errorf("3 * 3 mod 11 should be 9 was %d", output["3*3"])
	}
	if output["9*9"].Cmp(big.NewInt(4)) != 0 {
		t.Errorf("9 * 9 mod 11 should be 4 was %d", output["9*9"])
	}
	if output["4*4"].Cmp(big.NewInt(5)) != 0 {
		t.Errorf("4 * 4 mod 11 should be 5 was %d", output["4*4"])
	}
}

func TestRandomBit(t *testing.T) {
	test := func() {
		//The random field element is zero with pr. 1/5
		parties := setting(5, 1, 3)
		res := make([]map[string]*big.Int, 3)
	
		run := func(i int) {
			res[i] = parties[i].Run()
		}
		for _, party := range parties {
			party.scanInstructions("tests/testRandomBit/prog")
		}
		go run(1)
		go run(2)
		output := parties[3].Run()
	
		yield(10)//not thread safe
		if output["b"].Cmp(big.NewInt(0)) != 0 && output["b"].Cmp(big.NewInt(1)) != 0 {
			t.Errorf("Random bit is not a bit: %d", output["b"])
		}
	
		if output["b"].Cmp(res[1]["b"]) != 0 || output["b"].Cmp(res[2]["b"]) != 0 {
			t.Errorf("Random bits do not agree: %d %d %d", output["b"], res[1]["b"], res[2]["b"])
		}
	}
	for i := 0; i < 20; i++ {
		test()
	}
}