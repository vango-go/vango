package server

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

func (s *Server) csrfCookie(r *http.Request, token string) (*http.Cookie, error) {
	secure, err := s.cookieSecureFlag(r)
	if err != nil {
		return nil, err
	}

	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Must be readable by JS for Double Submit
		SameSite: s.config.SameSiteMode,
		Secure:   secure,
	}
	if s.config.CookieDomain != "" {
		cookie.Domain = s.config.CookieDomain
	}

	return cookie, nil
}

func (s *Server) cookieSecureFlag(r *http.Request) (bool, error) {
	if !s.config.SecureCookies {
		return false, nil
	}
	if s.isRequestSecure(r) {
		return true, nil
	}
	return false, ErrSecureCookiesRequired
}

func (s *Server) isRequestSecure(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	if !s.isTrustedProxy(r) {
		return false
	}

	if proto := forwardedProto(r.Header.Get("Forwarded")); proto != "" {
		return isSecureProto(proto)
	}
	if proto := forwardedProtoValue(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		return isSecureProto(proto)
	}

	return false
}

func (s *Server) isTrustedProxy(r *http.Request) bool {
	if s.trustedProxies == nil {
		return false
	}
	return s.trustedProxies.IsTrusted(remoteIPFromRequest(r))
}

func forwardedProto(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	first := strings.TrimSpace(parts[0])
	if first == "" {
		return ""
	}
	for _, param := range strings.Split(first, ";") {
		param = strings.TrimSpace(param)
		if param == "" {
			continue
		}
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			continue
		}
		if strings.EqualFold(kv[0], "proto") {
			value := strings.TrimSpace(kv[1])
			value = strings.Trim(value, "\"")
			return strings.ToLower(value)
		}
	}
	return ""
}

func forwardedProtoValue(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	value := strings.TrimSpace(parts[0])
	value = strings.Trim(value, "\"")
	return strings.ToLower(value)
}

func isSecureProto(proto string) bool {
	switch strings.ToLower(proto) {
	case "https", "wss":
		return true
	default:
		return false
	}
}

func remoteIPFromRequest(r *http.Request) net.IP {
	if r == nil {
		return nil
	}
	host := strings.TrimSpace(r.RemoteAddr)
	if host == "" {
		return nil
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if zone := strings.Index(host, "%"); zone != -1 {
		host = host[:zone]
	}
	return net.ParseIP(host)
}

type proxyMatcher struct {
	ips  map[string]struct{}
	nets []*net.IPNet
}

func newProxyMatcher(entries []string, logger *slog.Logger) *proxyMatcher {
	if len(entries) == 0 {
		return nil
	}

	ips := make(map[string]struct{})
	var nets []*net.IPNet

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "/") {
			_, network, err := net.ParseCIDR(entry)
			if err != nil {
				if logger != nil {
					logger.Warn("invalid trusted proxy CIDR", "entry", entry, "error", err)
				}
				continue
			}
			nets = append(nets, network)
			continue
		}
		ip := net.ParseIP(entry)
		if ip == nil {
			if logger != nil {
				logger.Warn("invalid trusted proxy IP", "entry", entry)
			}
			continue
		}
		ips[ip.String()] = struct{}{}
	}

	if len(ips) == 0 && len(nets) == 0 {
		return nil
	}

	return &proxyMatcher{ips: ips, nets: nets}
}

func (m *proxyMatcher) IsTrusted(ip net.IP) bool {
	if m == nil || ip == nil {
		return false
	}
	if len(m.ips) > 0 {
		if _, ok := m.ips[ip.String()]; ok {
			return true
		}
	}
	for _, network := range m.nets {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
