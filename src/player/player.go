package player

import (
	"fmt"
	"math/big"
	"../network"
)

//Player runs the protocol
type Player struct {
	prime *big.Int
	t int
	n int
	index int
	network network.Network
}

func NewPlayer(prime *big.Int, t, n, index int) *Player {
	p := new(Player)
	p.prime = prime
	p.t = t
	p.n = n
	p.index = index
	return p
}

func (p *Player)Send(data interface{}, party int) {
	p.network.Send(data, party)
}

func (p *Player)share(x *big.Int, identifier string) {

}

//Handle handles data from
func (p *Player)Handle(data interface{}, sender int) {
	fmt.Println(p.index, "received", data, "from", sender)
}

//Index of player
func (p *Player)Index() int {
	return p.index
}

func (p *Player)RegisterNetwork(network network.Network) {
	p.network = network
}