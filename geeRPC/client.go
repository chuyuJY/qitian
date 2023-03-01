package geerpc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"qitian/geeRPC/codec"
	"strings"
	"sync"
	"time"
)

// client: rpc 客户端

// 请求格式
type Call struct {
	Seq           uint64
	ServiceMethod string // service.method
	Args          interface{}
	Reply         interface{}
	Error         error
	Done          chan *Call
}

// 调用结束时，会调用 call.done() 通知调用方
func (call *Call) done() {
	call.Done <- call
}

type Client struct {
	cc       codec.Codec      // 消息的编解码器
	opt      *Option          // 标记
	sending  sync.Mutex       // 保证请求的有序发送
	header   codec.Header     // 请求的消息头
	mu       sync.Mutex       // 保证有序
	seq      uint64           // 请求编号
	pending  map[uint64]*Call // 存储未处理完的请求 key: 编号 val: Call实例
	closing  bool             // 用户主动关闭
	shutdown bool             // 错误导致关闭
}

var ErrorShutdown = errors.New("connection is shutdown")

// 创建 Client 实例
func NewClient(conn net.Conn, opt *Option) (*Client, error) {
	// 判断编码方式是否合法
	f := codec.NewCodecFuncMap[opt.CodeType]
	if f == nil {
		err := fmt.Errorf("invalid codec type: %s", opt.CodeType)
		log.Println("rpc client: codec error:", err)
		return nil, err
	}
	// 发送 Option 信息给服务端, 协商编码方式
	if err := json.NewEncoder(conn).Encode(opt); err != nil {
		log.Println("rpc client: options error:", err)
		_ = conn.Close()
		return nil, err
	}
	// 创建一个子协程调用 receive() 接收响应
	return newClientCodec(f(conn), opt), nil
}

func newClientCodec(cc codec.Codec, opt *Option) *Client {
	client := &Client{
		seq:     1,
		cc:      cc,
		opt:     opt,
		pending: make(map[uint64]*Call),
	}
	go client.receive()
	return client
}

func (client *Client) Close() error {
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.closing {
		return ErrorShutdown
	}
	client.closing = true
	return client.cc.Close()
}

func (client *Client) IsAvailable() bool {
	client.mu.Lock()
	defer client.mu.Unlock()
	return !client.shutdown && !client.closing
}

// 将 call实例 添加到 client.pending 中，并更新 client.seq
func (client *Client) registerCall(call *Call) (uint64, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	if client.closing || client.shutdown {
		return 0, ErrorShutdown
	}
	call.Seq = client.seq
	client.pending[call.Seq] = call
	client.seq++
	return call.Seq, nil
}

// 根据 seq，从 client.pending 中移除对应的 call，并返回该 call
func (client *Client) removeCall(seq uint64) *Call {
	client.mu.Lock()
	defer client.mu.Unlock()
	call := client.pending[seq]
	delete(client.pending, seq)
	return call
}

// 服务端或客户端发生错误时调用，将 shutdown 设置为 true，且将错误信息通知所有 pending 状态的 call
func (client *Client) terminateCalls(err error) {
	client.sending.Lock()
	defer client.sending.Unlock()
	client.mu.Lock()
	defer client.mu.Unlock()
	client.shutdown = true
	for _, call := range client.pending {
		call.Error = err
		call.done()
	}
}

// 接收功能
func (client *Client) receive() {
	var err error
	for err == nil {
		var h codec.Header
		if err = client.cc.ReadHeader(&h); err != nil {
			break
		}
		call := client.removeCall(h.Seq)
		switch {
		case call == nil:
			err = client.cc.ReadBody(nil)
		case h.Error != "":
			call.Error = fmt.Errorf(h.Error)
			err = client.cc.ReadBody(nil)
			call.done()
		default:
			err = client.cc.ReadBody(call.Reply)
			if err != nil {
				call.Error = errors.New("reading body " + err.Error())
			}
			call.done()
		}
	}
	client.terminateCalls(err)
}

// 发送功能
func (client *Client) send(call *Call) {
	client.sending.Lock()
	defer client.sending.Unlock()

	// 注册 call
	seq, err := client.registerCall(call)
	if err != nil {
		call.Error = err
		call.done()
		return
	}

	// 请求头
	client.header.Seq = seq
	client.header.ServiceMethod = call.ServiceMethod
	client.header.Error = ""

	// encode并发送请求
	if err := client.cc.Write(&client.header, call.Args); err != nil {
		call := client.removeCall(seq)
		if call == nil {
			call.Error = err
			call.done()
		}
	}
}

// RPC调用: 异步接口
func (client *Client) Go(serviceMethod string, args, reply interface{}, done chan *Call) *Call {
	if done == nil {
		done = make(chan *Call, 10)
	} else if cap(done) == 0 {
		log.Panic("rpc client: done channel is unbuffered")
	}
	call := &Call{
		ServiceMethod: serviceMethod,
		Args:          args,
		Reply:         reply,
		Done:          done,
	}
	client.send(call)
	return call
}

// RPC调用: 同步接口
// 超时处理: 使用 context.WithTimeout 创建具备超时检测能力的 context 对象来控制
func (client *Client) Call(ctx context.Context, serviceMethod string, args, reply interface{}) error {
	call := client.Go(serviceMethod, args, reply, make(chan *Call, 1))
	select {
	case <-ctx.Done():
		client.removeCall(call.Seq)
		return errors.New("rpc client: call failed: " + ctx.Err().Error())
	case call := <-call.Done:
		return call.Error
	}
}

// 解析 Option
func parseOptions(opts ...*Option) (*Option, error) {
	if len(opts) == 0 || opts[0] == nil {
		return DefaultOption, nil
	}
	if len(opts) != 1 {
		return nil, errors.New("number of opts is more than 1")
	}
	opt := opts[0]
	opt.MagicNumber = DefaultOption.MagicNumber
	if opt.CodeType == "" {
		opt.CodeType = DefaultOption.CodeType
	}
	return opt, nil
}

type clientResult struct {
	client *Client
	err    error
}

type newClientFunc func(conn net.Conn, opt *Option) (client *Client, err error)

func dialTimeout(f newClientFunc, network, address string, opts ...*Option) (client *Client, err error) {
	opt, err := parseOptions(opts...)
	if err != nil {
		return nil, err
	}
	// 连接超时处理
	conn, err := net.DialTimeout(network, address, opt.ConnectTimeout)
	if err != nil {
		return nil, err
	}
	// 闭包
	defer func() {
		if err != nil {
			_ = conn.Close()
		}
	}()
	// 连接超时处理
	ch := make(chan clientResult)
	go func() {
		client, err := f(conn, opt)
		ch <- clientResult{client: client, err: err}
	}()
	// 默认不设限
	if opt.ConnectTimeout == 0 {
		result := <-ch
		return result.client, result.err
	}
	// 设限
	select {
	// 超时
	case <-time.After(opt.ConnectTimeout):
		return nil, fmt.Errorf("rpc client: connect timeout: expect within %s", opt.ConnectTimeout)
	// 没超时
	case result := <-ch:
		return result.client, result.err
	}
}

// 连接功能: 建立和 RPC 服务端的连接, 便于用户传入服务端地址
func Dial(network, address string, opts ...*Option) (client *Client, err error) {
	return dialTimeout(NewClient, network, address, opts...)
}

// NewHTTPClient 通过 HTTP 作为传输协议新建客户端实例
func NewHTTPClient(conn net.Conn, opt *Option) (*Client, error) {
	_, _ = io.WriteString(conn, fmt.Sprintf("CONNETC %s HTTP/1.0\n\n", defaultRPCPath))

	// 在转换成 RPC 前，需要成功的 HTTP Response
	resp, err := http.ReadResponse(bufio.NewReader(conn), &http.Request{Method: "CONNECT"})
	if err == nil && resp.Status == connected {
		return NewClient(conn, opt)
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	return nil, err
}

// DialHTTP 连接到位于指定网络地址的 HTTP RPC 服务器，监听默认的 HTTP RPC 路径
func DialHTTP(network, address string, opts ...*Option) (*Client, error) {
	return dialTimeout(NewHTTPClient, network, address, opts...)
}

func XDial(rpcAddr string, opts ...*Option) (*Client, error) {
	parts := strings.Split(rpcAddr, "@")
	if len(parts) != 2 {
		return nil, fmt.Errorf("rpc client err: wrong format '%s', expect protocol@addr", rpcAddr)
	}
	protocol, addr := parts[0], parts[1]
	switch protocol {
	case "http":
		return DialHTTP("tcp", addr, opts...)
	default:
		//  tcp, unix or other transport protocol
		return Dial(protocol, addr, opts...)
	}
}
