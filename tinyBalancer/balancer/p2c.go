package balancer

import (
	"hash/crc32"
	"math/rand"
	"sync"
	"time"
)

const salt = "%#!"

func init() {
	factories[P2CBalancer] = NewP2C
}

type P2C struct {
	hosts   []*host
	rnd     *rand.Rand
	loadMap map[string]*host
	sync.RWMutex
}

type host struct {
	name string
	load uint64
}

func NewP2C(hosts []string) Balancer {
	p := &P2C{
		hosts:   []*host{},
		rnd:     rand.New(rand.NewSource(time.Now().UnixNano())),
		loadMap: map[string]*host{},
		RWMutex: sync.RWMutex{},
	}
	for _, h := range hosts {
		p.Add(h)
	}
	return p
}

func (p *P2C) Add(hostName string) {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.loadMap[hostName]; ok {
		return
	}
	h := &host{
		name: hostName,
		load: 0,
	}
	p.hosts = append(p.hosts, h)
	p.loadMap[hostName] = h
}

func (p *P2C) Remove(hostName string) {
	p.Lock()
	defer p.Unlock()
	if _, ok := p.loadMap[hostName]; !ok {
		return
	}

	delete(p.loadMap, hostName)

	for i, h := range p.hosts {
		if h.name == hostName {
			p.hosts = append(p.hosts[:i], p.hosts[i+1:]...)
			return
		}
	}
}

func (p *P2C) Balance(key string) (string, error) {
	p.Lock()
	defer p.Unlock()

	if len(p.hosts) == 0 {
		return "", nil
	}

	h1, h2 := p.hash(key)
	host := h1
	if p.loadMap[h1].load > p.loadMap[h2].load {
		host = h2
	}
	return host, nil
}

func (p *P2C) hash(key string) (string, string) {
	var h1, h2 string
	if len(key) == 0 {
		h1, h2 = p.hosts[p.rnd.Intn(len(p.hosts))].name, p.hosts[p.rnd.Intn(len(p.hosts))].name
		return h1, h2
	}
	h1 = p.hosts[crc32.ChecksumIEEE([]byte(key))%uint32(len(p.hosts))].name
	h2 = p.hosts[crc32.ChecksumIEEE([]byte(key+salt))%uint32(len(p.hosts))].name
	return h1, h2
}

func (p *P2C) Inc(host string) {
	p.Lock()
	defer p.Unlock()

	if h, ok := p.loadMap[host]; ok {
		h.load++
	}
}

func (p *P2C) Done(host string) {
	p.Lock()
	defer p.Unlock()

	if h, ok := p.loadMap[host]; ok && h.load > 0 {
		h.load--
	}
}
