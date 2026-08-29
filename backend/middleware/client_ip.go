package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/koezuka404/notehub/config"
	"github.com/labstack/echo/v4"
)

const DefaultClientIPHeader = config.DefaultClientIPHeader

func NewClientIPExtractor(
	trustedCIDRs []string,
	headerName string,
) echo.IPExtractor {
	networks := parseTrustedProxyCIDRs(trustedCIDRs)

	headerName = strings.TrimSpace(headerName)
	if headerName == "" || strings.EqualFold(headerName, "X-Forwarded-For") {
		headerName = DefaultClientIPHeader
	}

	return func(req *http.Request) string {
		remote := remoteAddrIP(req.RemoteAddr)

		if len(networks) == 0 {
			return remote
		}

		peer := net.ParseIP(remote)
		if peer == nil {
			return remote
		}

		if !ipInTrustedProxies(peer, networks) {
			return remote
		}

		forwarded := firstValidForwardedIP(
			req.Header.Get(headerName),
		)

		if forwarded != "" {
			return forwarded
		}

		return remote
	}
}

func parseTrustedProxyCIDRs(
	cidrs []string,
) []*net.IPNet {
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

func ipInTrustedProxies(
	ip net.IP,
	networks []*net.IPNet,
) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func remoteAddrIP(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return strings.Trim(host, "[]")
	}

	return strings.Trim(remoteAddr, "[]")
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
