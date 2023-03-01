package balancer

import "sync"

// RoundRobin 轮询算法是最经典的负载均衡算法之一，负载均衡器将请求依次分发到后端的每一个主机中
type RoundRobin struct {
	i uint64 // 请求序号
	BaseBalancer
}

func init() {
	factories[R2Balancer] = NewRoundRobin
}

func NewRoundRobin(hosts []string) Balancer {
	return &RoundRobin{
		i: 0,
		BaseBalancer: BaseBalancer{
			RWMutex: sync.RWMutex{},
			hosts:   hosts,
		},
	}
}

func (r *RoundRobin) Balance(_ string) (string, error) {
	r.Lock()
	defer r.Unlock()
	if len(r.hosts) == 0 {
		return "", nil
	}
	host := r.hosts[r.i%uint64(len(r.hosts))]
	r.i++
	return host, nil
}
