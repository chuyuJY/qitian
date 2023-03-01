package consistenthash

import (
	"strconv"
	"testing"
)

func TestHash(t *testing.T) {
	hash := NewMap(3, func(data []byte) uint32 {
		i, _ := strconv.Atoi(string(data))
		return uint32(i)
	})

	// Given the above hash function, this will give replicas with "hashes":
	// 2, 4, 6; 12, 14, 16; 22, 24, 26;
	hash.Add("6", "4", "2")
	testCases := map[string]string{
		"2":  "2",
		"11": "2",
		"23": "4",
		"27": "2",
	}

	// 测试均衡缓存
	for k, v := range testCases {
		if target := hash.Get(k); target != v {
			t.Errorf("Asking for %s, should have yielded %s, but have yield %s", k, v, target)
		}
	}

	// 测试新增节点
	// 8, 18, 28
	hash.Add("8")
	testCases["27"] = "8"
	for k, v := range testCases {
		if target := hash.Get(k); target != v {
			t.Errorf("Asking for %s, should have yielded %s, but have yield %s", k, v, target)
		}
	}
}
