package balancer

import (
	"github.com/lafikl/consistent"
)

func init() {
	factories[ConsistentHashBalancer] = NewConsistent
}

type Consistent struct {
	BaseBalancer
	ch *consistent.Consistent
}

func NewConsistent(hosts []string) Balancer {
	c := &Consistent{
		BaseBalancer: BaseBalancer{},
		ch:           consistent.New(),
	}
	for _, h := range hosts {
		c.ch.Add(h)
	}
	return c
}

func (c *Consistent) Add(host string) {
	c.ch.Add(host)
}

func (c *Consistent) Remove(host string) {
	c.ch.Remove(host)
}

func (c *Consistent) Balance(key string) (string, error) {
	if len(c.ch.Hosts()) == 0 {
		return "", NoHostError
	}
	return c.ch.Get(key)
}
