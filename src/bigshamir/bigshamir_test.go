package bigshamir

import (
	"math/big"
	"testing"
)

func TestShare(t *testing.T) {
	setting := NewSS(11, 1, 3)
	shares := setting.Share(big.NewInt(7))
	reconstructed := setting.Reconstruct(shares)
	if reconstructed.Cmp(big.NewInt(7)) != 0 {
		t.Errorf("Share + reconstruct failed")
	}
}

func TestEvaluatePolynomialAt(t *testing.T) {
	p := polynomial{big.NewInt(7), big.NewInt(3), big.NewInt(2)}
	prime := big.NewInt(11)

	fun := func(x, y int64) {
		res := evaluatePolynomialAt(p, x, prime)
		if res.Cmp(big.NewInt(y)) != 0 {
			t.Errorf("Expected f(%d) = %d. Got %d", x, y, res)
		}
	}

	fun(0, 7)
	fun(1, 1)  //7 + 3 + 2 mod 11 = 1
	fun(2, 10) //7 + 3 * 2 + 2 * 2 ^ 2 mod 11 = 21 mod 11 = 10
	fun(3, 1)  //7 + 3 * 3 + 2 * 3 ^ 2 mod 11 = 34 mod 11 = 1
}

func TestHornersEvaluatePolynomialAt(t *testing.T) {
	p := polynomial{big.NewInt(7), big.NewInt(3), big.NewInt(2)}
	prime := big.NewInt(11)

	fun := func(x, y int64) {
		res := hornersEvaluatePolynomialAt(p, x, prime)
		if res.Cmp(big.NewInt(y)) != 0 {
			t.Errorf("Expected f(%d) = %d. Got %d", x, y, res)
		}
	}

	fun(0, 7)
	fun(1, 1)  //7 + 3 + 2 mod 11 = 1
	fun(2, 10) //7 + 3 * 2 + 2 * 2 ^ 2 mod 11 = 21 mod 11 = 10
	fun(3, 1)  //7 + 3 * 3 + 2 * 3 ^ 2 mod 11 = 34 mod 11 = 1
}

func TestLagrangeInterpolationAtZero(t *testing.T) {
	shares := []Point{
		newPoint(1, 1),
		newPoint(2, 8),
		newPoint(3, 6),
	}
	zeroValue := lagrangeInterpolationAtZero(shares, big.NewInt(11))
	if zeroValue.Cmp(big.NewInt(7)) != 0 {
		t.Errorf(zeroValue.String())
	}
}

func TestAdd(t *testing.T) {
	setting := NewSS(11, 1, 3)
	aShares := setting.Share(big.NewInt(7))
	bShares := setting.Share(big.NewInt(9))
	aPlusBShares := setting.Add(aShares, bShares)
	reconstructed := setting.Reconstruct(aPlusBShares)
	if reconstructed.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("Share two values and Add then reconstruct failed")
	}
}

func TestScale(t *testing.T) {
	setting := NewSS(11, 1, 3)
	shares := setting.Share(big.NewInt(7))
	scaled := setting.Scale(big.NewInt(2), shares)
	reconstructed := setting.Reconstruct(scaled)
	if reconstructed.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("Share and scale then reconstruct failed")
	}
}

func TestMul(t *testing.T) {
	setting := NewSS(11, 1, 3)
	fun := func(aShares, bShares []Point) []Point {
		aTimesBShares := setting.Mul(aShares, bShares)
		a := setting.Reconstruct(aShares)
		b := setting.Reconstruct(bShares)
		aTimesB := new(big.Int).Mul(a, b)
		aTimesB.Mod(aTimesB, setting.p)
		reconstructed := setting.Reconstruct(aTimesBShares)
		if aTimesB.Cmp(reconstructed) != 0 {
			t.Errorf("Share two values and Mul then reconstruct failed. Expected %d, got %d.", aTimesB, reconstructed)
		}
		return aTimesBShares
	}
	aShares := setting.Share(big.NewInt(7))
	bShares := setting.Share(big.NewInt(9))
	aTimesBShares := fun(aShares, bShares)
	aSquaredShares := fun(aShares, aShares)
	bSquaredShares := fun(bShares, bShares)

	//Following fails if no reduction of polynomial degree is done during multiplication
	fun(aSquaredShares, aSquaredShares) //7^4 % 11 = 3
	fun(bSquaredShares, bSquaredShares) //9^4 % 11 = 5
	fun(aSquaredShares, bSquaredShares) //7^2 * 9^2 % 11 = 9
	fun(aTimesBShares, aTimesBShares) //7^2 * 9^2 % 11 = 9
}


func TestReconstructionVector(t *testing.T) {
	r := ReconstructionVector(big.NewInt(11), 3, 4, 5)
	test := func(index int, rIndex int64) {
		rI, contains := r[index]
		if !contains {t.Errorf("Does not contain r%d", index)}
		if rI.Int64() != rIndex {t.Errorf("r%d should be %d was %d", index, rIndex, rI.Int64())}
	}
	test(3, 10)
	test(4, 7)
	test(5, 6)
}