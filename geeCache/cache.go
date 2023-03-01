package geecache

import (
	"qitian/geeCache/lru"
	"sync"
)

// cache.go 负责并发控制, 主要就是在lru上封装一层并发控制

type cache struct {
	mu         sync.Mutex // 支持并发, 必须有锁
	lru        *lru.Cache
	cacheBytes int64
}

// 向缓存中添加键值对
func (c *cache) put(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil { // 延迟初始化: 主要用于提高性能，并减少程序内存要求
		c.lru = lru.NewCache(c.cacheBytes, nil)
	}
	c.lru.Put(key, value)
}

// 从缓存中获取键值对
func (c *cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
