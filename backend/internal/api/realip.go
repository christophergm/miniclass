package api

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// TrustedProxyRealIP updates RemoteAddr only when the immediate peer is in a
// configured trusted proxy network. Forwarded addresses are then walked from
// right to left, so the first address outside the trusted networks is used.
// Invalid forwarding data leaves RemoteAddr unchanged.
func TrustedProxyRealIP(trustedProxyCIDRs ...string) func(http.Handler) http.Handler {
	prefixes := parseTrustedProxyCIDRs(trustedProxyCIDRs)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isTrustedProxy(r.RemoteAddr, prefixes) {
				if clientIP, ok := forwardedClientIP(r, prefixes); ok {
					r.RemoteAddr = clientIP.String()
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseTrustedProxyCIDRs(cidrs []string) []netip.Prefix {
	prefixes := make([]netip.Prefix, 0, len(cidrs))
	for _, cidr := range cidrs {
		prefix, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err == nil {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

func isTrustedProxy(remoteAddr string, prefixes []netip.Prefix) bool {
	addr, ok := parseRemoteAddr(remoteAddr)
	if !ok {
		return false
	}
	return addressInPrefixes(addr, prefixes)
}

func parseRemoteAddr(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(host)
	if err != nil || addr.Zone() != "" {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func forwardedClientIP(r *http.Request, prefixes []netip.Prefix) (netip.Addr, bool) {
	if values := r.Header.Values("X-Forwarded-For"); len(values) > 0 {
		return rightmostUntrustedForwardedIP(values, prefixes)
	}
	if values := r.Header.Values("X-Real-IP"); len(values) > 0 {
		return parseForwardedIP(values[len(values)-1])
	}
	return netip.Addr{}, false
}

func rightmostUntrustedForwardedIP(values []string, prefixes []netip.Prefix) (netip.Addr, bool) {
	for valueIndex := len(values) - 1; valueIndex >= 0; valueIndex-- {
		entries := strings.Split(values[valueIndex], ",")
		for entryIndex := len(entries) - 1; entryIndex >= 0; entryIndex-- {
			addr, ok := parseForwardedIP(entries[entryIndex])
			if !ok {
				return netip.Addr{}, false
			}
			if !addressInPrefixes(addr, prefixes) {
				return addr, true
			}
		}
	}
	return netip.Addr{}, false
}

func parseForwardedIP(value string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil || addr.Zone() != "" {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

func addressInPrefixes(addr netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
