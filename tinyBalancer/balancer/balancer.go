package balancer

import "errors"

var (
	NoHostError             = errors.New("no host")
	AlgorithmSupportedError = errors.New("algorithm not supported")
)

const (
	IPHashBalancer         = "ip-hash"
	ConsistentHashBalancer = "consistent-hash"
	P2CBalancer            = "p2c"
	RandomBalancer         = "random"
	R2Balancer             = "round-robin"
	LeastLoadBalancer      = "least-load"
	BoundedBalancer        = "bounded"
)

type Balancer interface {
	Add(string)                     // 为负载均衡器添加主机
	Remove(string)                  // 为负载均衡器删除主机
	Balance(string) (string, error) // 负载均衡策略：根据传入的key值选取主机响应
	Inc(string)                     // 对代理主机的连接数+1
	Done(string)                    // 对代理主机的连接数-1
}

type Factory func([]string) Balancer

var factories = make(map[string]Factory)

// Build 根据传入的算法，创建Balancer实例
func Build(algorithm string, hosts []string) (Balancer, error) {
	if factory, ok := factories[algorithm]; ok {
		return factory(hosts), nil
	}
	return nil, AlgorithmSupportedError
}
