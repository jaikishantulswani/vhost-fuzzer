package scanner

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/dsecuredcom/vhost-fuzzer/pkg/config"
	"github.com/valyala/fasthttp"
)

type clientCache struct {
	clients         map[string]*fasthttp.Client
	mu              sync.Mutex
	followRedirects bool
}

func newClientCache(followRedirects bool) *clientCache {
	return &clientCache{
		clients:         make(map[string]*fasthttp.Client),
		followRedirects: followRedirects,
	}
}

func (cc *clientCache) getClient(ip string, cfg config.Config) *fasthttp.Client {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if client, ok := cc.clients[ip]; ok {
		return client
	}

	// Create a custom dialer with increased timeout for DNS resolution
	dialer := &fasthttp.TCPDialer{
		Concurrency:      1000,
		DNSCacheDuration: time.Minute, // Cache DNS results for 1 minute
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second, // Increase DNS resolution timeout
				}
				return d.DialContext(ctx, network, "8.8.8.8:53") // Use Google's public DNS
			},
		},
	}

	client := &fasthttp.Client{
		MaxIdleConnDuration: 1 * time.Second, // Close idle connections after 1 second
		MaxConnDuration:     5 * time.Second, // Close connections after 5 seconds
		ReadTimeout:         cfg.ReadTimeout,
		WriteTimeout:        cfg.WriteTimeout,
		Dial:                dialer.Dial, // Use the custom dialer
	}

	cc.clients[ip] = client
	return client
}
