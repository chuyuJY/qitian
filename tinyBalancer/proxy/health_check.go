package proxy

import (
	"log"
	"time"
)

var HealthCheckTimeout = 5 * time.Second

func (h *HTTPProxy) ReadAlive(url string) bool {
	h.Lock()
	defer h.Unlock()
	return h.alive[url]
}

func (h *HTTPProxy) SetAlive(url string, alive bool) {
	h.Lock()
	defer h.Unlock()
	h.alive[url] = alive
}

func (h *HTTPProxy) HealthCheck(interval uint) {
	for host := range h.hostMap {
		go h.healthCheck(host, interval)
	}
}

func (h *HTTPProxy) healthCheck(host string, interval uint) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second) // 定时器
	// 每 time.Duration(interval) * time.Second 检查一次
	for range ticker.C {
		if !IsBackendAlive(host) && h.ReadAlive(host) {
			log.Printf("Site unreachable, remove %s from load balancer.", host)
			h.SetAlive(host, false)
			h.lb.Remove(host)
		} else if IsBackendAlive(host) && !h.ReadAlive(host) {
			log.Printf("Site reachable, add %s to load balancer.", host)
			h.SetAlive(host, true)
			h.lb.Add(host)
		}
	}
}
