package lru

import "testing"

type String string

func (d String) Len() int {
	return len(d)
}

func TestLru(t *testing.T) {
	lru := NewCache(int64(0), nil)
	lru.Put("key1", String("1234"))
	if v, ok := lru.Get("key1"); !ok || string(v.(String)) != "1234" {
		t.Fatalf("cache hit key1=1234 failed")
	}
	if _, ok := lru.Get("key2"); ok {
		t.Fatalf("cache miss key2 failed")
	}
}
