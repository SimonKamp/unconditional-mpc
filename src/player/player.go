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
	prime     *big.Int
	threshold int
	n         int
	index     int
	l         int

	bitLength    int
	primeSharing []string

	ss           *bigshamir.SecretSharingScheme
	network      network.Network
	inputValues  map[string]*big.Int
	instructions []instruction

	//Concurrently accessed:
	//Regular shares
	shareLock             sync.RWMutex
	idVals                map[string]*big.Int //todo consider using sync.Map
	idValBlockingChannels map[string][]chan *big.Int
	secrets               map[string]bool //subset of values

	reconstructionShareLock             sync.RWMutex
	reconstructionShares                map[string]map[int]*big.Int
	reconstructionShareBlockingChannels map[string][]chan map[int]*big.Int

	//Recombination shares for multiplication
	multShareLock sync.RWMutex
	multShares    map[string][]multiplicationShare

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
	reconstructionShare struct {
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
	p.idVals = make(map[string]*big.Int)
	p.idValBlockingChannels = make(map[string][]chan *big.Int)
	p.secrets = make(map[string]bool)
	p.reconstructionShares = make(map[string]map[int]*big.Int)
	p.reconstructionShareBlockingChannels = make(map[string][]chan map[int]*big.Int)
	p.multShares = make(map[string][]multiplicationShare)
	p.randFieldElemShares = make(map[string][]localRandomFieldElementShare)
	p.randomBitASquaredShares = make(map[string][]bigshamir.SecretShare)
	p.inputValues = make(map[string]*big.Int)

	p.bitLength = p.prime.BitLen() + 1
	p.primeSharing = make([]string, p.bitLength)
	for i := range p.primeSharing {
		id := "_primeSharing_" + strconv.Itoa(i)
		p.primeSharing[i] = id
		p.setShareValue(id, big.NewInt(int64(p.prime.Bit(i))), false)
	}

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
	yValue, _ := p.getShareValue(identifier)
	secretShare := bigshamir.SecretShare{
		X: p.index,
		Y: yValue,
	}
	share := identifiedShare{
		point: secretShare,
		id:    identifier,
	}
	// p.reconstructionShareLock.Lock()
	// p.reconstructionShares[identifier] = append(p.reconstructionShares[identifier], secretShare)
	// p.reconstructionShareLock.Unlock()
	for i := 1; i <= p.n; i++ {
		if i == p.index {
			continue
		}
		go p.Send(share, i)
	}
}

func (p *Player) mapToPoints(m map[int]*big.Int) (points []bigshamir.SecretShare) {
	p.reconstructionShareLock.RLock()
	for x, y := range m {
		share := bigshamir.SecretShare{
			X: x,
			Y: y,
		}
		points = append(points, share)
	}
	p.reconstructionShareLock.RUnlock()
	return
}

//Reconstruct ...
func (p *Player) Reconstruct(identifier string) *big.Int {
	p.reconstructionShareLock.RLock()
	points := p.reconstructionShares[identifier]
	p.reconstructionShareLock.RUnlock()

	if len(points) > p.threshold {
		return p.ss.Reconstruct(p.mapToPoints(points))
	}

	p.reconstructionShareLock.Lock()
	points = p.reconstructionShares[identifier]
	if len(points) > p.threshold {
		p.reconstructionShareLock.Unlock()
		return p.ss.Reconstruct(p.mapToPoints(points))
	}

	channel := make(chan map[int]*big.Int)
	channels := p.reconstructionShareBlockingChannels[identifier]
	p.reconstructionShareBlockingChannels[identifier] = append(channels, channel)
	p.reconstructionShareLock.Unlock()
	points = <-channel

	return p.ss.Reconstruct(p.mapToPoints(points))
}

func (p *Player) getShareValue(identifier string) (val *big.Int, isSecret bool) {
	p.shareLock.RLock()
	val, exists := p.idVals[identifier]
	isSecret = p.secrets[identifier]
	p.shareLock.RUnlock()

	if exists {
		return
	}

	p.shareLock.Lock()
	val, exists = p.idVals[identifier]
	if exists {
		isSecret = p.secrets[identifier]
		p.shareLock.Unlock()
		return
	}
	resultChannel := make(chan *big.Int)
	channels := p.idValBlockingChannels[identifier]
	p.idValBlockingChannels[identifier] = append(channels, resultChannel)
	p.shareLock.Unlock()
	val = <-resultChannel

	//todo send both over channel
	p.shareLock.RLock()
	isSecret = p.secrets[identifier]
	p.shareLock.RUnlock()
	return

}

func (p *Player) setShareValue(id string, val *big.Int, isSecret bool) {
	p.shareLock.Lock()
	p.idVals[id] = val
	if isSecret {
		p.secrets[id] = true
	}
	for _, channel := range p.idValBlockingChannels[id] {
		channel <- val
	}
	delete(p.idValBlockingChannels, id)
	p.shareLock.Unlock()
}

//Add ...
func (p *Player) Add(aID, bID, cID string) {
	sum := new(big.Int)
	a, aIsSecret := p.getShareValue(aID)
	b, bIsSecret := p.getShareValue(bID)
	sum.Add(a, b)
	sum.Mod(sum, p.prime)

	secret := aIsSecret || bIsSecret
	p.setShareValue(cID, sum, secret)
}

//AddConstant ...
func (p *Player) AddConstant(aShare *big.Int, bID, cID string) {
	sum := new(big.Int)
	b, bIsSecret := p.getShareValue(bID)
	sum.Add(aShare, b)
	sum.Mod(sum, p.prime)

	p.setShareValue(cID, sum, bIsSecret)
}

//Sub ...
func (p *Player) Sub(aID, bID, cID string) {
	a, aIsSecret := p.getShareValue(aID)
	b, bIsSecret := p.getShareValue(bID)

	res := new(big.Int)
	res.Sub(a, b)
	res.Mod(res, p.prime)

	secret := aIsSecret || bIsSecret
	p.setShareValue(cID, res, secret)
}

//SubFromConstant ...
func (p *Player) SubFromConstant(aShare *big.Int, bID, cID string) {
	b, bIsSecret := p.getShareValue(bID)

	res := new(big.Int)
	res.Sub(aShare, b)
	res.Mod(res, p.prime)

	p.setShareValue(cID, res, bIsSecret)
}

//SubConstant ...
func (p *Player) SubConstant(aID string, bID *big.Int, cID string) {
	a, aIsSecret := p.getShareValue(aID)

	res := new(big.Int)
	res.Add(a, bID)
	res.Mod(res, p.prime)

	p.setShareValue(cID, res, aIsSecret)
}

//Scale ...
func (p *Player) Scale(scalar *big.Int, aID, bID string) {
	a, aIsSecret := p.getShareValue(aID)

	res := new(big.Int)
	res.Mul(a, scalar)
	res.Mod(res, p.prime)

	p.setShareValue(bID, res, aIsSecret)
}

//Multiply ...
func (p *Player) Multiply(aID, bID, cID string) {
	a, aIsSecret := p.getShareValue(aID)
	b, bIsSecret := p.getShareValue(bID)

	//Compute and secret share product of local shares
	localProduct := new(big.Int).Mul(a, b)

	cIsSecret := aIsSecret || bIsSecret
	shouldReconconstruct := aIsSecret && bIsSecret
	if !shouldReconconstruct {
		//Multiplication does not need communication
		p.setShareValue(cID, localProduct, cIsSecret)
		return
	}

	//Secret multiplication
	for _, share := range p.ss.Share(localProduct) {
		ms := multiplicationShare{
			recombinationShare: bigshamir.RecombinationShare{
				SecretShare: share,
				Index:       p.index,
			},
			id: cID,
		}
		go p.Send(ms, share.X)
	}

	//Wait for 2t+1 sharings of local products
	p.multShareLock.RLock()
	shares := p.multShares[cID]
	p.multShareLock.RUnlock()
	if len(shares) >= p.threshold*2+1 {
		p.recombineMultiplicationShares(cID, shares)
	}
}

func (p *Player) recombineMultiplicationShares(cID string, shares []multiplicationShare) {
	multShares := make([]bigshamir.RecombinationShare, len(shares))
	for i := range shares {
		multShares[i] = shares[i].recombinationShare
	}
	cShareValue := p.ss.RecombineMultiplicationShares(multShares)

	p.setShareValue(cID, cShareValue, true)
}

//GreaterThan takes shares of aShare and b as input and outputs 1 iff a > b, and 0 otherwise
func (p *Player) GreaterThan(aID, bID, cID string) {

	//compute sharings of bits of aShare and b:
	aBitIDs := make([]string, p.l+1)
	bBitIDs := make([]string, p.l+1)
	for i := range aBitIDs {
		aBitIDs[i] = cID + "=" + aID + ">" + bID +
			"_:_bits_of_a_" + aID + "_index_" + strconv.Itoa(i)
		bBitIDs[i] = cID + "=" + aID + ">" + bID +
			"_:_bits_of_b_" + bID + "_index_" + strconv.Itoa(i)
	}
	go p.bits(aID, aBitIDs) //compute bits in parallel
	p.bits(bID, bBitIDs)

	p.bitCompare(aBitIDs, bBitIDs, cID)
}

//GreaterThanOrEqual takes shares of aShare and b as input and outputs 1 iff a >= b, and 0 otherwise
func (p *Player) GreaterThanOrEqual(aID, bID, cID string) {
	bGreaterThanAID := cID + "_GreaterThanOrEqual_b>a"
	p.GreaterThan(bID, aID, bGreaterThanAID)
	p.SubFromConstant(big.NewInt(1), bGreaterThanAID, cID)
}

//LessThan takes shares of aShare and b as input and outputs 1 iff a < b, and 0 otherwise
func (p *Player) LessThan(aID, bID, cID string) {
	p.GreaterThan(bID, aID, cID)
}

//LessThanOrEqual takes shares of aShare and b as input and outputs 1 iff a <= b, and 0 otherwise
func (p *Player) LessThanOrEqual(aID, bID, cID string) {
	aGreaterThanBID := cID + "_LessThanOrEqual_a>b"
	p.GreaterThan(aID, bID, aGreaterThanBID)
	p.SubFromConstant(big.NewInt(1), aGreaterThanBID, cID)
}

//NotEqual takes shares of aShare and b as input and outputs 1 iff a != b, and 0 otherwise
func (p *Player) NotEqual(aID, bID, cID string) {
	//todo temporary solution, should use Fermats little thm. and repeated squaring
	aGreaterThanBID := cID + "_NotEqual_a>b"
	p.GreaterThan(aID, bID, aGreaterThanBID)
	bGreaterThanAID := cID + "_NotEqual_b>a"
	p.GreaterThan(bID, aID, bGreaterThanAID)
	//sum is either
	//1 if one is greater than the other
	//0 if a == b
	p.Add(aGreaterThanBID, bGreaterThanAID, cID)
}

//Equal takes shares of aShare and b as input and outputs 1 iff a == b, and 0 otherwise
func (p *Player) Equal(aID, bID, cID string) {
	notEqualID := cID + "_Equal_tmp"
	p.NotEqual(aID, bID, notEqualID)
	p.SubFromConstant(big.NewInt(1), notEqualID, cID)
}

//For debugging
func (p *Player) openBits(bitIDs []string) []*big.Int {
	bits := make([]*big.Int, p.bitLength)
	for i, bit := range bitIDs {
		p.Open(bit)
		bits[p.bitLength-1-i] = p.Reconstruct(bit)
	}
	return bits
}

func (p *Player) bits(ID string, resultBitIDs []string) {
	rID, rBitIDs := p.randomSolvedBits(resultBitIDs[0])
	p.Sub(ID, rID, resultBitIDs[0]+"_bits_c") //c = a - r
	p.Open(resultBitIDs[0] + "_bits_c")
	c := p.Reconstruct(resultBitIDs[0] + "_bits_c")

	//Compute bit sharing of sum of r and c, i.e. bit sharing of ID
	cBitIDs := make([]string, p.l+1)
	dBitIDs := make([]string, p.l+1)
	epBitIDs := make([]string, p.l+1)

	p.shareLock.RLock()
	bitsAreSecret := p.secrets[ID]
	p.shareLock.RUnlock()

	for i := 0; i <= p.l; i++ {
		cBitIDs[i] = resultBitIDs[i] + "_bits_cBits"
		dBitIDs[i] = resultBitIDs[i] + "_bits_dBits"
		epBitIDs[i] = resultBitIDs[i] + "_bits_cepBits"
		go p.setShareValue(cBitIDs[i], big.NewInt(int64(c.Bit(i))), bitsAreSecret)
	}

	p.bitAdd(rBitIDs, cBitIDs, dBitIDs)
	notE := resultBitIDs[0] + "_bits_compareBits_not_e"
	e := resultBitIDs[0] + "_bits_compareBits_e"
	p.bitCompare(p.primeSharing, dBitIDs, notE)
	notEVal, notESecret := p.getShareValue(notE)
	p.setShareValue(e, bitNot(notEVal), notESecret)

	for i := range epBitIDs {
		p.Multiply(e, p.primeSharing[i], epBitIDs[i])
	}

	p.bitSub(dBitIDs, epBitIDs, resultBitIDs)
}

func (p *Player) bitCompare(aBitIDs, bBitIDs []string, cBitID string) {
	//Compute sharing of XOR
	xorShareIDs := make([]string, p.l+1)
	for i := 0; i <= p.l; i++ {
		xorShareIDs[i] = cBitID + "_bitCompare_xor" + strconv.Itoa(i)
		go p.bitXor(aBitIDs[i], bBitIDs[i], xorShareIDs[i])
	}

	//Compute MS1(XOR), i.e. most significant 1 in max(a,b)
	dBitIDs := p.mostSignificant1(xorShareIDs)

	eBitIDs := make([]string, p.l+1)
	for i := range eBitIDs {
		eBitIDs[i] = cBitID + "_bitCompare_e" + strconv.Itoa(i)
		go p.Multiply(aBitIDs[i], dBitIDs[i], eBitIDs[i])
	}
	cShare := big.NewInt(0)
	for i := range eBitIDs {
		eI, _ := p.getShareValue(eBitIDs[i])
		cShare.Add(cShare, eI)
	}
	cShare.Mod(cShare, p.prime)

	p.shareLock.RLock()
	resultIsSecret := p.secrets[aBitIDs[0]] || p.secrets[bBitIDs[0]]
	p.shareLock.RUnlock()

	p.setShareValue(cBitID, cShare, resultIsSecret)
}

func (p *Player) bitXor(aBitID, bBitID, cBitID string) *big.Int {
	p.Multiply(aBitID, bBitID, cBitID+"_xor_tmp")
	aShare, aIsSecret := p.getShareValue(aBitID)
	bShare, bIsSecret := p.getShareValue(bBitID)
	xorShare, _ := p.getShareValue(cBitID + "_xor_tmp") //ab
	xorShare.Mul(big.NewInt(2), xorShare)               //2ab
	xorShare.Neg(xorShare)                              //-2ab
	xorShare.Add(xorShare, aShare)                      //a - 2ab
	xorShare.Add(xorShare, bShare)                      //a+b-2ab

	p.setShareValue(cBitID, xorShare, aIsSecret || bIsSecret)

	return xorShare
}

func (p *Player) fullAdder(aBitID, bBitID, carryInBitID, carryOutBitID, cBitID string) {
	//carry_out = (a & b) | (a & carry_in) | (b & carry_in)
	//			= ! (!(a & b) & !(a & carry_in) & !(b & carry_in))
	//			= 1 - ((1 - a * b) * (1 - a * carry_in) * (1 - b * carry_in))
	p.Multiply(aBitID, bBitID, cBitID+"_a&b") //todo in parallel
	p.Multiply(aBitID, carryInBitID, cBitID+"_a&carryIn")
	p.Multiply(bBitID, carryInBitID, cBitID+"_b&carryIn")

	ab, resultIsSecret := p.getShareValue(cBitID + "_a&b")
	aCarryIn, _ := p.getShareValue(cBitID + "_a&carryIn")

	p.setShareValue(cBitID+"_!a&b", bitNot(ab), resultIsSecret)
	p.setShareValue(cBitID+"_!a&carryIn", bitNot(aCarryIn), resultIsSecret)

	p.Multiply(cBitID+"_!a&b", cBitID+"_!a&carryIn", cBitID+"_!a&b_&_!a&carryIn")

	bCarryIn, _ := p.getShareValue(cBitID + "_b&carryIn")
	p.setShareValue(cBitID+"_!b&carryIn", bitNot(bCarryIn), resultIsSecret)

	p.Multiply(cBitID+"_!a&b_&_!a&carryIn", cBitID+"_!b&carryIn", cBitID+"_disjunct")

	disjunctShare, _ := p.getShareValue(cBitID + "_disjunct")
	carryOutShare := bitNot(disjunctShare)
	carryOutShare.Mod(carryOutShare, p.prime)

	p.setShareValue(carryOutBitID, carryOutShare, resultIsSecret)

	//c = a + b + c - 2 * carry_out
	resBitShare := new(big.Int).Mul(big.NewInt(2), carryOutShare)
	resBitShare.Neg(resBitShare)
	a, _ := p.getShareValue(aBitID)
	b, _ := p.getShareValue(bBitID)
	carryIn, _ := p.getShareValue(carryInBitID)
	resBitShare.Add(resBitShare, a)
	resBitShare.Add(resBitShare, b)
	resBitShare.Add(resBitShare, carryIn)
	resBitShare.Mod(resBitShare, p.prime)

	p.setShareValue(cBitID, resBitShare, resultIsSecret)
}

func bitNot(aBitShare *big.Int) *big.Int {
	return new(big.Int).Sub(big.NewInt(1), aBitShare)
}

func (p *Player) bitAdd(aBitIDs, bBitIDs, resBitIDs []string) {
	if len(aBitIDs) != p.bitLength ||
		len(aBitIDs) != p.bitLength ||
		len(resBitIDs) != p.bitLength { //todo + 1
		panic("bit add different lengths")
		return
	}

	for i := range resBitIDs {
		if i == 0 {
			//Carry in = 0
			p.setShareValue(resBitIDs[i]+"_carry_in_0", big.NewInt(0), false)
			//initial carry is not secret
			carryInID := resBitIDs[i] + "_carry_in_0"
			carryOutID := resBitIDs[i] + "_carry_out_0"
			p.fullAdder(aBitIDs[i], bBitIDs[i], carryInID, carryOutID, resBitIDs[i])
		} else {
			carryInID := resBitIDs[i-1] + "_carry_out_" + strconv.Itoa(i-1)
			carryOutID := resBitIDs[i] + "_carry_out_" + strconv.Itoa(i)
			p.fullAdder(aBitIDs[i], bBitIDs[i], carryInID, carryOutID, resBitIDs[i])
		}
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
		bit, isSecret := p.getShareValue(bBitIDs[i])
		flippedBit := bitNot(bit)
		p.setShareValue(flippedBBitIDs[i], flippedBit, isSecret)
	}

	for i := range resBitIDs {
		if i == 0 {
			//Carry in = 1
			p.setShareValue(resBitIDs[i]+"_carry_in_0", big.NewInt(1), false)
			//initial carry is not secret
			carryInID := resBitIDs[i] + "_carry_in_0"
			carryOutID := resBitIDs[i] + "_carry_out_0"
			p.fullAdder(aBitIDs[i], flippedBBitIDs[i], carryInID, carryOutID, resBitIDs[i])
		} else {
			carryInID := resBitIDs[i-1] + "_carry_out_" + strconv.Itoa(i-1)
			carryOutID := resBitIDs[i] + "_carry_out_" + strconv.Itoa(i)
			p.fullAdder(aBitIDs[i], flippedBBitIDs[i], carryInID, carryOutID, resBitIDs[i])
		}
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
		//Draw random bits
		for i := 0; i <= p.l; i++ {
			bitIDs[i] = identifier + "_randBits_" + iterationString + "_r" + strconv.Itoa(i)
			xorBitIdentifiers[i] = identifier + "_randBits_" + iterationString + "_xor" + strconv.Itoa(i)
			p.RandomBit(bitIDs[i])
		}
		//Check if random bits represent a field element
		comparisonID := identifier + "_randBits_" + iterationString + "_comparisonBit"
		p.bitCompare(p.primeSharing, bitIDs, comparisonID)
		p.Open(comparisonID)

		comparisonBit := p.Reconstruct(comparisonID)

		if comparisonBit.Sign() == 0 {
			iteration++
			iterationString = "iteration" + strconv.Itoa(iteration)
		} else {
			break
		}
	}

	fieldElementShare := big.NewInt(0)
	for i := range bitIDs {
		bit, _ := p.getShareValue(bitIDs[i])
		term := new(big.Int).Set(bit)
		term.Lsh(term, uint(i))
		fieldElementShare.Add(fieldElementShare, term)
	}

	// ithPowerOfTwo := big.NewInt(1)
	// two := big.NewInt(2)
	// for i := range bitIDs {
	// 	bit, _ := p.getShareValue(bitIDs[i])
	// 	term := new(big.Int).Set(bit)
	// 	term.Mul(term, ithPowerOfTwo)
	// 	ithPowerOfTwo.Mul(ithPowerOfTwo, two)
	// 	fieldElementShare.Add(fieldElementShare, term)
	// }

	fieldElementShare.Mod(fieldElementShare, p.prime)

	p.setShareValue(fieldElemID, fieldElementShare, true)

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
		time.Sleep(time.Millisecond)
		p.randomBitLock.RLock()
		shares := p.randFieldElemShares[iterationIdentifier]
		p.randomBitLock.RUnlock()
		for len(shares) < p.n {
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

		time.Sleep(time.Millisecond)
		p.randomBitLock.RLock()
		aSquaredShares := p.randomBitASquaredShares[iterationIdentifier]
		p.randomBitLock.RUnlock()
		for len(aSquaredShares) < p.n {
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

	p.setShareValue(identifier, r, true)
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
	p.SubFromConstant(big.NewInt(1), bitIds[i], fBitIds[i])
	//d_l = 1 - f_l
	p.SubFromConstant(big.NewInt(1), fBitIds[i], resBitIds[i])
	i--
	for i >= 0 {
		//f_i = f_i+1 * (1 - c_i)
		p.SubFromConstant(big.NewInt(1), bitIds[i], fBitIds[i]+"tmp")

		p.Multiply(fBitIds[i+1], fBitIds[i]+"tmp", fBitIds[i])
		//d_i = f_i+1 - f_i
		p.Sub(fBitIds[i+1], fBitIds[i], resBitIds[i])
		i--
	}
	return
}

//******************  NETWORK:  ****************

//Send any type of data to party with index receiver
func (p *Player) Send(data interface{}, receiver int) {
	if receiver == p.index {
		go p.Handle(data, p.index)
	} else {
		go p.network.Send(data, receiver)
	}
}

//Handle handles data from
func (p *Player) Handle(data interface{}, sender int) {
	switch t := data.(type) {
	case identifiedShare:
		if t.point.X == p.index {
			//We have received aShare regular share
			p.setShareValue(t.id, t.point.Y, true)
		} else {
			//We have received another party's share
			p.reconstructionShareLock.Lock()
			if p.reconstructionShares[t.id] == nil {
				p.reconstructionShares[t.id] = make(map[int]*big.Int)
			}
			p.reconstructionShares[t.id][t.point.X] = t.point.Y
			if len(p.reconstructionShares[t.id]) == p.threshold+1 {
				//Notify waiting routines
				for _, channel := range p.reconstructionShareBlockingChannels[t.id] {
					channel <- p.reconstructionShares[t.id]
				}
				delete(p.reconstructionShareBlockingChannels, t.id)
			}

			p.reconstructionShareLock.Unlock()
		}
	case multiplicationShare:
		p.multShareLock.Lock()
		shares := append(p.multShares[t.id], t)
		if len(shares) < p.threshold*2+1 {
			p.multShares[t.id] = shares
			p.multShareLock.Unlock()
		} else {
			p.multShareLock.Unlock()
			p.recombineMultiplicationShares(t.id, shares)
		}
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
			p.AddConstant(constant, insn[2], insn[3])
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
