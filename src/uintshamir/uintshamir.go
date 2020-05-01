package uintshamir

import (
	"fmt"
	"crypto/rand"
	"math/big"
)

//The SecretSharingScheme defines the parameters for the Shamir secret sharing
type SecretSharingScheme struct {
	p int64
	t int
	n int
}

var standardSetting = SecretSharingScheme { p : 5, t : 2,	n : 3}
var wikipediaExample = SecretSharingScheme { p : 11, t : 2,	n : 5}

type point = struct {
	x, y int64
}

type polynomial = []int64

//Share splits a secret into shares
func (ss *SecretSharingScheme) Share(secret int64) []point { //todo probably returns array not slice?
	secret = secret % ss.p

	fmt.Println("Creating", ss.n, "shares")
	var h = polynomial{secret}

	for coefficient := 1; coefficient <= ss.t; coefficient++ {
		randBigInt, _ := rand.Int(rand.Reader, big.NewInt( int64(ss.p)  ))
		h = append(h, randBigInt.Int64())
	}

	fmt.Println(h)

	var shares []point

	for i := 1; i <= ss.n; i++ {//Todo optimize and make overflow safe
		//Evaluate h in "i" and add share (i, h(i))
		y := secret
		x := int64(i)
		xpower := int64(i)
		for coefficient := 1; coefficient <= ss.t; coefficient++ {
			// fmt.Println(i, y)
			y += h[coefficient] * xpower
			xpower = xpower * x
		}
		// fmt.Println(i, y)
		fmt.Println(i, y % ss.p)
		shares = append(shares, point{x: x, y : y % ss.p})
	}

	return shares
}

//Reconstruct extracts the secret from t or more shares
func (ss *SecretSharingScheme) Reconstruct(shares []point) int64{
	k := len(shares)
	if k < ss.t + 1 {
		fmt.Println("Not enough shares to reconstruct")
		return 0
	}
	//todo check k distinct x's


	
	var nums []int64
	var dens []int64
	for _, share := range(shares) {
		fmt.Println("Coordinates", share.x, share.y)
		num := int64(1)
		den := int64(1)
		for _, otherShare := range(shares) {
			if share.x == otherShare.x { continue}
			num = num * (0 - otherShare.x) 
			den = den * (share.x - otherShare.x)
		}
		nums = append(nums, num)
		dens = append(dens, den)
		
	}

	den := int64(1)
	for _, d := range(dens) {den *= d}

	num := int64(0)
	for i := range(dens) {
		num += divideMod(nums[i] * den * shares[i].y % ss.p, dens[i], ss.p)
	}

	return (divideMod(num, den, ss.p) + ss.p) % ss.p

	// secret := int64(0)
	// for _, share := range(shares) {
	// 	fmt.Println("Coordinates", share.x, share.y)
	// 	num := int64(1)
	// 	den := int64(1)
	// 	for _, otherShare := range(shares) {
	// 		if share.x == otherShare.x { continue}
	// 		num = num * (otherShare.x)
	// 		den = den * (otherShare.x - share.x)
	// 	}
	// 	//num*(den)^-1= delta_i
	// 	secret += share.y * num * multiplicativeInverse(den, ss.p)
	// }


	// return secret % ss.p

}

//**********Number theory:
func extEuclid(a, b int64) (int64, int64) {
	//fmt.Println("a", a, "b", b)
	x := int64(0)
	lastX := int64(1)
	y := int64(1)
	lastY := int64(0)
	for b != 0 {
		quot := a / b
		//fmt.Println(quot)
		a, b = b, a % b
		//fmt.Println("a", a, "b", b)
		x, lastX = lastX - quot * x, x
		y, lastY = lastY - quot * y, y
	}

	//lastX is multiplicative inverse of a mod b
	return lastX, lastY
}

func extGcd(a, b int64) (int64, int64, int64) {
	if a == 0 {
		return b, 0, 1
	}

	//gcd, x, y := extGcd(b % a, a)

	return 0, 1, 2
}

func divideMod(dividend, divisor, p int64) int64 {
	divisorInv, _ := extEuclid(mod(divisor, p), p)
	//fmt.Println(divisorInv)
	//for divisorInv < 0 {divisorInv += int64(mod)}

	return (dividend * divisorInv) //% mod
}

func multiplicativeInverse(element, mod int64) int64 {
	inv, _ := extEuclid(element % mod, mod)
	for inv < 0 {inv += mod}
	return int64(inv) % mod
}

func lagrangeInterpolationAtZero(points []point, prime int64) int64 {
	res := int64(0)
	for _, pointI := range(points) {
		num := int64(1)
		den := int64(1)
		for _, pointJ := range(points) {
			if pointI.x != pointJ.x {
				num = (num * pointJ.x)// % prime
				den = (den * (pointJ.x - pointI.x))// % prime
			}
		}
		//fmt.Println("num", num, "den", den)


		// res = (res + (pointI.y * num * multiplicativeInverse(den, prime)))// % prime
		delta := divideMod(num, den, prime)
		//fmt.Println("delta", delta)
		res = (res + (pointI.y * delta))// % prime
		//fmt.Println(res )
	}
	return mod(res, prime)
}

func mod(a, b int64) int64 {
	res := a % b
	if res < 0 {
		return res + b
	}
	return res
}