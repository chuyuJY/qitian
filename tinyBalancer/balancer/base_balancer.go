package balancer

import (
	"sync"
)

// BaseBalancer 实现 Balancer 接口
type BaseBalancer struct {
	sync.RWMutex
	hosts []string
}

func (b *BaseBalancer) Add(host string) {
	b.Lock()
	defer b.Unlock()
	for _, h := range b.hosts {
		if h == host {
			return
		}
	}
	b.hosts = append(b.hosts, host)
}

func (b *BaseBalancer) Remove(host string) {
	b.Lock()
	defer b.Unlock()
	for index, h := range b.hosts {
		if h == host {
			b.hosts = append(b.hosts[:index], b.hosts[index+1:]...)
			return
		}
	}
}

func (b *BaseBalancer) Balance(key string) (string, error) {
	return "", nil
}

func (b *BaseBalancer) Inc(_ string) {

}

func (b *BaseBalancer) Done(_ string) {

}
