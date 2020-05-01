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
	reconstructionVector []*big.Int
}

//NewSS constructs a secret sharing scheme with initialised reconstruction vector
func NewSS(p int64, n, t int) SecretSharingScheme {
	ss := SecretSharingScheme{}
	ss.p = big.NewInt(p)
	ss.n = n
	ss.t = t
	ss.reconstructionVector = reconstructionVector(int64(n), ss.p)
	return ss
}

var standardSetting = SecretSharingScheme { p : big.NewInt(5), t : 1, n : 3}

//Share splits a secret into shares (points)
func (ss *SecretSharingScheme) Share(secret *big.Int) []Point {
	//Draw random polynomial h
	h := polynomial{secret}
	for coefficient := 1; coefficient <= ss.t; coefficient++ {
		randBigInt, _ := rand.Int(rand.Reader, ss.p )
		h = append(h, randBigInt)
	}
	//Evaluate points on h
	var shares []Point
	for i := 1; i <= ss.n; i++ {
		shares = append(
			shares, 
			Point{
				x: int64(i), 
				y: hornersEvaluatePolynomialAt(h, int64(i), ss.p)})
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
	var aPlusBShares []Point

	for i := range(aShares) {
		x := aShares[i].x
		y := new(big.Int).Add(aShares[i].y, bShares[i].y)
		aPlusBShares = append(aPlusBShares, Point{x : x, y : y})
	}

	return aPlusBShares
}

//Scale multiplies a secret sharing by a constant
func (ss *SecretSharingScheme)Scale(scalar *big.Int, shares []Point) []Point {
	var scaledShares []Point
	for _, share := range(shares) {
		scaled := new(big.Int).Mul(scalar, share.y)
		scaledShares = append(scaledShares, Point{x : share.x, y : scaled})
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
	var aTimesBShares []Point
	for i := range(aShares) {
		x := aShares[i].x
		y := new(big.Int).Mul(aShares[i].y, bShares[i].y)//Todo assumes sorted in same order
		aTimesBShares = append(aTimesBShares, Point{x : x, y : y})
	}
	fmt.Println(aTimesBShares)

	//Step 2:Each P_i distributes [h(i);f_i]_t
	var shareShares [][]Point = make([][]Point, ss.n)
	for _, share := range(aTimesBShares) {
		shareShares[share.x - 1] = ss.Share(share.y)
	}

	//todo: Step 3: Create degree t sharing
	//Efficently compute reconstruction vector?

	return aTimesBShares
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
	sum := big.NewInt(0)
	for _, share := range(points) {
		num := int64(1)
		den := int64(1)
		for _, otherShare := range(points) {
			if share.x == otherShare.x { continue}
			num *= otherShare.x
			den *= (otherShare.x - share.x)
		}
		tmp := big.NewInt(den)
		tmp.ModInverse(tmp, prime)//den^-1
		tmp.Mul(big.NewInt(num), tmp)//delta_i(0)=num*den^-1
		tmp.Mul(share.y, tmp)//y_i*delta_i(0)
		sum = sum.Add(sum, tmp)
	}
	
	return sum.Mod(sum, prime)
}

func reconstructionVector(n int64, prime *big.Int) []*big.Int {
	terms := make([]*big.Int, n)
	product := big.NewInt(1)
	for i := int64(1); i <= n; i++ {
		num := int64(1)
		den := int64(1)
		for j := int64(1); j <= n; j++ {
			if i == j {continue}
			num *= j
			den *= j - i
		}
		term := big.NewInt(den) //den
		term.ModInverse(term, prime) //den^-1
		term.Mul(term, big.NewInt(num)) //num*den^-1
		product.Mul(product, term)
		terms[i-1] = term
	}
	product.Mod(product, prime)
	for i := int64(0); i < n; i++ {
		terms[i].Mul(terms[i].ModInverse(terms[i], prime), product)
	}

	return terms
}

func (ss *SecretSharingScheme)r(i int) *big.Int {
	//todo check and construct if not set?
	return ss.reconstructionVector[i-1]
}