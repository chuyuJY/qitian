package balancer

import (
	"sync"

	fibHeap "github.com/starwander/GoFibonacciHeap"
)

func init() {
	factories[LeastLoadBalancer] = NewLeastLoad
}

func (h *host) Tag() interface{} {
	return h.name
}

func (h *host) Key() float64 {
	return float64(h.load)
}

type LeastLoad struct {
	sync.RWMutex
	heap *fibHeap.FibHeap
}

func NewLeastLoad(hosts []string) Balancer {
	ll := &LeastLoad{
		RWMutex: sync.RWMutex{},
		heap:    fibHeap.NewFibHeap(),
	}
	for _, host := range hosts {
		ll.Add(host)
	}
	return ll
}

func (ll *LeastLoad) Add(hostName string) {
	ll.Lock()
	defer ll.Unlock()
	if ok := ll.heap.GetValue(hostName); ok != nil {
		return
	}
	_ = ll.heap.InsertValue(&host{hostName, 0})
}

func (ll *LeastLoad) Remove(hostName string) {
	ll.Lock()
	defer ll.Unlock()
	if ok := ll.heap.GetValue(hostName); ok == nil {
		return
	}
	_ = ll.heap.Delete(hostName)
}

func (ll *LeastLoad) Balance(_ string) (string, error) {
	ll.Lock()
	defer ll.Unlock()
	if ll.heap.Num() == 0 {
		return "", NoHostError
	}
	return ll.heap.MinimumValue().Tag().(string), nil
}

func (ll *LeastLoad) Inc(hostName string) {
	ll.Lock()
	defer ll.Unlock()
	if ok := ll.heap.GetValue(hostName); ok == nil {
		return
	}
	h := ll.heap.GetValue(hostName)
	h.(*host).load++
	_ = ll.heap.IncreaseKeyValue(h)
}

func (ll *LeastLoad) Done(hostName string) {
	ll.Lock()
	defer ll.Unlock()
	if ok := ll.heap.GetValue(hostName); ok == nil {
		return
	}
	h := ll.heap.GetValue(hostName)
	h.(*host).load--
	_ = ll.heap.DecreaseKeyValue(h)
}
