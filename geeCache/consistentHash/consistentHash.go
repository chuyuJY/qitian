package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

// consistentHash 实现一致性哈希算法

// Hash maps bytes to uint32
type Hash func(data []byte) uint32 // 默认为 crc32.ChecksumIEEE 算法

// Map contains all hashed keys
type Map struct {
	hash     Hash           // Hash 函数
	replicas int            // 虚拟节点倍数
	keys     []int          // 哈希环 sorted
	hashMap  map[int]string // 虚拟节点与真实节点的映射表 键是虚拟节点的哈希值，值是真实节点的名称
}

func NewMap(replicas int, fn Hash) *Map {
	m := &Map{
		hash:     fn,
		replicas: replicas,
		hashMap:  map[int]string{},
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 添加真实节点/机器  key 是真实节点名
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// 选择节点
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key)))
	// 顺时针寻找第一个匹配的虚拟节点的下标(不一定是key对应的那个)
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	return m.hashMap[m.keys[idx%len(m.keys)]] // 若[0, n)中没找到, 会返回 n, 所以此处取 %n
}
