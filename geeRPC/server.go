package geerpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"qitian/geeRPC/codec"
	"reflect"
	"strings"
	"sync"
	"time"
)

/*
报文发送格式：
	| Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{} |
	| <------      固定 JSON 编码      ------>  | <-------   编码方式由 CodeType 决定   ------->|

在一次连接中，Option 固定在报文的最开始，Header 和 Body 可以有多个，即报文可能是这样的：
	| Option | Header1 | Body1 | Header2 | Body2 | ...
*/

const MagicNumber = 0x3bef5c

type Option struct {
	MagicNumber    int           // 标记该请求是rpc请求
	CodeType       codec.Type    // 客户端选择不同的codec去解码body
	ConnectTimeout time.Duration // 0: 不设限
	HandleTimeout  time.Duration
}

var DefaultOption = &Option{
	MagicNumber:    MagicNumber,
	CodeType:       codec.GobType,
	ConnectTimeout: time.Second * 10,
	HandleTimeout:  time.Second * 10,
}

// RPC server
type Server struct {
	serviceMap sync.Map
}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer()

// accept 连接, 并对request响应
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		// 子协程处理request
		go server.ServeConn(conn)
	}
}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}

func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()
	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error:", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magic number: %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodeType]
	if f == nil {
		log.Printf("rpc server: invalid codec type: %s", opt.CodeType)
		return
	}
	server.serveCodec(f(conn))
}

var invalidRequest = struct{}{}

func (server *Server) serveCodec(cc codec.Codec) {
	sending := new(sync.Mutex) // 确保响应完毕
	wg := new(sync.WaitGroup)  // 确保所有request都完成
	// 允许接收多个请求，即多个 request header 和 request body
	for {
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		go server.handleRequest(cc, req, sending, wg, DefaultOption.HandleTimeout)
	}
	wg.Wait()
	_ = cc.Close()
}

// 注册服务方法
func (server *Server) Register(rcvr interface{}) error {
	s := newService(rcvr)
	if _, dup := server.serviceMap.LoadOrStore(s.name, s); dup {
		return errors.New("rpc: service already defined: " + s.name)
	}
	return nil
}

func Register(rcvr interface{}) error { return DefaultServer.Register(rcvr) }

// 从 serviceMap 中找到对应的 service
func (server *Server) findService(serviceMethod string) (svc *service, mtype *methodType, err error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot == -1 {
		err = errors.New("rpc server: service/method request ill-formed: " + serviceMethod)
		return
	}
	// 先在 serviceMap 中找到对应的 service 实例，
	// 再从 service 实例的 method 中，找到对应的 methodType
	serviceName, methodName := serviceMethod[:dot], serviceMethod[dot+1:]
	scvi, ok := server.serviceMap.Load(serviceName)
	if !ok {
		err = errors.New("rpc server: can't find service: " + serviceName)
		return
	}
	svc = scvi.(*service)
	mtype = svc.method[methodName]
	if mtype == nil {
		err = errors.New("rpc server: can't find method: " + methodName)
		return
	}
	return
}

type request struct {
	h            *codec.Header
	argv, replyv reflect.Value
	mtype        *methodType
	svc          *service
}

func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read header error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	req.svc, req.mtype, err = server.findService(h.ServiceMethod)
	if err != nil {
		return req, err
	}
	req.argv = req.mtype.newArgv()
	req.replyv = req.mtype.newReplyv()

	argvi := req.argv.Interface()
	if req.argv.Type().Kind() != reflect.Ptr {
		argvi = req.argv.Addr().Interface()
	}
	if err = cc.ReadBody(argvi); err != nil {
		log.Println("rpc server: read body err:", err)
		return req, err
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header, body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

// 处理客户端请求
func (server *Server) handleRequest(cc codec.Codec, req *request, sending *sync.Mutex, wg *sync.WaitGroup, timeout time.Duration) {
	defer wg.Done()
	// 相当于锁
	called := make(chan struct{})
	sent := make(chan struct{})
	go func() {
		err := req.svc.call(req.mtype, req.argv, req.replyv)
		called <- struct{}{}
		if err != nil {
			req.h.Error = err.Error()
			server.sendResponse(cc, req.h, invalidRequest, sending)
			sent <- struct{}{}
			return
		}
		server.sendResponse(cc, req.h, req.replyv.Interface(), sending)
		sent <- struct{}{}
	}()

	if timeout == 0 {
		<-called
		<-sent
		return
	}

	select {
	case <-time.After(timeout):
		req.h.Error = fmt.Sprintf("rpc server: request handle timeout: expect within %s", timeout)
		// 此时必还没有sendResponse
		server.sendResponse(cc, req.h, invalidRequest, sending)
	case <-called:
		<-sent
	}
}

const (
	connected        = "200 Connected to Gee RPC"
	defaultRPCPath   = "/_geerpc_"
	defaultDebugPath = "/debug/geerpc"
)

// ServeHTTP 实现了一个响应 RPC 请求的 Http.Handler接口
func (server *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method != "CONNECT" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must CONNECT\n")
		return
	}

	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Println("rpc hijacking ", req.RemoteAddr, ":", err.Error())
		return
	}
	_, _ = io.WriteString(conn, "HTTP/1.0 "+connected+"\n\n")
	server.ServeConn(conn)
}

// HandleHTTP 为 rpcPath 上的 RPC 消息注册 HTTP Handler
// 仍然需要调用 HTTP.Serve()，通常是在 GO 语句中
func (server *Server) HandleHTTP() {
	// 只需要实现接口 Handler 即可作为一个 HTTP Handler 处理 HTTP 请求
	// 接口 Handler 只定义了一个方法 ServeHTTP，实现该方法即可
	http.Handle(defaultRPCPath, server)
	http.Handle(defaultDebugPath, debugHTTP{server})
	log.Println("rpc server debug path:", defaultDebugPath)
}

// HandleHTTP 是 default server 注册 HTTP Handler 的一种方便方法
func HandleHTTP() {
	DefaultServer.HandleHTTP()
}
