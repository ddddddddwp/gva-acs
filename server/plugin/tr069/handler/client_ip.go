package handler

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
)

type ClientIPResolver struct {
	trusted []netip.Prefix
}

func NewClientIPResolver(trustedCIDRs []string) (*ClientIPResolver, error) {
	resolver := &ClientIPResolver{trusted: make([]netip.Prefix, 0, len(trustedCIDRs))}
	for _, raw := range trustedCIDRs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR: %w", err)
		}
		resolver.trusted = append(resolver.trusted, prefix.Masked())
	}
	return resolver, nil
}

func (r *ClientIPResolver) Resolve(request *http.Request) string {
	if request == nil {
		return ""
	}
	peer, ok := parseRemoteIP(request.RemoteAddr)
	if !ok {
		return ""
	}
	if !r.isTrusted(peer) {
		return peer.String()
	}

	if forwarded := strings.TrimSpace(request.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		hops := make([]netip.Addr, 0, len(parts))
		for _, part := range parts {
			address, err := netip.ParseAddr(strings.TrimSpace(part))
			if err != nil {
				return peer.String()
			}
			hops = append(hops, address.Unmap())
		}
		current := peer
		for index := len(hops) - 1; index >= 0; index-- {
			if !r.isTrusted(current) {
				return current.String()
			}
			current = hops[index]
		}
		return current.String()
	}
	if realIP := strings.TrimSpace(request.Header.Get("X-Real-IP")); realIP != "" {
		if address, err := netip.ParseAddr(realIP); err == nil {
			return address.Unmap().String()
		}
	}
	return peer.String()
}

func (r *ClientIPResolver) isTrusted(address netip.Addr) bool {
	if r == nil {
		return false
	}
	for _, prefix := range r.trusted {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

func parseRemoteIP(remoteAddress string) (netip.Addr, bool) {
	remoteAddress = strings.TrimSpace(remoteAddress)
	if host, _, err := net.SplitHostPort(remoteAddress); err == nil {
		address, parseErr := netip.ParseAddr(strings.Trim(host, "[]"))
		return address.Unmap(), parseErr == nil
	}
	address, err := netip.ParseAddr(strings.Trim(remoteAddress, "[]"))
	return address.Unmap(), err == nil
}

var clientIPResolverCache struct {
	sync.Mutex
	key      string
	resolver *ClientIPResolver
}

func remoteIPFromRequest(request *http.Request) string {
	trusted := config.CurrentRuntime().Settings.FileIngress.TrustedProxies
	key := strings.Join(trusted, "\x00")
	clientIPResolverCache.Lock()
	if clientIPResolverCache.resolver == nil || clientIPResolverCache.key != key {
		resolver, err := NewClientIPResolver(trusted)
		if err != nil {
			resolver, _ = NewClientIPResolver(nil)
		}
		clientIPResolverCache.key = key
		clientIPResolverCache.resolver = resolver
	}
	resolver := clientIPResolverCache.resolver
	clientIPResolverCache.Unlock()
	return resolver.Resolve(request)
}
