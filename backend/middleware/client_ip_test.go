package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func realIP(t *testing.T, extractor echo.IPExtractor, remoteAddr, headerName, headerValue string) string {
	t.Helper()

	e := echo.New()
	e.IPExtractor = extractor
	var got string
	e.GET("/", func(ctx echo.Context) error {
		got = ctx.RealIP()
		return ctx.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if headerName != "" {
		req.Header.Set(headerName, headerValue)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	return got
}

func TestClientIPExtractor_IgnoresSpoofedHeadersWithoutTrustedProxy(t *testing.T) {
	extractor := NewClientIPExtractor(nil, "")
	got := realIP(t, extractor, "203.0.113.10:1234", "X-Forwarded-For", "1.2.3.4")
	if got != "203.0.113.10" {
		t.Fatalf("RealIP = %q, want remote address", got)
	}

	got = realIP(t, extractor, "203.0.113.10:1234", DefaultClientIPHeader, "1.2.3.4")
	if got != "203.0.113.10" {
		t.Fatalf("RealIP = %q, want remote address when Vercel header is spoofed", got)
	}
}

func TestClientIPExtractor_TrustsDedicatedHeaderFromVercelCIDR(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"76.76.21.0/24"}, DefaultClientIPHeader)
	got := realIP(t, extractor, "76.76.21.10:443", DefaultClientIPHeader, "198.51.100.20")
	if got != "198.51.100.20" {
		t.Fatalf("RealIP = %q, want client IP from dedicated header", got)
	}
}

func TestClientIPExtractor_RejectsXForwardedForAsHeaderName(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"76.76.21.0/24"}, "X-Forwarded-For")
	got := realIP(t, extractor, "76.76.21.10:443", "X-Forwarded-For", "1.2.3.4")
	if got != "76.76.21.10" {
		t.Fatalf("RealIP = %q, want remote address when X-Forwarded-For is requested", got)
	}

	got = realIP(t, extractor, "76.76.21.10:443", DefaultClientIPHeader, "198.51.100.20")
	if got != "198.51.100.20" {
		t.Fatalf("RealIP = %q, want dedicated Vercel header after XFF name is rejected", got)
	}
}

func TestClientIPExtractor_IgnoresXFFEvenFromTrustedProxy(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"76.76.21.0/24"}, DefaultClientIPHeader)
	got := realIP(t, extractor, "76.76.21.10:443", "X-Forwarded-For", "198.51.100.20")
	if got != "76.76.21.10" {
		t.Fatalf("RealIP = %q, want remote address when only XFF is set", got)
	}
}

func TestClientIPExtractor_IgnoresDedicatedHeaderFromUntrustedPeer(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"76.76.21.0/24"}, DefaultClientIPHeader)
	got := realIP(t, extractor, "203.0.113.10:1234", DefaultClientIPHeader, "198.51.100.20")
	if got != "203.0.113.10" {
		t.Fatalf("RealIP = %q, want remote address from untrusted peer", got)
	}
}

func TestClientIPExtractor_InvalidHeaderFallsBackToRemote(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"76.76.21.0/24"}, DefaultClientIPHeader)
	got := realIP(t, extractor, "76.76.21.10:443", DefaultClientIPHeader, "not-an-ip, still-bad")
	if got != "76.76.21.10" {
		t.Fatalf("RealIP = %q, want remote address", got)
	}
}

func TestClientIPExtractor_TakesFirstValidIPInHeader(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"76.76.21.0/24"}, DefaultClientIPHeader)
	got := realIP(t, extractor, "76.76.21.10:443", DefaultClientIPHeader, "not-an-ip, 198.51.100.20, 192.0.2.1")
	if got != "198.51.100.20" {
		t.Fatalf("RealIP = %q", got)
	}
}

func TestClientIPExtractor_IPv6TrustedProxy(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"2001:db8::/32"}, "X-Real-IP")
	got := realIP(t, extractor, "[2001:db8::1]:443", "X-Real-IP", "2001:db8:aaaa::10")
	if got != "2001:db8:aaaa::10" {
		t.Fatalf("RealIP = %q", got)
	}
}

func TestClientIPExtractor_RemoteAddrWithoutPort(t *testing.T) {
	extractor := NewClientIPExtractor(nil, "")
	got := realIP(t, extractor, "203.0.113.10", "", "")
	if got != "203.0.113.10" {
		t.Fatalf("RealIP = %q", got)
	}
}

func TestClientIPExtractor_SkipsInvalidCIDREntries(t *testing.T) {
	extractor := NewClientIPExtractor([]string{"not-a-cidr", "76.76.21.0/24"}, DefaultClientIPHeader)
	got := realIP(t, extractor, "76.76.21.10:443", DefaultClientIPHeader, "198.51.100.20")
	if got != "198.51.100.20" {
		t.Fatalf("RealIP = %q", got)
	}
}

func TestParseTrustedProxyCIDRsAndHelpers(t *testing.T) {
	if got := firstValidForwardedIP(""); got != "" {
		t.Fatalf("empty header = %q", got)
	}
	if got := remoteAddrIP("[::1]:8080"); got != "::1" {
		t.Fatalf("ipv6 remote = %q", got)
	}
}
