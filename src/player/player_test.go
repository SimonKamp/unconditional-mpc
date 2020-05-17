package player

import (
	"math/big"
	"strconv"
	"testing"
	"time"

	"../network"
	"../network/localnetwork"
)

func yield(milliseonds time.Duration) {
	time.Sleep(time.Millisecond * milliseonds)
}

func setting(prime int64, threshold, n int) map[int]*Player {
	parties := make(map[int]*Player, n)
	handlers := make([]network.Handler, n)
	for i := range handlers {
		parties[i+1] = NewPlayer(prime, threshold, n, i+1)
		handlers[i] = parties[i+1]
	}

	for _, party := range parties {
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

		yield(10) //not thread safe
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

func TestRandomSolvedBits(t *testing.T) {
	parties := setting(47, 1, 3)

	go parties[1].randomSolvedBits("r")
	go parties[2].randomSolvedBits("r")
	rID, rBitsIDs := parties[3].randomSolvedBits("r")
	for _, party := range parties {
		go party.Open(rID)
		for _, id := range rBitsIDs {
			go party.Open(id)
		}
	}
	r1 := parties[1].Reconstruct(rID)
	r2 := parties[2].Reconstruct(rID)
	r3 := parties[3].Reconstruct(rID)
	//Agreement on r and r < P
	if r1.Cmp(r2) != 0 || r1.Cmp(r3) != 0 || parties[1].prime.Cmp(r1) != 1 {
		t.Error(r1, r2, r3, parties[1].prime)
	}
	for bitIndex := range rBitsIDs {
		rBit := parties[1].Reconstruct(rBitsIDs[bitIndex])
		if rBit.Int64() != int64(r1.Bit(bitIndex)) {
			t.Error(rBit, r1.Bit(bitIndex))
		}
	}
}

func TestCompare(t *testing.T) {
	parties := setting(31, 1, 3)

	parties[1].Share(big.NewInt(3), "id3")
	parties[2].Share(big.NewInt(5), "id5")

	for _, party := range parties {
		go party.Compare("id3", "id5", "id3>5")
		go party.Open("id3>5")
	}

	r1 := parties[1].Reconstruct("id3>5")
	r2 := parties[2].Reconstruct("id3>5")
	r3 := parties[3].Reconstruct("id3>5")
	//Agreement on 3 > 5
	if r1.Sign() != 0 || r2.Sign() != 0 || r3.Sign() != 0 {
		t.Error(r1, r2, r3)
	}
}

func TestFullAdder(t *testing.T) {
	parties := setting(13, 1, 3)

	parties[1].Share(big.NewInt(0), "0")
	parties[2].Share(big.NewInt(1), "1")

	for _, party := range parties {
		go party.fullAdder("0", "0", "0", "carryout000", "add000")
		go party.fullAdder("0", "0", "1", "carryout001", "add001")
		go party.fullAdder("0", "1", "0", "carryout010", "add010")
		go party.fullAdder("0", "1", "1", "carryout011", "add011")
		go party.fullAdder("1", "0", "0", "carryout100", "add100")
		go party.fullAdder("1", "0", "1", "carryout101", "add101")
		go party.fullAdder("1", "1", "0", "carryout110", "add110")
		go party.fullAdder("1", "1", "1", "carryout111", "add111")

		go party.Open("carryout000")
		go party.Open("carryout001")
		go party.Open("carryout010")
		go party.Open("carryout011")
		go party.Open("carryout100")
		go party.Open("carryout101")
		go party.Open("carryout110")
		go party.Open("carryout111")

		go party.Open("add000")
		go party.Open("add001")
		go party.Open("add010")
		go party.Open("add011")
		go party.Open("add100")
		go party.Open("add101")
		go party.Open("add110")
		go party.Open("add111")

	}

	party := parties[3]
	shouldBe(0, party.Reconstruct("carryout000"), "carryout000", t)
	shouldBe(0, party.Reconstruct("carryout001"), "carryout001", t)
	shouldBe(0, party.Reconstruct("carryout010"), "carryout010", t)
	shouldBe(1, party.Reconstruct("carryout011"), "carryout011", t)
	shouldBe(0, party.Reconstruct("carryout100"), "carryout100", t)
	shouldBe(1, party.Reconstruct("carryout101"), "carryout101", t)
	shouldBe(1, party.Reconstruct("carryout110"), "carryout110", t)
	shouldBe(1, party.Reconstruct("carryout111"), "carryout111", t)
	shouldBe(0, party.Reconstruct("add000"), "add000", t)
	shouldBe(1, party.Reconstruct("add001"), "add001", t)
	shouldBe(1, party.Reconstruct("add010"), "add010", t)
	shouldBe(0, party.Reconstruct("add011"), "add011", t)
	shouldBe(1, party.Reconstruct("add100"), "add100", t)
	shouldBe(0, party.Reconstruct("add101"), "add101", t)
	shouldBe(0, party.Reconstruct("add110"), "add110", t)
	shouldBe(1, party.Reconstruct("add111"), "add111", t)
}

func shouldBe(target int64, val *big.Int, desc string, t *testing.T) {
	if val.Cmp(big.NewInt(target)) != 0 {
		t.Error(desc, "Should be", target, "was", val)
	}
}

func bitIDs(val *big.Int, prime int64) []string {
	res := make([]string, big.NewInt(prime).BitLen()+1)
	for i := range res {
		if val.Bit(i) == 0 {
			res[i] = "0"
		} else {
			res[i] = "1"
		}
	}
	return res
}

func TestBitCompare(t *testing.T) {
	var prime int64 = 4001
	parties := setting(prime, 1, 3)

	parties[1].Share(big.NewInt(0), "0")
	parties[2].Share(big.NewInt(1), "1")
	var bitIDReps [][]string
	for i := 0; i < 13; i++ {
		bitIDReps =
			append(bitIDReps, bitIDs(big.NewInt(int64(i)), prime))
	}
	var tests []string
	var testResults []int64
	for _, party := range parties {
		for i := range bitIDReps {
			for j := range bitIDReps {
				id := strconv.Itoa(i) + " > " + strconv.Itoa(j)
				tests = append(tests, id)
				if i > j {
					testResults = append(testResults, 1)
				} else {
					testResults = append(testResults, 0)
				}
				go party.bitCompare(bitIDReps[i], bitIDReps[j], id)
				go party.Open(id)
			}
		}
	}

	party := parties[3]
	for i := range tests {
		shouldBe(testResults[i], party.Reconstruct(tests[i]), tests[i], t)

	}
}

func TestBitSub(t *testing.T) {
	prime := int64(4001)
	parties := setting(prime, 1, 3)

	iterations := 7

	parties[1].Share(big.NewInt(0), "0")
	parties[2].Share(big.NewInt(1), "1")
	for i := 0; i < iterations; i++ {
		for j := 0; j < iterations; j++ {
			iVal := big.NewInt(int64(i))
			jVal := big.NewInt(int64(j))
			iPlusJVal := big.NewInt(int64(i - j))
			iBits := bitIDs(iVal, prime)
			jBits := bitIDs(jVal, prime)
			// iPlusJBits := bitIDs(iPlusJVal, prime)
			iMinusJResultsBits := make([]string, len(iBits))
			for k := range iBits {
				iMinusJResultsBits[k] =
					strconv.Itoa(i) + " - " +
						strconv.Itoa(j) + "_index_" + strconv.Itoa(k)
			}
			for _, party := range parties {
				go party.bitSub(iBits, jBits, iMinusJResultsBits)
				for _, id := range iMinusJResultsBits {
					go party.Open(id)
				}
			}

			for bitIndex := range iMinusJResultsBits {
				shouldBe(
					int64(iPlusJVal.Bit(bitIndex)),
					parties[1].Reconstruct(iMinusJResultsBits[bitIndex]),
					iMinusJResultsBits[bitIndex], t)
			}
		}
	}

}

func TestBits(t *testing.T) {
	parties := setting(4001, 1, 3)

	for i := 0; i < 17; i++ {
		input := big.NewInt(int64(i))
		test := input.String()
		parties[1].Share(input, test)

		resultBitsIDs := make([]string, parties[1].prime.BitLen()+1)
		for bitIndex := range resultBitsIDs {
			resultBitsIDs[bitIndex] = test + "_index_" + strconv.Itoa(bitIndex)
		}

		for _, party := range parties {
			go party.bits(test, resultBitsIDs)
			for bitIndex := range resultBitsIDs {
				go party.Open(resultBitsIDs[bitIndex])
			}
		}

		for bitIndex := range resultBitsIDs {
			//TODO fails
			shouldBe(int64(input.Bit(bitIndex)),
				parties[1].Reconstruct(resultBitsIDs[bitIndex]), resultBitsIDs[bitIndex], t)
		}
	}

}
