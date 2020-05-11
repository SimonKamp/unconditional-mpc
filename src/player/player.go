package player

import (
	"fmt"
	"math/big"
	"../network"
	"../bigshamir"
	"sync"
	"time"
)

//Player runs the protocol
type Player struct {
	prime *big.Int
	threshold int
	n int
	index int
	ss *bigshamir.SecretSharingScheme
	network network.Network
	shareLock sync.RWMutex
	shares map[string]*big.Int
	multiplicationShares map[string][]multiplicationShare
	reconstructionShares map[string][]bigshamir.Point
}

type identifiedShare struct {
	point bigshamir.Point
	identifier string
}

type multiplicationShare struct {
	point bigshamir.Point
	identifier string
	index int
}

//NewPlayer ...
func NewPlayer(prime int64, threshold, n, index int) *Player {
	p := new(Player)
	p.prime = big.NewInt(prime)
	p.threshold = threshold
	p.n = n
	p.index = index
	p.ss = new(bigshamir.SecretSharingScheme)
	*p.ss = bigshamir.NewSS(prime, threshold, n)
	p.shares = make(map[string]*big.Int)
	p.multiplicationShares = make(map[string][]multiplicationShare)
	p.reconstructionShares = make(map[string][]bigshamir.Point)
	return p
}


//Share ...
func (p *Player)Share(x *big.Int, identifier string) {
	points := p.ss.Share(x)
	for _, point := range points {
		p.Send(identifiedShare{point : point, identifier : identifier }, point.X)
	}
}

//Open ... 
func (p *Player)Open(identifier string) {
	yValue := p.getShareValue(identifier)
	share := identifiedShare{
		point: bigshamir.Point{
			X: p.index,
			Y: yValue,
		},
		identifier : identifier,
	}
	for i := 1; i <= p.n; i++{
		p.Send(share, i)
	}
}

//Reconstruct ...
func (p *Player)Reconstruct(identifier string) *big.Int {
	p.shareLock.RLock()
	points := p.reconstructionShares[identifier]
	p.shareLock.RUnlock()
	for len(points) <= p.threshold {
		time.Sleep(time.Millisecond)
		p.shareLock.RLock()
		points = p.reconstructionShares[identifier]
		p.shareLock.RUnlock()
	}
	return p.ss.Reconstruct(points)
}

func (p *Player)getShareValue(identifier string) *big.Int {
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
func (p *Player)Add(aIdentifier, bIdentifier, cIdentifier string) {
	sum := new(big.Int)
	sum.Add(p.getShareValue(aIdentifier), p.getShareValue(bIdentifier))
	sum.Mod(sum, p.prime)
	p.shareLock.Lock()
	p.shares[cIdentifier] = sum
	p.shareLock.Unlock()
}

//Multiply ...
func (p *Player)Multiply(aIdentifier, bIdentifier, cIdentifier string) {
	localProduct := new(big.Int)
	localProduct.Mul(p.getShareValue(aIdentifier), p.getShareValue(bIdentifier))
	for _, share := range p.ss.Share(localProduct){
		ms := multiplicationShare{
			point: share,
			identifier: cIdentifier,
			index: p.index,
		}
		go p.Send(ms, share.X)
	}

	p.shareLock.RLock()
	multiplicationShares := p.multiplicationShares[cIdentifier]
	p.shareLock.RUnlock()
	for len(multiplicationShares) < p.threshold * 2 + 1 {
		time.Sleep(time.Millisecond)
		p.shareLock.RLock()
		multiplicationShares = p.multiplicationShares[cIdentifier]
		p.shareLock.RUnlock()
	}

	xs := make([]int, len(multiplicationShares))
	for i := range multiplicationShares {
		xs[i] = multiplicationShares[i].index
	}

	r := bigshamir.ReconstructionVector(p.prime, xs...)
	sum := big.NewInt(0)
	for _, share := range(multiplicationShares) {
		ri := r[share.index]

		product := new(big.Int).Mul(ri, share.point.Y)
		sum.Add(sum, product)
	}
	sum.Mod(sum, p.prime)

	p.shareLock.Lock()
	p.shares[cIdentifier] = sum
	p.shareLock.Unlock()
}

//******************  NETWORK:  ****************


//Send ...
func (p *Player)Send(data interface{}, receiver int) {
	p.network.Send(data, receiver)
}

//Handle handles data from
func (p *Player)Handle(data interface{}, sender int) {
	switch t :=data.(type) {
	case bigshamir.Point:
		fmt.Println(t, "is point")
	case identifiedShare:
		if t.point.X == p.index {
			//We have received a regular share
			p.shareLock.Lock()
			p.shares[t.identifier] = t.point.Y
			p.shareLock.Unlock()
		} else {
			//We have received another party's share
			p.shareLock.Lock()
			p.reconstructionShares[t.identifier] = append(p.reconstructionShares[t.identifier], t.point)
			//todo adding own index?
			p.shareLock.Unlock()
		}
	case multiplicationShare:
		p.shareLock.Lock()
		p.multiplicationShares[t.identifier] =
			append(p.multiplicationShares[t.identifier], t)
		p.shareLock.Unlock()
	}
}

//Index of player
func (p *Player)Index() int {
	return p.index
}

//RegisterNetwork ... 
func (p *Player)RegisterNetwork(network network.Network) {
	p.network = network
}