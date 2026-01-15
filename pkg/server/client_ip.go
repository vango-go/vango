package server

import (
	"net"
	"net/http"
	"strings"
)

func (s *Server) clientIP(r *http.Request) string {
	ip := clientIPFromRequest(r, s.trustedProxies)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func clientIPFromRequest(r *http.Request, trusted *proxyMatcher) net.IP {
	remoteIP := remoteIPFromRequest(r)
	if remoteIP == nil {
		return nil
	}
	if trusted == nil || !trusted.IsTrusted(remoteIP) {
		return remoteIP
	}

	forwarded := parseForwardedFor(r.Header.Get("Forwarded"))
	if len(forwarded) == 0 {
		forwarded = parseXForwardedFor(r.Header.Get("X-Forwarded-For"))
	}
	if len(forwarded) == 0 {
		return remoteIP
	}

	var candidates []net.IP
	for _, ip := range forwarded {
		if ip != nil {
			candidates = append(candidates, ip)
		}
	}
	if len(candidates) == 0 {
		return remoteIP
	}

	for i := len(candidates) - 1; i >= 0; i-- {
		if !trusted.IsTrusted(candidates[i]) {
			return candidates[i]
		}
	}

	return candidates[0]
}

func parseForwardedFor(header string) []net.IP {
	if header == "" {
		return nil
	}

	var out []net.IP
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		params := strings.Split(part, ";")
		for _, param := range params {
			param = strings.TrimSpace(param)
			if param == "" {
				continue
			}
			kv := strings.SplitN(param, "=", 2)
			if len(kv) != 2 {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(kv[0]), "for") {
				continue
			}
			ip := parseForwardedIP(strings.TrimSpace(kv[1]))
			if ip != nil {
				out = append(out, ip)
			}
		}
	}
	return out
}

func parseXForwardedFor(header string) []net.IP {
	if header == "" {
		return nil
	}

	var out []net.IP
	parts := strings.Split(header, ",")
	for _, part := range parts {
		ip := parseForwardedIP(part)
		if ip != nil {
			out = append(out, ip)
		}
	}
	return out
}

func parseForwardedIP(value string) net.IP {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")
	if value == "" || strings.EqualFold(value, "unknown") {
		return nil
	}

	host := value
	if strings.HasPrefix(host, "[") {
		if end := strings.Index(host, "]"); end != -1 {
			host = host[1:end]
		}
	} else if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	} else if strings.Count(host, ":") > 1 {
		host = strings.Trim(host, "[]")
	}

	if zone := strings.Index(host, "%"); zone != -1 {
		host = host[:zone]
	}
	return net.ParseIP(host)
}
