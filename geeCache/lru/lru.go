package lru

import (
	"container/list"
)

// lru.go 负责缓存淘汰策略

type Cache struct {
	maxBytes  int64                         // 允许使用的最大内存
	nbytes    int64                         // 当前已使用的内存
	ll        *list.List                    // 双向链表
	cache     map[string]*list.Element      // 键是字符串，值是双向链表中对应节点的指针
	OnEvicted func(key string, value Value) // 某条记录被移除时的回调函数，可以为 nil
}

// 双向链表节点的数据类型
type entry struct {
	key   string
	value Value
}

// 为了通用性, 允许值是实现了 Value 接口的任意类型
// 该接口只包含了一个方法 Len() int，用于返回值所占用的内存大小
type Value interface {
	Len() int
}

// 初始化
func NewCache(maxBytes int64, onEvicated func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		nbytes:    0,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicated,
	}
}

// Get 查找功能
func (c *Cache) Get(key string) (Value, bool) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		e := element.Value.(*entry)
		return e.value, true
	}
	return nil, false
}

// Put 更新功能: 增加/更新
func (c *Cache) Put(key string, value Value) {
	if element, ok := c.cache[key]; ok {
		c.ll.MoveToFront(element)
		e := element.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(e.value.Len())
		e.value = value
	} else {
		e := &entry{
			key:   key,
			value: value,
		}
		element := c.ll.PushFront(e)
		c.nbytes += int64(len(key)) + int64(value.Len())
		c.cache[key] = element
	}

	// 若缓存空间超出, 删除队尾元素
	for c.maxBytes != 0 && c.nbytes > c.maxBytes {
		c.RemoveOldest()
	}
}

// RemoveOldest 删除最久没使用过的元素
func (c *Cache) RemoveOldest() {
	element := c.ll.Back()
	if element != nil {
		c.ll.Remove(element)
		old := element.Value.(*entry)
		delete(c.cache, old.key)
		// 更新缓存空间
		c.nbytes -= int64(len(old.key)) + int64(old.value.Len())

		if c.OnEvicted != nil {
			c.OnEvicted(old.key, old.value)
		}
	}
}

func (c Cache) Len() int {
	return c.ll.Len()
}
