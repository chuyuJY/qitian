package codec

import "io"

type Header struct {
	ServiceMethod string // 服务名.方法名 "Service.Method"
	Seq           uint64 // 请求的序号, 即某个请求的ID
	Error         string // 错误信息
}

// 对消息体进行编解码的接口
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

type NewCodecFunc func(io.ReadWriteCloser) Codec

type Type string

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

// init() 一定要小写开头
func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
