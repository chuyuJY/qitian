package geecache

import (
	"errors"
	"qitian/geeCache/geeCachePb/pb"
	singleflight "qitian/geeCache/singleFlight"

	"log"
	"sync"
)

// geecache.go 负责与外部交互，控制缓存存储和获取的主流程

/*
	                        是
接收 key --> 检查是否被缓存 -----> 返回缓存值 ⑴
                |  否                         是
                |-----> 是否应当从远程节点获取 -----> 与远程节点交互 --> 返回缓存值 ⑵
                            |  否
                            |-----> 调用`回调函数`，获取值并添加到缓存 --> 返回缓存值 ⑶
*/

// 未命中时从数据源获取数据
type Getter interface {
	Get(key string) ([]byte, error) // 回调函数
}

// 函数类型实现某一个接口，称之为接口型函数。
// 方便使用者在调用时既能够传入函数作为参数，也能够传入实现了该接口的结构体作为参数。
// 是一个将函数转换为接口的技巧
type GetterFunc func(key string) ([]byte, error)

// 实现Getter接口
func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key)
}

var (
	// 对map的并发访问需要上锁
	mu     sync.Mutex
	groups = make(map[string]*Group)
)

// 获取特定名称的Group
func GetGroup(name string) *Group {
	mu.Lock()
	defer mu.Unlock()
	g := groups[name]
	return g
}

// 最核心的部分！
// 一个Group可被认为是一个缓存的命名空间
type Group struct {
	name      string              // 缓存名字
	getter    Getter              // 缓存未命中时获取源数据的回调(callback)
	mainCache cache               // 单机并发缓存
	peers     PeerPicker          // 分布式缓存
	loader    *singleflight.Group // 确保每个key只请求一次, 即 load 过程只会调用一次
}

func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		mainCache: cache{cacheBytes: cacheBytes},
		getter:    getter,
		loader:    &singleflight.Group{},
	}
	groups[name] = g
	return g
}

// 注册一个PeerPicker, 用来选择远端peer
func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 获取键值对
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, errors.New("key is required")
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache hit]")
		return v, nil
	}

	// 若cache没有, 需要从数据源获取
	return g.load(key)
}

// 从源数据获取: 分布式/本地
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		// 先从远端peer获取
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		// 没找到, 从本地源数据获取
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

// 访问远程节点, 获取缓存值
func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	req := &pb.Request{
		Group: g.name,
		Key:   key,
	}
	resp := &pb.Response{}
	err := peer.Get(req, resp)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: cloneBytes(resp.Value)}, nil
}

// 从本地源数据获取
func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, nil
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 将源数据添加到缓存 mainCache 中
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.put(key, value)
}
