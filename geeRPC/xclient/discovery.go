package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

const (
	RandomSelect     SelectMode = iota // 随机
	RoundRobinSelect                   // 轮询
)

// 服务发现接口
type Discovery interface {
	Refresh() error                      // 从注册中心更新注册列表
	Update(servers []string) error       // 手动更新服务列表
	Get(mode SelectMode) (string, error) // 根据负载均衡策略，选择一个服务实例
	GetAll() ([]string, error)           // 返回所有服务实例
}

// 一个不需要注册中心，服务列表由手工维护的服务发现的结构体
type MultiServerDiscovery struct {
	r       *rand.Rand   // 随机数
	mu      sync.RWMutex // 锁
	servers []string     // 服务列表
	index   int          // 记录轮询算法选择到的位置
}

func NewMultiServerDiscovery(servers []string) *MultiServerDiscovery {
	// r: 初始化时使用时间戳设定随机数种子, 避免每次产生相同的随机数序列
	d := &MultiServerDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	// index: 为了避免每次从 0 开始，初始化时随机设定一个值。
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

// 实现 Discovery 接口
var _ Discovery = (*MultiServerDiscovery)(nil)

// 手动, 因此此处直接返回
func (d *MultiServerDiscovery) Refresh() error {
	return nil
}

func (d *MultiServerDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}

func (d *MultiServerDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n]
		d.index = (d.index + 1) % n
		return s, nil
	default:
		return "", errors.New("rpc discovery: not supported selected mode")
	}
}

func (d *MultiServerDiscovery) GetAll() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	servers := make([]string, len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}
