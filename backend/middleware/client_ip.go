package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

const DefaultClientIPHeader = "X-Vercel-Forwarded-For"

func NewClientIPExtractor(trustedCIDRs []string, headerName string) echo.IPExtractor {
	networks := parseTrustedProxyCIDRs(trustedCIDRs)

	// Client IPとして利用するヘッダーはVercelのものに固定する。
	// 呼び出し側から任意のForwarded系ヘッダーを指定できないようにする。
	headerName = DefaultClientIPHeader

	return func(req *http.Request) string {
		remote := remoteAddrIP(req.RemoteAddr)

		// Trusted Proxyが設定されていない場合、
		// Forwarded系ヘッダーは一切信用しない。
		if len(networks) == 0 {
			return remote
		}

		peer := net.ParseIP(remote)
		if peer == nil {
			return remote
		}

		// 直接接続してきた相手がTrusted Proxyではない場合、
		// クライアントIPヘッダーを信用しない。
		if !ipInTrustedProxies(peer, networks) {
			return remote
		}

		// Trusted Proxyからの接続である場合のみ、
		// Vercel専用のクライアントIPヘッダーを利用する。
		if forwarded := firstValidForwardedIP(
			req.Header.Get(headerName),
		); forwarded != "" {
			return forwarded
		}

		return remote
	}
}

func parseTrustedProxyCIDRs(cidrs []string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(cidrs))

	for _, cidr := range cidrs {
		cidr = strings.TrimSpace(cidr)

		if cidr == "" {
			continue
		}

		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}

		networks = append(networks, network)
	}

	return networks
}

func ipInTrustedProxies(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return strings.Trim(remoteAddr, "[]")
	}

	return strings.Trim(host, "[]")
}

func firstValidForwardedIP(header string) string {
	for _, part := range strings.Split(header, ",") {
		candidate := strings.TrimSpace(part)
		candidate = strings.Trim(candidate, "[]")

		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}

	return ""
}
