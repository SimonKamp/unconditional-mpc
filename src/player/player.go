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
func (p *Player) Compare(aIdentifier, bIdentifier, cIdentifier string) {
	//compute sharings of bits of aShare and b:
	// aBits := p.bits(aIdentifier)
	// bBits := p.bits(bIdentifier)
	// cBits := make([]bool, len(aBits))
	// for i := range aBits {
	// 	cBits[i] = aBits[i] != bBits[i] //todo aShare+b-2ab
	// }
	// dBits := p.mostSignificant1(cBits)
	// eBits := make([]bool, len(aBits))
	// for i := range aBits {
	// 	eBits[i] = aBits[i] && dBits[i] //todo e = ab
	// }
	// c := false
	// for i := range aBits {
	// 	c = c != eBits[i] //todo c = sum i=0..l: ei
	// }
}

func (p *Player) bitCompare(aIdentifier, bIdentifier, cIdentifier string) {

}

func (p *Player) bitAdd(bits []string) (resBitIds []string) { return }
func (p *Player) bitSub(bits []string) (resBitIds []string) { return }

func (p *Player) bits(identifier string) []bool {

	return []bool{}
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
			go p.RandomBit(bitIDs[i])
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
			fmt.Println("random bits were < P", comparisonBit)
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
