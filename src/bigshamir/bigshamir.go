package bigshamir

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
)

//Point represents a point on a polynomial, i.e. the share of party x
type Point struct {
	x int64
	y *big.Int
}

func newPoint(x, y int) Point {
	return Point{x: int64(x), y: big.NewInt(int64(y))}
}

func (p Point) String() string {
	return "(" + strconv.FormatInt(p.x, 10) + "," + p.y.String() + ")"
}

type polynomial []*big.Int

//SecretSharingScheme ...
type SecretSharingScheme struct {
	p *big.Int
	t int
	n int
}

//NewSS constructs a secret sharing scheme with initialised reconstruction vector
func NewSS(p int64, n, t int) SecretSharingScheme {
	ss := SecretSharingScheme{}
	ss.p = big.NewInt(p)
	ss.n = n
	ss.t = t
	return ss
}

var standardSetting = SecretSharingScheme { p : big.NewInt(5), t : 1, n : 3}

//Share splits a secret into shares (points)
func (ss *SecretSharingScheme) Share(secret *big.Int) []Point {
	//Draw random polynomial h
	h := make([]*big.Int, ss.t + 1)
	h[0] = secret
	for coefficient := 1; coefficient <= ss.t; coefficient++ {
		randBigInt, _ := rand.Int(rand.Reader, ss.p )
		h[coefficient] = randBigInt
	}

	//Evaluate points on h
	shares := make([]Point, ss.n)
	for i := 1; i <= ss.n; i++ {
		shares[i-1] = Point{
			x: int64(i), 
			y: hornersEvaluatePolynomialAt(h, int64(i), ss.p)}
	}

	return shares
}

//Reconstruct extracts the secret from t or more shares
func (ss *SecretSharingScheme)Reconstruct(shares []Point) *big.Int {
	k := len(shares)

	if k < ss.t + 1 {
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
		x := aShares[i].x
		y := new(big.Int).Add(aShares[i].y, bShares[i].y)
		aPlusBShares[i] = Point{x : x, y : y}
	}

	return aPlusBShares
}

//Scale multiplies a secret sharing by a constant
func (ss *SecretSharingScheme)Scale(scalar *big.Int, shares []Point) []Point {
	scaledShares := make([]Point, len(shares))

	for _, share := range(shares) {
		scaled := new(big.Int).Mul(scalar, share.y)
		scaledShares[share.x - 1] = Point{x : share.x, y : scaled}
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
		x := aShares[party].x
		y := new(big.Int).Mul(aShares[party].y, bShares[party].y)
		aTimesBShares[party] = Point{x : x, y : y.Mod(y, ss.p)}
	}

	//Step 2:Each P_i distributes [h(i);f_i]_t
	var shareShares [][]Point = make([][]Point, ss.n)
	for party, share := range(aTimesBShares) {
		shareShares[party] = ss.Share(share.y)
	}

	//Step 3: Create degree t sharing

	//todo clean up
	r := reconstructionVectorFromPoints(aShares, ss.p)
	shares := make([]Point, ss.n)
	for party := range(shares) {
		sum := big.NewInt(0)
		for i, share := range(aShares) {
			ri := r[share.x]
			hishare := shareShares[i][party]
			product := new(big.Int).Mul(ri, hishare.y)
			sum.Add(sum, product)
		}

		shares[party] = Point{
			x: int64(party + 1),
			y: sum.Mod(sum, ss.p),
		}
	}
	
	return shares
}

func evaluatePolynomialAt(p polynomial, x int64, prime *big.Int) *big.Int {
	if len(p) == 0 { return big.NewInt(0)}
	bigX := big.NewInt(x)
	xPower := big.NewInt(x)
	sum := new(big.Int).Set(p[0])
	for i := 1; i < len(p); i++ {
		term := new(big.Int).Mul(p[i], xPower)
		sum.Add(sum, term)
		xPower.Mul(xPower, bigX)
	}
	return sum.Mod(sum, prime)
}

func hornersEvaluatePolynomialAt(p polynomial, x int64, prime *big.Int) *big.Int {
	bigX := big.NewInt(x)
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
		tmp.Mul(share.y, r[share.x])//y_i*delta_i(0)
		sum = sum.Add(sum, tmp)
	}
	
	return sum.Mod(sum, prime)
}

func reconstructionVectorFromPoints(points []Point, prime *big.Int) map[int64]*big.Int {
	m := make([]int64, len(points))
	for i, p := range(points) {
		m[i] = p.x
	}
	return reconstructionVector(prime, m...)
}

func reconstructionVector(prime *big.Int, xs ...int64) map[int64]*big.Int {
	terms := make(map[int64]*big.Int)
	for _, i := range(xs) {
		num := int64(1)
		den := int64(1)
		for _, j := range(xs) {
			if i == j {continue}
			num *= j
			den *= j - i
		}
		term := big.NewInt(den) //den
		term.ModInverse(term, prime) //den^-1
		term.Mul(term, big.NewInt(num)) //num*den^-1
		term.Mod(term, prime)
		terms[i] = term
	}

	return terms
}
