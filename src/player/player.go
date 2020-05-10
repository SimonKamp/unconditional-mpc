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
	shareLock sync.Mutex
	shares map[string]*big.Int
	reconstructionShares map[string][]bigshamir.Point
}

type identifiedShare struct {
	point bigshamir.Point
	identifier string
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
	p.reconstructionShares = make(map[string][]bigshamir.Point)
	return p
}


//Share ...
func (p *Player)Share(x *big.Int, identifier string) {
	points := p.ss.Share(x)
	for _, point := range points {
		fmt.Println(point)
		p.Send(identifiedShare{point : point, identifier : identifier }, point.X)
	}
}

//Add ...
func (p *Player)Add(aIdentifier, bIdentifier, cIdentifier string) {
	sum := new(big.Int)
	sum.Add(p.getShareValue(aIdentifier), p.getShareValue(bIdentifier))
	p.shares[cIdentifier] = sum
}

//Open ... 
func (p *Player)Open(identifier string) {
	yValue, exists := p.shares[identifier]
	for !exists { 
		time.Sleep(time.Millisecond)
		yValue, exists = p.shares[identifier]
	}
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
func (p *Player)Reconstruct(identifier string) {
	points := p.reconstructionShares[identifier]
	for len(points) <= p.threshold {
		fmt.Println("wait for more shares to reconstruct")
		time.Sleep(time.Millisecond)
		points = p.reconstructionShares[identifier]
	}
	fmt.Println(p.ss.Reconstruct(points))

}

func (p *Player)getShareValue(identifier string) *big.Int {
	value, exists := p.shares[identifier]
	for !exists { 
		time.Sleep(time.Millisecond)
		value, exists = p.shares[identifier]
	}
	return value
}

//******************  NETWORK:  ****************


//Send ...
func (p *Player)Send(data interface{}, receiver int) {
	p.network.Send(data, receiver)
}

//Handle handles data from
func (p *Player)Handle(data interface{}, sender int) {
	//fmt.Println(p.index, "received", data, "from", sender)
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
			//todo locks
			p.shareLock.Lock()
			p.reconstructionShares[t.identifier] = append(p.reconstructionShares[t.identifier], t.point)
			if len(p.reconstructionShares[t.identifier]) == p.threshold {
				//We have enough identifiedShare if we add our own
				//todo delete?
				p.reconstructionShares[t.identifier] = 
				append(p.reconstructionShares[t.identifier], 
					bigshamir.Point{X: p.index, Y: p.shares[t.identifier]})
			}
			p.shareLock.Unlock()
		}
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