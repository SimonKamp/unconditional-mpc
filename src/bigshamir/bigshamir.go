package bigshamir

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
)

//Point represents a point on a polynomial, i.e. the share of party X
type Point struct {
	X int
	Y *big.Int
}

func newPoint(X, Y int) Point {
	return Point{X: X, Y: big.NewInt(int64(Y))}
}

func (p Point) String() string {
	return "(" + strconv.FormatInt(int64(p.X), 10) + "," + p.Y.String() + ")"
}

type polynomial []*big.Int

//SecretSharingScheme ...
type SecretSharingScheme struct {
	p *big.Int
	threshold int
	n int
}

//NewSS constructs a secret sharing scheme with initialised reconstruction vector
func NewSS(p int64, threshold, n int) SecretSharingScheme {
	ss := SecretSharingScheme{}
	ss.p = big.NewInt(p)
	ss.n = n
	ss.threshold = threshold
	return ss
}

var standardSetting = SecretSharingScheme { p : big.NewInt(5), threshold : 1, n : 3}

//Share splits a secret into shares (points)
func (ss *SecretSharingScheme) Share(secret *big.Int) []Point {
	//Draw random polynomial h
	h := make([]*big.Int, ss.threshold + 1)
	h[0] = secret
	for coefficient := 1; coefficient <= ss.threshold; coefficient++ {
		randBigInt, _ := rand.Int(rand.Reader, ss.p )
		h[coefficient] = randBigInt
	}

	//Evaluate points on h
	shares := make([]Point, ss.n)
	for i := 1; i <= ss.n; i++ {
		shares[i-1] = Point{
			X: i, 
			Y: hornersEvaluatePolynomialAt(h, int64(i), ss.p)}
	}

	return shares
}

//Reconstruct extracts the secret from threshold or more shares
func (ss *SecretSharingScheme)Reconstruct(shares []Point) *big.Int {
	k := len(shares)

	if k < ss.threshold + 1 {
		fmt.Println("Not enough shares to reconstruct")
		return big.NewInt(0)
	}

	return lagrangeInterpolationAtZero(shares, ss.p)
}

//Add creates a secret sharing of the sum of two secret shared values 
//Slices should have same indicies in same order
//Can panic
//Todo: implement sorting?
func (ss *SecretSharingScheme)Add(aShares, bShares []Point) []Point {
	aPlusBShares := make([]Point, len(aShares))
	
	for i := range(aShares) {
		X := aShares[i].X
		Y := new(big.Int).Add(aShares[i].Y, bShares[i].Y)
		aPlusBShares[i] = Point{X : X, Y : Y}
	}

	return aPlusBShares
}

//Scale multiplies a secret sharing by a constant
func (ss *SecretSharingScheme)Scale(scalar *big.Int, shares []Point) []Point {
	scaledShares := make([]Point, len(shares))

	for _, share := range(shares) {
		scaled := new(big.Int).Mul(scalar, share.Y)
		scaledShares[share.X - 1] = Point{X : share.X, Y : scaled}
	}

	return scaledShares
}

//Mul creates a secret sharing of the product of two secret shared values
//slices should have same indicies in same order
//Can panic
//Todo: implement sorting?
func (ss *SecretSharingScheme)Mul(aShares, bShares []Point) []Point {
	if len(aShares) != ss.n || len(bShares) != ss.n {
		panic("Missing shares")
	}

	//Step 1: Each party locally computes the product of its two shares
	aTimesBShares := make([]Point, len(aShares))
	for party := range(aShares) {
		X := aShares[party].X
		Y := new(big.Int).Mul(aShares[party].Y, bShares[party].Y)
		aTimesBShares[party] = Point{X : X, Y : Y.Mod(Y, ss.p)}
	}

	//Step 2:Each P_i distributes [h(i);f_i]_t
	var shareShares [][]Point = make([][]Point, ss.n)
	for party, share := range(aTimesBShares) {
		shareShares[party] = ss.Share(share.Y)
	}

	//Step 3: Create degree threshold sharing

	//todo clean up
	r := reconstructionVectorFromPoints(aShares, ss.p)
	shares := make([]Point, ss.n)
	for party := range(shares) {
		sum := big.NewInt(0)
		for i, share := range(aShares) {
			ri := r[share.X]
			hishare := shareShares[i][party]
			product := new(big.Int).Mul(ri, hishare.Y)
			sum.Add(sum, product)
		}

		shares[party] = Point{
			X: party + 1,
			Y: sum.Mod(sum, ss.p),
		}
	}
	
	return shares
}

func evaluatePolynomialAt(p polynomial, X int64, prime *big.Int) *big.Int {
	if len(p) == 0 { return big.NewInt(0)}
	bigX := big.NewInt(X)
	xPower := big.NewInt(X)
	sum := new(big.Int).Set(p[0])
	for i := 1; i < len(p); i++ {
		term := new(big.Int).Mul(p[i], xPower)
		sum.Add(sum, term)
		xPower.Mul(xPower, bigX)
	}
	return sum.Mod(sum, prime)
}

func hornersEvaluatePolynomialAt(p polynomial, X int64, prime *big.Int) *big.Int {
	bigX := big.NewInt(X)
	res := big.NewInt(0)
	for i := len(p) - 1; i >= 0; i-- {
		res.Mul(res, bigX)
		res.Add(res, p[i])
		// res.Mod(res, prime) //todo benchmark, with and without
	}
	return res.Mod(res, prime)
}


func lagrangeInterpolationAtZero(points []Point, prime *big.Int) *big.Int {
	r := reconstructionVectorFromPoints(points, prime)
	
	sum := big.NewInt(0)
	for _, share := range(points) {
		tmp := new(big.Int)
		tmp.Mul(share.Y, r[share.X])//y_i*delta_i(0)
		sum = sum.Add(sum, tmp)
	}
	
	return sum.Mod(sum, prime)
}

func reconstructionVectorFromPoints(points []Point, prime *big.Int) map[int]*big.Int {
	m := make([]int, len(points))
	for i, p := range(points) {
		m[i] = p.X
	}
	return reconstructionVector(prime, m...)
}

func reconstructionVector(prime *big.Int, xs ...int) map[int]*big.Int {
	terms := make(map[int]*big.Int)
	for _, i := range(xs) {
		num := 1
		den := 1
		for _, j := range(xs) {
			if i == j {continue}
			num *= j
			den *= j - i
		}
		term := big.NewInt(int64(den)) //den
		term.ModInverse(term, prime) //den^-1
		term.Mul(term, big.NewInt(int64(num))) //num*den^-1
		term.Mod(term, prime)
		terms[i] = term
	}

	return terms
}
