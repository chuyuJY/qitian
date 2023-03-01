package geecache

// byteview.go 负责缓存值的抽象与封装

type ByteView struct {
	b []byte // 存储缓存值 只读的
}

// 实现Value接口
func (bv ByteView) Len() int {
	return len(bv.b)
}

// 返回一个拷贝，防止缓存值被外部程序修改
func (bv *ByteView) ByteSlice() []byte {
	return cloneBytes(bv.b)
}

func (bv *ByteView) String() string {
	return string(bv.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
