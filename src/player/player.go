package player

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"../bigshamir"
	"../network"
)

//Player runs the protocol
type Player struct {
	//Fixed after setup:
	prime        *big.Int
	threshold    int
	n            int
	index        int
	l            int
	ss           *bigshamir.SecretSharingScheme
	network      network.Network
	inputValues  map[string]*big.Int
	instructions []instruction

	//Concurrently accessed:
	//Regular shares
	shareLock            sync.RWMutex
	shares               map[string]*big.Int //todo consider using sync.Map
	reconstructionShares map[string][]bigshamir.SecretShare

	//Recombination shares for multiplication
	multShares map[string][]multiplicationShare

	//Random bit shares
	randomBitLock           sync.RWMutex
	randFieldElemShares     map[string][]localRandomFieldElementShare
	randomBitASquaredShares map[string][]bigshamir.SecretShare
}

type (
	identifiedShare struct {
		point bigshamir.SecretShare
		id    string
	}
	multiplicationShare struct {
		recombinationShare bigshamir.RecombinationShare
		id                 string
	}
	localRandomFieldElementShare struct {
		point     bigshamir.SecretShare
		id        string
		index     int
		iteration int
	}
	aSquaredShare struct {
		point     bigshamir.SecretShare
		id        string
		iteration int
	}
)

//NewPlayer ...
func NewPlayer(prime int64, threshold, n, index int) *Player {
	p := new(Player)
	p.prime = big.NewInt(prime)
	p.threshold = threshold
	p.n = n
	p.l = p.prime.BitLen()
	p.index = index
	p.ss = new(bigshamir.SecretSharingScheme)
	*p.ss = bigshamir.NewSS(prime, threshold, n)
	p.shares = make(map[string]*big.Int)
	p.reconstructionShares = make(map[string][]bigshamir.SecretShare)
	p.multShares = make(map[string][]multiplicationShare)
	p.randFieldElemShares = make(map[string][]localRandomFieldElementShare)
	p.randomBitASquaredShares = make(map[string][]bigshamir.SecretShare)
	p.inputValues = make(map[string]*big.Int)
	return p
}

//Share ...
func (p *Player) Share(x *big.Int, identifier string) {
	points := p.ss.Share(x)
	for _, point := range points {
		p.Send(identifiedShare{point: point, id: identifier}, point.X)
	}
}

//Open ...
func (p *Player) Open(identifier string) {
	yValue := p.getShareValue(identifier)
	share := identifiedShare{
		point: bigshamir.SecretShare{
			X: p.index,
			Y: yValue,
		},
		id: identifier,
	}
	for i := 1; i <= p.n; i++ {
		p.Send(share, i)
	}
}

//Reconstruct ...
func (p *Player) Reconstruct(identifier string) *big.Int {
	p.shareLock.RLock()
	points := p.reconstructionShares[identifier]
	p.shareLock.RUnlock()
	for len(points) < p.threshold+1 {
		time.Sleep(time.Millisecond)
		p.shareLock.RLock()
		points = p.reconstructionShares[identifier]
		p.shareLock.RUnlock()
	}
	return p.ss.Reconstruct(points)
}

func (p *Player) getShareValue(identifier string) *big.Int {
	p.shareLock.RLock()
	value, exists := p.shares[identifier]
	p.shareLock.RUnlock()
	for !exists {
		time.Sleep(time.Millisecond)
		p.shareLock.RLock()
		value, exists = p.shares[identifier]
		p.shareLock.RUnlock()
	}
	return value
}

//Add ...
func (p *Player) Add(aIdentifier, bIdentifier, cIdentifier string) {
	sum := new(big.Int)
	sum.Add(p.getShareValue(aIdentifier), p.getShareValue(bIdentifier))
	sum.Mod(sum, p.prime)
	p.shareLock.Lock()
	p.shares[cIdentifier] = sum
	p.shareLock.Unlock()
}

//AddOpen ...
func (p *Player) AddOpen(aShare *big.Int, bIdentifier, cIdentifier string) {
	sum := new(big.Int)
	sum.Add(aShare, p.getShareValue(bIdentifier))
	sum.Mod(sum, p.prime)
	p.shareLock.Lock()
	p.shares[cIdentifier] = sum
	p.shareLock.Unlock()
}

//Sub ...
func (p *Player) Sub(aIdentifier, bIdentifier, cIdentifier string) {
	sum := new(big.Int)
	sum.Sub(p.getShareValue(aIdentifier), p.getShareValue(bIdentifier))
	sum.Mod(sum, p.prime)
	p.shareLock.Lock()
	p.shares[cIdentifier] = sum
	p.shareLock.Unlock()
}

//SubFromOpen ...
func (p *Player) SubFromOpen(aShare *big.Int, bID, cID string) {
	res := new(big.Int)
	res.Sub(aShare, p.getShareValue(bID))
	res.Mod(res, p.prime)
	p.shareLock.Lock()
	p.shares[cID] = res
	p.shareLock.Unlock()
}

//SubOpen ...
func (p *Player) SubOpen(aId string, bShare *big.Int, cId string) {
	res := new(big.Int)
	res.Add(p.getShareValue(aId), bShare)
	res.Mod(res, p.prime)
	p.shareLock.Lock()
	p.shares[cId] = res
	p.shareLock.Unlock()
}

//Scale ...
func (p *Player) Scale(scalar *big.Int, aIdentifier, bIdentifier string) {
	sum := new(big.Int)
	sum.Mul(p.getShareValue(aIdentifier), scalar)
	sum.Mod(sum, p.prime)
	p.shareLock.Lock()
	p.shares[bIdentifier] = sum
	p.shareLock.Unlock()
}

//Multiply ...
func (p *Player) Multiply(aIdentifier, bIdentifier, cIdentifier string) {
	//Compute and secret share product of local shares
	localProduct := new(big.Int)
	localProduct.Mul(p.getShareValue(aIdentifier), p.getShareValue(bIdentifier))
	for _, share := range p.ss.Share(localProduct) {
		ms := multiplicationShare{
			recombinationShare: bigshamir.RecombinationShare{
				SecretShare: share,
				Index:       p.index,
			},
			id: cIdentifier,
		}
		go p.Send(ms, share.X)
	}

	//Wait for 2t+1 sharings of local products
	p.shareLock.RLock()
	shares := p.multShares[cIdentifier]
	p.shareLock.RUnlock()
	for shares == nil || len(shares) < p.threshold*2+1 {
		time.Sleep(time.Millisecond)
		p.shareLock.RLock()
		shares = p.multShares[cIdentifier]
		p.shareLock.RUnlock()
	}

	multShares := make([]bigshamir.RecombinationShare, len(shares))
	for i := range shares {
		multShares[i] = shares[i].recombinationShare
	}
	cShareValue := p.ss.RecombineMultiplicationShares(multShares)

	p.shareLock.Lock()
	p.shares[cIdentifier] = cShareValue
	p.shareLock.Unlock()
}

//Compare takes shares of aShare and b as input and outputs 1 iff aShare > b
func (p *Player) Compare(aID, bID, cID string) {
	//compute sharings of bits of aShare and b:
	aBitIDs := make([]string, p.l+1)
	bBitIDs := make([]string, p.l+1)
	for i := range aBitIDs {
		aBitIDs[i] = cID + "=" + aID + ">" + bID + "_:_bits_of_" + aID + "_index_" + strconv.Itoa(i)
		bBitIDs[i] = cID + "=" + aID + ">" + bID + "_:_bits_of_" + bID + "_index_" + strconv.Itoa(i)
	}
	p.bits(aID, aBitIDs)
	p.bits(bID, bBitIDs)
	fmt.Println("a bits", aBitIDs)
	fmt.Println("b bits", bBitIDs)
	//p.bitCompare(aBitIDs, bBitIDs, cID)
}

func (p *Player) bits(ID string, resultBitIDs []string) {
	rID, rBitIDs := p.randomSolvedBits(ID)
	p.Sub(ID, rID, resultBitIDs[0]+"_bits_c") //c = a - r
	p.Open(resultBitIDs[0] + "_bits_c")
	c := p.Reconstruct(resultBitIDs[0] + "_bits_c")

	//Compute bit sharing of sum of r and c, i.e. bit sharing of ID
	cBitIDs := make([]string, p.l+1)
	dBitIDs := make([]string, p.l+1)
	pBitIDs := make([]string, p.l+1)
	epBitIDs := make([]string, p.l+1)
	p.shareLock.Lock()
	for i := 0; i <= p.l; i++ {
		cBitIDs[i] = resultBitIDs[i] + "_bits_cBits"
		dBitIDs[i] = resultBitIDs[i] + "_bits_dBits"
		pBitIDs[i] = resultBitIDs[i] + "_bits_pBits"
		epBitIDs[i] = resultBitIDs[i] + "_bits_cepBits"
		p.shares[cBitIDs[i]] = big.NewInt(int64(c.Bit(i)))
		p.shares[pBitIDs[i]] = big.NewInt(int64(p.prime.Bit(i)))
	}
	p.shareLock.Unlock()

	p.bitAdd(rBitIDs, cBitIDs, dBitIDs)
	e := resultBitIDs[0] + "_bits_compareBits_e"
	p.bitCompare(dBitIDs, pBitIDs, e)
	for i := range epBitIDs {
		p.Multiply(e, pBitIDs[i], epBitIDs[i])
	}

	p.bitSub(dBitIDs, epBitIDs, resultBitIDs)

}

func (p *Player) bitCompare(aBitIDs, bBitIDs []string, cBitID string) {
	//Compute sharing of XOR
	xorShareIDs := make([]string, p.l+1)
	for i := 0; i <= p.l; i++ {
		xorShareIDs[i] = cBitID + "_bitCompare_xor" + strconv.Itoa(i)
		ithBitOfAShare := p.getShareValue(aBitIDs[i])
		ithBitOfBShare := p.getShareValue(bBitIDs[i])
		sum := new(big.Int).Add(ithBitOfAShare, ithBitOfBShare) //[a_i]+[b_i]
		multID := xorShareIDs[i] + "_tmp"
		p.Multiply(aBitIDs[i], bBitIDs[i], multID)
		xorProduct := p.getShareValue(multID)
		xorProduct.Mul(big.NewInt(2), xorProduct) //2[a_i][b_i]
		xorProduct.Mod(xorProduct, p.prime)
		sum.Sub(sum, xorProduct) //[a_i]+[b_i] - 2[a_i][b_i]
		sum.Mod(sum, p.prime)

		p.shareLock.Lock()
		p.shares[xorShareIDs[i]] = sum
		p.shareLock.Unlock()
	}
	dBitIDs := p.mostSignificant1(xorShareIDs)
	eBitIDs := make([]string, p.l+1)
	for i := range eBitIDs {
		eBitIDs[i] = cBitID + "_bitCompare_e" + strconv.Itoa(i)
		go p.Multiply(aBitIDs[i], dBitIDs[i], eBitIDs[i])
	}
	cShare := big.NewInt(0)
	for i := range eBitIDs {
		cShare.Add(cShare, p.getShareValue(eBitIDs[i]))
	}
	cShare.Mod(cShare, p.prime)

	p.shareLock.Lock()
	p.shares[cBitID] = cShare
	p.shareLock.Unlock()
}

func (p *Player) bitXor(aBitID, bBitID, cBitID string) *big.Int {
	p.Multiply(aBitID, bBitID, cBitID+"_xor_tmp")
	aShare := p.getShareValue(aBitID)
	bShare := p.getShareValue(bBitID)
	xorShare := p.getShareValue(cBitID + "_xor_tmp") //ab
	xorShare.Mul(big.NewInt(2), xorShare)            //2ab
	xorShare.Neg(xorShare)                           //-2ab
	xorShare.Add(xorShare, aShare)                   //a - 2ab
	xorShare.Add(xorShare, bShare)                   //a+b-2ab

	p.shareLock.Lock()
	delete(p.shares, cBitID+"_xor_tmp")
	p.shares[cBitID] = xorShare //todo get rid of this?
	p.shareLock.Unlock()

	return xorShare
}

func (p *Player) fullAdder(aBitID, bBitID, carryInBitID, carryOutBitID, cBitID string) {
	//carry_out = (a & b) | (a & carry_in) | (b & carry_in)
	//			= ! (!(a & b) & !(a & carry_in) & !(b & carry_in))
	//			= 1 - ((1 - a * b) * (1 - a * carry_in) * (1 - b * carry_in))

	p.Multiply(aBitID, bBitID, cBitID+"_a&b")
	p.Multiply(aBitID, carryInBitID, cBitID+"_a&carryIn")
	p.Multiply(bBitID, carryInBitID, cBitID+"_b&carryIn")

	ab := p.getShareValue(cBitID + "_a&b")
	aCarryIn := p.getShareValue(cBitID + "_a&carryIn")

	p.shareLock.Lock()
	p.shares[cBitID+"_!a&b"] = bitNot(ab)
	p.shares[cBitID+"_!a&carryIn"] = bitNot(aCarryIn)
	p.shareLock.Unlock()
	p.Multiply(cBitID+"_!a&b", cBitID+"_!a&carryIn", cBitID+"_!a&b_&_!a&carryIn")

	bCarryIn := p.getShareValue(cBitID + "_b&carryIn")
	p.shareLock.Lock()
	p.shares[cBitID+"_!b&carryIn"] = bitNot(bCarryIn)
	p.shareLock.Unlock()

	p.Multiply(cBitID+"_!a&b_&_!a&carryIn", cBitID+"_!b&carryIn", cBitID+"_disjunct")

	carryOutShare := bitNot(p.getShareValue(cBitID + "_disjunct"))
	carryOutShare.Mod(carryOutShare, p.prime)
	p.shareLock.Lock()
	p.shares[carryOutBitID] = carryOutShare
	p.shareLock.Unlock()

	//c = a + b + c - 2 * carry_out
	resBitShare := new(big.Int).Mul(big.NewInt(2), carryOutShare)
	resBitShare.Neg(resBitShare)
	resBitShare.Add(resBitShare, p.getShareValue(aBitID))
	resBitShare.Add(resBitShare, p.getShareValue(bBitID))
	resBitShare.Add(resBitShare, p.getShareValue(carryInBitID))
	resBitShare.Mod(resBitShare, p.prime)
	p.shareLock.Lock()
	p.shares[cBitID] = resBitShare
	p.shareLock.Unlock()
}

func bitNot(aBitShare *big.Int) *big.Int {
	return new(big.Int).Sub(big.NewInt(1), aBitShare)
}

func (p *Player) bitAdd(aBitIDs, bBitIDs, resBitIDs []string) {
	if len(aBitIDs) != len(bBitIDs) || len(aBitIDs) != len(resBitIDs) {
		panic("bit add different lengths")
		return
	}

	for i := range resBitIDs {
		if i == 0 {
			//Carry in = 0
			p.shareLock.Lock()
			p.shares[resBitIDs[i]+"_carryin0"] = big.NewInt(0)
			p.shareLock.Unlock()
		}
		carryInID := resBitIDs[i] + "_carryin" + strconv.Itoa(i)
		carryOutID := resBitIDs[i] + "_carryin" + strconv.Itoa(i+1)
		p.fullAdder(aBitIDs[i], bBitIDs[i], carryInID, carryOutID, resBitIDs[i])
	}

	//todo remove tmps?
}

func (p *Player) bitSub(aBitIDs, bBitIDs, resBitIDs []string) {
	if len(aBitIDs) != len(bBitIDs) || len(aBitIDs) != len(resBitIDs) {
		panic("bit add different lengths")
		return
	}
	flippedBBitIDs := make([]string, len(bBitIDs))
	for i := range flippedBBitIDs {
		flippedBBitIDs[i] = bBitIDs[i] + "_sub_flipped"
		flippedBit := bitNot(p.getShareValue(bBitIDs[i]))
		p.shareLock.Lock()
		p.shares[flippedBBitIDs[i]] = flippedBit
		p.shareLock.Unlock()
	}

	for i := range resBitIDs {
		if i == 0 {
			//Carry in = 1
			p.shareLock.Lock()
			p.shares[resBitIDs[i]+"_carryin0"] = big.NewInt(1)
			p.shareLock.Unlock()
		}
		carryInID := resBitIDs[i] + "_carryin" + strconv.Itoa(i)
		carryOutID := resBitIDs[i] + "_carryin" + strconv.Itoa(i+1)
		p.fullAdder(aBitIDs[i], flippedBBitIDs[i], carryInID, carryOutID, resBitIDs[i])
	}

	//todo remove tmps?
}

func (p *Player) randomSolvedBits(identifier string) (fieldElemID string, bitIDs []string) {
	fieldElemID = identifier + "_randBits_r"
	bitIDs = make([]string, p.l+1)
	xorBitIdentifiers := make([]string, p.l+1)
	iteration := 0
	iterationString := "iteration" + strconv.Itoa(iteration)
	for {

		for i := 0; i <= p.l; i++ {
			bitIDs[i] = identifier + "_randBits_" + iterationString + "_r" + strconv.Itoa(i)
			xorBitIdentifiers[i] = identifier + "_randBits_" + iterationString + "_xor" + strconv.Itoa(i)
			p.RandomBit(bitIDs[i])
		}
		//get bits of P
		//p.prime.Bit(i)

		//compare bits of P and r usign "bitCompare" optimized for only one secret set of bits
		//Compute sharing of XOR
		for i := 0; i <= p.l; i++ {
			ithBitOfRShare := p.getShareValue(bitIDs[i])
			ithBitOfPrime := big.NewInt(int64(p.prime.Bit(i)))
			xorProduct := big.NewInt(2)
			xorProduct.Mul(xorProduct, ithBitOfPrime)
			xorProduct.Mul(xorProduct, ithBitOfRShare)
			xorProduct.Mod(xorProduct, p.prime)
			ithBitOfXor := new(big.Int).Add(ithBitOfRShare, ithBitOfPrime)
			ithBitOfXor.Sub(ithBitOfXor, xorProduct)
			ithBitOfXor.Mod(ithBitOfXor, p.prime)
			p.shareLock.Lock()
			p.shares[xorBitIdentifiers[i]] = ithBitOfXor
			p.shareLock.Unlock()
		}

		//Find most signigicant bit
		dBitIDs := p.mostSignificant1(xorBitIdentifiers)

		comparisonBitShare := big.NewInt(0)
		for i := range dBitIDs {
			ithBitOfD := p.getShareValue(dBitIDs[i])
			ithBitOfPrime := big.NewInt(int64(p.prime.Bit(i)))
			ithBitOfPrime.Mul(ithBitOfPrime, ithBitOfD)
			comparisonBitShare.Add(comparisonBitShare, ithBitOfPrime)
		}
		comparisonBitShare.Mod(comparisonBitShare, p.prime)

		//Open comparison bit
		share := identifiedShare{
			point: bigshamir.SecretShare{
				X: p.index,
				Y: comparisonBitShare,
			},
			id: identifier + "_randBits_" + iterationString + "_comparisonBit",
		}
		for i := 1; i <= p.n; i++ {
			p.Send(share, i)
		}

		comparisonBit := p.Reconstruct(identifier + "_randBits_" + iterationString + "_comparisonBit")

		if comparisonBit.Sign() == 0 {
			iteration++
			iterationString = "iteration" + strconv.Itoa(iteration)
		} else {
			break
		}
	}

	fieldElementShare := big.NewInt(0)
	for i := range bitIDs {
		tmp := big.NewInt(2)
		tmp.Exp(tmp, big.NewInt(int64(i)), p.prime)
		fieldElementShare.Add(fieldElementShare, tmp)
	}
	fieldElementShare.Mod(fieldElementShare, p.prime)

	p.shareLock.Lock()
	p.shares[fieldElemID] = fieldElementShare
	p.shareLock.Unlock()

	return
}

//RandomBit stores a uniformly random bit as "identifier"
func (p *Player) RandomBit(identifier string) {
	iteration := 0
	var (
		aSquared *big.Int
		aShare   *big.Int
	)
	for {
		iterationIdentifier := identifier + "_iteration_" + strconv.Itoa(iteration)
		//generate random field element
		localRandomFieldElement, _ := rand.Int(rand.Reader, p.prime)
		points := p.ss.Share(localRandomFieldElement)
		for _, point := range points {
			share := localRandomFieldElementShare{
				point: point,
				id:    iterationIdentifier,
				index: p.index,
			}
			p.Send(share, point.X)
		}
		//Add at least t+1 shares to have randomness
		//in the passive corruption model we add all n shares
		//to avoid having to agree on which t+1 shares
		p.randomBitLock.RLock()
		shares := p.randFieldElemShares[iterationIdentifier]
		p.randomBitLock.RUnlock()
		for shares == nil || len(shares) < p.n {
			//todo seems like the time to use sync.Waitgroup
			time.Sleep(time.Millisecond)
			p.randomBitLock.RLock()
			shares = p.randFieldElemShares[iterationIdentifier]
			p.randomBitLock.RUnlock()
		}
		aShare = big.NewInt(0)
		for _, share := range shares {
			aShare.Add(aShare, share.point.Y)
		}
		aShare.Mod(aShare, p.prime)

		//Compute A = a^2
		aSquaredShareVal := new(big.Int).Mul(aShare, aShare)
		//suffices to multiply locally as we are immediately reconstructing
		aSquaredShareVal.Mod(aSquaredShareVal, p.prime)
		aSquaredShare := aSquaredShare{
			point: bigshamir.SecretShare{
				X: p.index,
				Y: aSquaredShareVal,
			},
			id:        iterationIdentifier,
			iteration: 0,
		}

		for i := 1; i <= p.n; i++ {
			p.Send(aSquaredShare, i)
		}

		p.randomBitLock.RLock()
		aSquaredShares := p.randomBitASquaredShares[iterationIdentifier]
		p.randomBitLock.RUnlock()
		for aSquaredShares == nil || len(aSquaredShares) < p.n {
			//todo seems like the time to use sync.Waitgroup
			time.Sleep(time.Millisecond)
			p.randomBitLock.RLock()
			aSquaredShares = p.randomBitASquaredShares[iterationIdentifier]
			p.randomBitLock.RUnlock()
		}
		//Reconstruct A
		aSquared = p.ss.Reconstruct(aSquaredShares)
		if aSquared.Sign() == 0 {
			//The random field element was zero, try again
			iteration++
			//Some cleanup
			p.randomBitLock.Lock()
			delete(p.randFieldElemShares, iterationIdentifier)
			delete(p.randomBitASquaredShares, iterationIdentifier)
			p.randomBitLock.Unlock()
		} else {
			break
		}
	}

	b := new(big.Int).ModSqrt(aSquared, p.prime)
	b.ModInverse(b, p.prime)
	cShareVal := b.Mul(b, aShare) //c = b^-1 * a
	cShareVal.Add(cShareVal, big.NewInt(1))
	twoInverse := new(big.Int).ModInverse(big.NewInt(2), p.prime)
	r := new(big.Int).Mul(twoInverse, cShareVal)
	r.Mod(r, p.prime)
	p.shareLock.Lock()
	p.shares[identifier] = r
	p.shareLock.Unlock()
}

func (p *Player) mostSignificant1(bitIds []string) (resBitIds []string) {
	fBitIds := make([]string, p.l+1)
	resBitIds = make([]string, p.l+1)
	for i := range resBitIds {
		fBitIds[i] = bitIds[i] + "_ms1_f" + strconv.Itoa(i)
		resBitIds[i] = bitIds[i] + "_ms1_d" + strconv.Itoa(i)
	}
	i := p.l
	//f_l = 1 - c_l
	p.SubFromOpen(big.NewInt(1), bitIds[i], fBitIds[i])
	//d_l = 1 - f_l
	p.SubFromOpen(big.NewInt(1), fBitIds[i], resBitIds[i])
	for i--; i >= 0; i-- {
		//f_i = f_i+1 * (1 - c_i)
		p.SubFromOpen(big.NewInt(1), bitIds[i], fBitIds[i]+"tmp")
		p.Multiply(fBitIds[i+1], fBitIds[i]+"tmp", fBitIds[i])
		//d_i = f_i+1 - f_i
		p.Sub(fBitIds[i+1], fBitIds[i], resBitIds[i])
	}
	return
}

//******************  NETWORK:  ****************

//Send any type of data to party with index receiver
func (p *Player) Send(data interface{}, receiver int) {
	if receiver == p.index {
		go p.Handle(data, p.index)
	} else {
		p.network.Send(data, receiver)
	}
}

//Handle handles data from
func (p *Player) Handle(data interface{}, sender int) {
	switch t := data.(type) {
	case identifiedShare:
		if t.point.X == p.index {
			//We have received aShare regular share
			p.shareLock.Lock()
			p.shares[t.id] = t.point.Y
			p.shareLock.Unlock()
		} else {
			//We have received another party's share
			p.shareLock.Lock()
			p.reconstructionShares[t.id] = append(p.reconstructionShares[t.id], t.point)
			//todo adding own index?
			p.shareLock.Unlock()
		}
	case multiplicationShare:
		p.shareLock.Lock()
		p.multShares[t.id] = append(p.multShares[t.id], t)
		p.shareLock.Unlock()
	case localRandomFieldElementShare:
		p.randomBitLock.Lock()
		p.randFieldElemShares[t.id] = append(p.randFieldElemShares[t.id], t)
		p.randomBitLock.Unlock()
	case aSquaredShare:
		p.randomBitLock.Lock()
		p.randomBitASquaredShares[t.id] = append(p.randomBitASquaredShares[t.id], t.point)
		p.randomBitLock.Unlock()
	}
}

//Index of player
func (p *Player) Index() int {
	return p.index
}

//RegisterNetwork ...
func (p *Player) RegisterNetwork(network network.Network) {
	p.network = network
}

//********** INTERPRETER **************
type instruction = []string

//Run executes the computations specified by instructions
func (p *Player) Run() map[string]*big.Int {
	output := make(map[string]*big.Int)
	//["ADD", "X", "Y", "Z"]
	//p.Add("X", "Y", "Z")
	//["MUL", "X", "Y", "Z"]
	//p.Multiply("X", "Y", "Z")

	//["INPUT", "2", "ID123"]
	//if p.index == strconv("2") {p.Share(readinput("ID123"), "ID123")}

	//["LT", "X", "Y", "Z"]

	for _, insn := range p.instructions {
		if len(insn) == 0 {
			continue
		}
		switch insn[0] {
		case "INPUT": //["INPUT", index of party, id]
			index, err := strconv.Atoi(insn[1])
			if err != nil {
				fmt.Println()
			}
			if err != nil || index != p.index {
				continue
			}
			value := p.readInput(insn[2])
			p.Share(value, insn[2])
		case "ADD": //["ADD", "ID", "ID", "ID"]
			p.Add(insn[1], insn[2], insn[3])
		case "MULTIPLY": //["MULTIPLY", "ID", "ID", "ID"]
			p.Multiply(insn[1], insn[2], insn[3])
		case "OPEN": //["OPEN", ID]
			p.Open(insn[1])
		case "OUTPUT": //["OUTPUT", ID]
			output[insn[1]] = p.Reconstruct(insn[1])
		case "SCALE": //["SCALE", "NUM", "ID", "ID"]
			scalar, isNumber := new(big.Int).SetString(insn[1], 10) //todo NaN?
			if !isNumber {
				continue
			}
			p.Scale(scalar, insn[2], insn[3])
		case "ADD_CONSTANT": //["ADD_CONSTANT", "NUM", "ID", "ID"]
			constant, isNumber := new(big.Int).SetString(insn[1], 10) //todo NaN?
			if !isNumber {
				continue
			}
			p.AddOpen(constant, insn[2], insn[3])
		case "RANDOM_BIT":
			p.RandomBit(insn[1])
		}
	}

	return output
}

func (p *Player) readInput(identifier string) *big.Int {
	value, exist := p.inputValues[identifier] //todo concurrency?
	if !exist {
		fmt.Println("Party", p.index, "has no input value named", identifier)
		for id, val := range p.inputValues {
			fmt.Println(id, ":", val)
		}
	}
	return value
}

func (p *Player) setInput(inputValues map[string]*big.Int) {
	p.inputValues = inputValues
}

func (p *Player) scanInput(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	p.inputValues = make(map[string]*big.Int)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, "=")
		if len(tokens) != 2 {
			continue
		}
		identifier := strings.TrimSpace(tokens[0])

		//read input as base 10 int:
		value, ok := new(big.Int).SetString(strings.TrimSpace(tokens[1]), 10)
		if !ok {
			fmt.Println("could not parse value of", identifier, ":", tokens[1])
			continue
		}
		p.inputValues[identifier] = value
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func (p *Player) scanInstructions(path string) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		tokens := strings.Split(line, " ")
		if len(tokens) == 0 {
			continue
		}
		p.instructions = append(p.instructions, tokens)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
