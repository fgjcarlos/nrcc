package middleware

import (
	"net"
	"net/http"
	"testing"
)

func mustCIDRs(t *testing.T, specs ...string) []*net.IPNet {
	t.Helper()
	return parseTrustedProxies(joinComma(specs))
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}

func reqWith(remoteAddr, xff string) *http.Request {
	r := &http.Request{Header: http.Header{}, RemoteAddr: remoteAddr}
	if xff != "" {
		r.Header.Set("X-Forwarded-For", xff)
	}
	return r
}

// TestExtractIP_IgnoresXFFWithoutTrustedProxies is the #280 regression: with no
// trusted proxies configured, a client-supplied X-Forwarded-For must be ignored
// so it cannot rotate rate-limit buckets — the peer address is authoritative.
func TestExtractIP_IgnoresXFFWithoutTrustedProxies(t *testing.T) {
	r := reqWith("203.0.113.7:5555", "1.2.3.4")
	if got := extractIP(r, nil); got != "203.0.113.7" {
		t.Fatalf("expected peer 203.0.113.7, got %q (XFF spoof not blocked)", got)
	}
}

func TestExtractIP_HonorsXFFFromTrustedProxy(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	r := reqWith("10.0.0.5:443", "1.2.3.4, 10.0.0.5")
	if got := extractIP(r, trusted); got != "1.2.3.4" {
		t.Fatalf("expected client 1.2.3.4 from trusted proxy, got %q", got)
	}
}

func TestExtractIP_IgnoresXFFFromUntrustedPeer(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	r := reqWith("203.0.113.7:5555", "1.2.3.4")
	if got := extractIP(r, trusted); got != "203.0.113.7" {
		t.Fatalf("expected peer 203.0.113.7 (untrusted peer), got %q", got)
	}
}

func TestExtractIP_FallsBackToPeerWhenNoXFF(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	r := reqWith("10.0.0.5:443", "")
	if got := extractIP(r, trusted); got != "10.0.0.5" {
		t.Fatalf("expected peer 10.0.0.5, got %q", got)
	}
}

func TestParseTrustedProxies(t *testing.T) {
	nets := parseTrustedProxies("10.0.0.0/8, 192.168.1.1 , , 2001:db8::/32")
	if len(nets) != 3 {
		t.Fatalf("expected 3 networks, got %d", len(nets))
	}
	// single IPv4 should become /32 and match exactly
	if !ipInNets("192.168.1.1", nets) {
		t.Error("192.168.1.1/32 should match")
	}
	if ipInNets("192.168.1.2", nets) {
		t.Error("192.168.1.2 should not match a /32")
	}
}
