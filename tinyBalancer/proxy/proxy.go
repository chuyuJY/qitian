package proxy

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"qitian/tinyBalancer/balancer"
	"strings"
	"sync"
	"time"
)

var (
	XRealIP       = http.CanonicalHeaderKey("X-Real-IP")
	XProxy        = http.CanonicalHeaderKey("X-Proxy")
	XForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
)

var (
	ReverseProxy = "Balancer-Reverse-Proxy"
)

var ConnectionTimeout = 3 * time.Second

type HTTPProxy struct {
	hostMap      map[string]*httputil.ReverseProxy // 需要反向代理的主机
	lb           balancer.Balancer                 // 负载均衡器
	sync.RWMutex                                   // 保护 alive 的并发安全
	alive        map[string]bool                   // 反向代理的主机是否处于健康状态
}

func NewHTTPProxy(targetHosts []string, algorithm string) (*HTTPProxy, error) {
	hosts := []string{}
	hostMap := map[string]*httputil.ReverseProxy{}
	alive := map[string]bool{}

	for _, targetHost := range targetHosts {
		url, err := url.Parse(targetHost) // 解析url
		if err != nil {
			return nil, err
		}
		proxy := httputil.NewSingleHostReverseProxy(url) // 生成反向代理
		originDirector := proxy.Director
		// 修改请求
		proxy.Director = func(request *http.Request) {
			originDirector(request)
			request.Header.Set(XProxy, ReverseProxy)
			request.Header.Set(XRealIP, GetIP(request))
		}
		host := GetHost(url)        // 获取主机名
		alive[host] = true          // 主机默认存活
		hostMap[host] = proxy       // 主机的代理
		hosts = append(hosts, host) // 主机列表
	}
	lb, err := balancer.Build(algorithm, hosts) // 根据算法构建负载均衡器
	if err != nil {
		return nil, err
	}
	return &HTTPProxy{
		hostMap: hostMap,
		lb:      lb,
		alive:   alive,
	}, nil
}

// ServeHTTP 反向代理处理请求
func (h *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("proxy causes panic: %s", err)
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(err.(error).Error()))
		}
	}()

	host, err := h.lb.Balance(GetIP(r)) // 根据 IP 分配主机host
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(fmt.Sprintf("balance error: %s", err.Error())))
		return
	}

	h.lb.Inc(host)
	defer h.lb.Done(host)
	h.hostMap[host].ServeHTTP(w, r) // 响应该请求
}

// GetIP 获得客户端IP
// 若客户端IP 为 192.168.1.1，通过代理 192.168.2.5 和 192.168.2.6
// X-Forwarded-For的值可能为 [192.168.2.5 ,192.168.2.6]
// X-Real-IP的值为 192.168.1.1
func GetIP(req *http.Request) string {
	clientIP, _, _ := net.SplitHostPort(req.RemoteAddr) // 解析客户端地址
	// 有可能是通过代理
	if xff := req.Header.Get(XForwardedFor); xff != "" { // 试图在 X-Forwarded-For 获取客户端IP
		s := strings.Index(xff, ",")
		if s == -1 {
			s = len(xff)
		}
		clientIP = xff[:s]
	} else if realIP := req.Header.Get(XRealIP); realIP != "" { // 试图在 X-Real-IP 获取IP
		clientIP = realIP
	}
	return clientIP
}

func GetHost(url *url.URL) string {
	if _, _, err := net.SplitHostPort(url.Host); err == nil {
		return url.Host
	}
	if url.Scheme == "http" {
		return fmt.Sprintf("%s:%s", url.Host, "80")
	}
	if url.Scheme == "https" {
		return fmt.Sprintf("%s:%s", url.Host, "443")
	}
	return url.Host
}

// IsBackendAlive 通过建立TCP连接检测的方式来判断代理主机的健康状态
func IsBackendAlive(host string) bool {
	addr, err := net.ResolveTCPAddr("tcp", host)
	if err != nil {
		return false
	}
	resolveAddr := fmt.Sprintf("%s:%d", addr.IP, addr.Port)
	conn, err := net.DialTimeout("tcp", resolveAddr, ConnectionTimeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
