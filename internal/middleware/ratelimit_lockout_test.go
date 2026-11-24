package middleware

import (
	"bytes"
	"log"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// #1021: six failures from one address lock that address for the user, not
// the user everywhere.
func TestAuthUser_LockoutIsPerAddress(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())
	for i := 0; i < maxAttempts; i++ {
		rl.RecordAuthUser("Admin", "203.0.113.1")
	}
	if blocked, _ := rl.CheckAuthUser("admin", "203.0.113.1"); !blocked {
		t.Fatal("the failing address must be locked for the user")
	}
	if blocked, _ := rl.CheckAuthUser("admin", "203.0.113.2"); blocked {
		t.Fatal("another address must not be locked out by the first one's failures")
	}
}

// The global per-user bucket is slower: it only trips after UserGlobalMaxAttempts
// failures spread over many addresses.
func TestAuthUser_GlobalLimitTripsAcrossAddresses(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())
	for i := 0; i < UserGlobalMaxAttempts-1; i++ {
		rl.RecordAuthUser("admin", "198.51.100."+strconv.Itoa(i))
	}
	if blocked, _ := rl.CheckAuthUser("admin", "192.0.2.50"); blocked {
		t.Fatalf("global bucket locked after %d failures, want %d", UserGlobalMaxAttempts-1, UserGlobalMaxAttempts)
	}
	rl.RecordAuthUser("admin", "192.0.2.99")
	if blocked, _ := rl.CheckAuthUser("admin", "192.0.2.50"); !blocked {
		t.Fatalf("global bucket must lock after %d failures", UserGlobalMaxAttempts)
	}
}

// A successful login clears the address bucket but leaves the global count,
// so a distributed attacker gains nothing from the victim logging in.
func TestAuthUser_ResetKeepsGlobalCount(t *testing.T) {
	rl := NewRateLimiter(t.TempDir())
	for i := 0; i < maxAttempts; i++ {
		rl.RecordAuthUser("admin", "203.0.113.1")
	}
	rl.ResetAuthUser("admin", "203.0.113.1")
	if blocked, _ := rl.CheckAuthUser("admin", "203.0.113.1"); blocked {
		t.Fatal("reset must unlock the address bucket")
	}
	for i := 0; i < UserGlobalMaxAttempts-maxAttempts; i++ {
		rl.RecordAuthUser("admin", "198.51.100.7")
	}
	if blocked, _ := rl.CheckAuthUser("admin", "192.0.2.50"); !blocked {
		t.Fatal("reset must not clear the global per-user count")
	}
}

// #1021: a client behind an appending proxy controls the leftmost entries; the
// bucket must come from the rightmost entry the trusted proxy wrote.
func TestExtractIP_SpoofedLeftmostXFFDoesNotChangeBucket(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	a := extractIP(reqWith("10.0.0.5:443", "6.6.6.6, 1.2.3.4"), trusted)
	b := extractIP(reqWith("10.0.0.5:443", "7.7.7.7, 1.2.3.4"), trusted)
	if a != "1.2.3.4" || b != "1.2.3.4" {
		t.Fatalf("spoofed leftmost XFF changed the bucket: %q, %q (want 1.2.3.4)", a, b)
	}
}

func TestExtractIP_SkipsTrustedHopsFromTheRight(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	r := reqWith("10.0.0.5:443", "6.6.6.6, 1.2.3.4, 10.0.0.9")
	if got := extractIP(r, trusted); got != "1.2.3.4" {
		t.Fatalf("want first untrusted hop 1.2.3.4, got %q", got)
	}
}

// A proxy may add its own header line instead of appending to the client's.
func TestExtractIP_ReadsEveryXFFHeaderLine(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	r := reqWith("10.0.0.5:443", "6.6.6.6")
	r.Header.Add("X-Forwarded-For", "1.2.3.4")
	if got := extractIP(r, trusted); got != "1.2.3.4" {
		t.Fatalf("want proxy-written 1.2.3.4, got %q", got)
	}
}

func TestExtractIP_MalformedRightmostEntryFallsBackToPeer(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8")
	r := reqWith("10.0.0.5:443", "1.2.3.4, not-an-ip")
	if got := extractIP(r, trusted); got != "10.0.0.5" {
		t.Fatalf("want peer 10.0.0.5 for a malformed entry, got %q", got)
	}
}

func TestExtractIP_WarnsOnceWhenXFFArrivesWithoutTrustedProxies(t *testing.T) {
	prevNets, prevOut, prevFlags := trustedProxies, log.Writer(), log.Flags()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	trustedProxies = nil
	warnIgnoredXFF = sync.Once{}
	t.Cleanup(func() {
		trustedProxies = prevNets
		log.SetOutput(prevOut)
		log.SetFlags(prevFlags)
		warnIgnoredXFF = sync.Once{}
	})

	ExtractIP(reqWith("203.0.113.7:5555", ""))
	if buf.Len() != 0 {
		t.Fatalf("no XFF must log nothing, got %q", buf.String())
	}
	ExtractIP(reqWith("203.0.113.7:5555", "1.2.3.4"))
	ExtractIP(reqWith("203.0.113.7:5555", "1.2.3.5"))
	if got := strings.Count(buf.String(), "WARN:"); got != 1 {
		t.Fatalf("want exactly one warning, got %d: %q", got, buf.String())
	}
}

func TestExtractIP_EdgeCases(t *testing.T) {
	trusted := mustCIDRs(t, "10.0.0.0/8", "fd00::/8")
	for _, tc := range []struct{ name, peer, xff, want string }{
		{"all trusted chain returns leftmost hop", "10.0.0.5:1", "10.0.0.7, 10.0.0.9", "10.0.0.7"},
		{"ipv6 client behind trusted proxy", "10.0.0.5:1", "6.6.6.6, 2001:db8::1", "2001:db8::1"},
		{"ipv6 trusted hop skipped", "10.0.0.5:1", "2001:db8::1, fd00::2", "2001:db8::1"},
		{"untrusted peer ignores xff", "192.0.2.1:1", "1.2.3.4", "192.0.2.1"},
		{"trailing comma falls back to peer", "10.0.0.5:1", "1.2.3.4,", "10.0.0.5"},
		{"empty xff returns peer", "10.0.0.5:1", "", "10.0.0.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractIP(reqWith(tc.peer, tc.xff), trusted); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
