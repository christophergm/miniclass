package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrustedProxyRealIP(t *testing.T) {
	const originalRemoteAddr = "203.0.113.20:4321"

	tests := []struct {
		name          string
		remoteAddr    string
		trustedCIDRs  []string
		xForwardedFor []string
		xRealIP       []string
		wantRemote    string
	}{
		{
			name:          "direct request ignores forwarded headers",
			remoteAddr:    originalRemoteAddr,
			xForwardedFor: []string{"198.51.100.10"},
			wantRemote:    originalRemoteAddr,
		},
		{
			name:          "trusted proxy uses rightmost untrusted address",
			remoteAddr:    "10.0.0.8:443",
			trustedCIDRs:  []string{"10.0.0.0/8"},
			xForwardedFor: []string{"198.51.100.10, 10.0.0.9"},
			wantRemote:    "198.51.100.10",
		},
		{
			name:          "duplicate forwarded headers are merged",
			remoteAddr:    "10.0.0.8:443",
			trustedCIDRs:  []string{"10.0.0.0/8"},
			xForwardedFor: []string{"127.0.0.1", "198.51.100.10, 10.0.0.9"},
			wantRemote:    "198.51.100.10",
		},
		{
			name:         "trusted proxy falls back to real IP header",
			remoteAddr:   "10.0.0.8:443",
			trustedCIDRs: []string{"10.0.0.0/8"},
			xRealIP:      []string{"198.51.100.10"},
			wantRemote:   "198.51.100.10",
		},
		{
			name:         "untrusted peer cannot use real IP header",
			remoteAddr:   originalRemoteAddr,
			trustedCIDRs: []string{"10.0.0.0/8"},
			xRealIP:      []string{"198.51.100.10"},
			wantRemote:   originalRemoteAddr,
		},
		{
			name:          "malformed forwarded chain fails closed",
			remoteAddr:    "10.0.0.8:443",
			trustedCIDRs:  []string{"10.0.0.0/8"},
			xForwardedFor: []string{"198.51.100.10, malformed"},
			wantRemote:    "10.0.0.8:443",
		},
		{
			name:          "invalid proxy configuration fails closed",
			remoteAddr:    "10.0.0.8:443",
			trustedCIDRs:  []string{"not-a-cidr"},
			xForwardedFor: []string{"198.51.100.10"},
			wantRemote:    "10.0.0.8:443",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotRemote string
			handler := TrustedProxyRealIP(test.trustedCIDRs...)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				gotRemote = r.RemoteAddr
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			request.RemoteAddr = test.remoteAddr
			for _, value := range test.xForwardedFor {
				request.Header.Add("X-Forwarded-For", value)
			}
			for _, value := range test.xRealIP {
				request.Header.Add("X-Real-IP", value)
			}

			handler.ServeHTTP(httptest.NewRecorder(), request)
			if gotRemote != test.wantRemote {
				t.Fatalf("RemoteAddr = %q, want %q", gotRemote, test.wantRemote)
			}
		})
	}
}
